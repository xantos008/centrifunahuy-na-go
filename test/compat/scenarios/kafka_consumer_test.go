package scenarios

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_CONS_06(t *testing.T) {
	h.Profiles(t, "kafka")
	h.Run(t, "CS-CONS-06", func(s *h.Session) {
		const commands, publications = "compat-commands", "compat-publications"
		kafka := func(tool string, args ...string) string {
			s.T.Helper()
			return s.Exec("kafka", append([]string{tool, "--bootstrap-server", "kafka:29092"}, args...)...)
		}
		// Track advances from this run's log end so diff, record and golden can
		// execute repeatedly on the same brokers with existing committed records.
		start := map[string]int64{}
		for _, topic := range []string{commands, publications} {
			kafka("kafka-topics", "--create", "--if-not-exists", "--topic", topic, "--partitions", "1", "--replication-factor", "1")
			out := kafka("kafka-get-offsets", "--topic", topic, "--time", "latest")
			prefix := topic + ":0:"
			for _, line := range strings.Split(out, "\n") {
				if n, ok := strings.CutPrefix(strings.TrimSpace(line), prefix); ok {
					pos, err := strconv.ParseInt(n, 10, 64)
					if err == nil && pos >= 0 {
						start[topic] = pos
					}
				}
			}
			if _, ok := start[topic]; !ok {
				s.T.Fatalf("no partition 0 log end for %s: %s", topic, out)
			}
			// The consumers may have polled metadata before their topics existed.
			// Wait for an assigned member before starting the WebSocket deadline.
			group := topic + "-group"
			deadline := time.Now().Add(45 * time.Second)
			for {
				groups := kafka("kafka-consumer-groups", "--list")
				if slices.Contains(strings.Fields(groups), group) {
					out = kafka("kafka-consumer-groups", "--describe", "--group", group, "--members", "--verbose")
					if slices.Contains(strings.Fields(out), topic+":0") {
						break
					}
				}
				if time.Now().After(deadline) {
					s.T.Fatalf("waiting for %s to own partition 0 of %s: groups %s, state %s", group, topic, groups, out)
				}
				time.Sleep(200 * time.Millisecond)
			}
		}

		channel := s.Ch("hist:kafka")
		ws := connect(s, h.HS256(h.Claims{Sub: "kafka-reader"}))
		defer ws.Close()
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, channel))
		ws.Recv(1)
		var sub protocol.Reply
		ws.Last(&sub)
		if sub.GetSubscribe() == nil {
			s.T.Fatalf("subscribe: %v", &sub)
		}
		record := func(n int) string {
			return fmt.Sprintf(`{"method":"publish","payload":{"channel":%q,"data":{"n":%d}}}`, channel, n)
		}
		// Supply record values as positional arguments, never as shell source.
		s.Exec("kafka", "bash", "-ec",
			`printf '%s\n' "$@" | kafka-console-producer --bootstrap-server kafka:29092 --topic compat-commands`,
			"producer", record(1), record(2))
		delivery := func(n int) {
			s.T.Helper()
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			push := reply.GetPush()
			pub := push.GetPub()
			if push.GetChannel() != channel || pub.GetOffset() != uint64(n) || string(pub.GetData()) != fmt.Sprintf(`{"n":%d}`, n) {
				s.T.Fatalf("unexpected Kafka delivery %d: %v", n, &reply)
			}
			if n == 3 && pub.GetTags()["source"] != "queue" {
				s.T.Fatalf("publication header did not produce a tag: %v", &reply)
			}
		}
		delivery(1)
		delivery(2)
		s.Exec("kafka", "bash", "-ec",
			`printf '%s\n' "$1" | kafka-console-producer --bootstrap-server kafka:29092 --topic compat-publications --property parse.headers=true --property 'headers.delimiter=|' --property 'headers.key.separator=='`,
			"producer", "centrifugo-channels="+channel+",centrifugo-tag-source=queue|"+`{"n":3}`)
		delivery(3)

		for _, topic := range []string{commands, publications} {
			count := int64(1)
			if topic == commands {
				count = 2
			}
			group := topic + "-group"
			want := strconv.FormatInt(start[topic]+count, 10)
			deadline := time.Now().Add(30 * time.Second)
			for {
				out := kafka("kafka-consumer-groups", "--describe", "--group", group)
				committed := false
				for _, line := range strings.Split(out, "\n") {
					f := strings.Fields(line)
					if len(f) >= 6 && f[0] == group && f[1] == topic && f[2] == "0" && f[3] == want && f[4] == want && f[5] == "0" {
						committed = true
						break
					}
				}
				if committed {
					s.Note(fmt.Sprintf("%s advanced partition 0 by %d with zero lag", group, count))
					break
				}
				if time.Now().After(deadline) {
					s.T.Fatalf("expected %s committed offset and log end %s, lag zero: %s", group, want, out)
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
	})
}
