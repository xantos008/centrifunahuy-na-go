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

func TestCS_CONS_01(t *testing.T) {
	h.Profiles(t, "r-opt")
	h.Run(t, "CS-CONS-01", func(s *h.Session) {
		const commands, publications, group = "compat_commands", "compat_publications", "compat_dispatch"
		redis := func(args ...string) string {
			s.T.Helper()
			cmd := []string{"redis-cli", "--raw", "--no-auth-warning", "--user", "compat", "--pass", "compat-test-redis-password"}
			return strings.TrimSpace(s.Exec("redis", append(cmd, args...)...))
		}
		wait := func(description string, probe func() string, done func(string) bool) {
			s.T.Helper()
			deadline := time.Now().Add(20 * time.Second)
			for {
				got := probe()
				if done(got) {
					return
				}
				if time.Now().After(deadline) {
					s.T.Fatalf("waiting for %s: last Redis response %q", description, got)
				}
				time.Sleep(150 * time.Millisecond)
			}
		}
		for _, stream := range []string{commands, publications} {
			wait("consumer group on "+stream, func() string { return redis("XINFO", "GROUPS", stream) },
				func(out string) bool { return slices.Contains(strings.Fields(out), group) })
		}

		channel := s.Ch("hist:consumed")
		ws := connect(s, h.HS256(h.Claims{Sub: "consumer-reader"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, channel))
		ws.Recv(1)
		var sub protocol.Reply
		ws.Last(&sub)
		if sub.Subscribe == nil {
			s.T.Fatalf("subscriber setup: %v", &sub)
		}
		enqueue := func(stream string, fields ...string) string {
			s.T.Helper()
			id := redis(append([]string{"XADD", stream, "*"}, fields...)...)
			ms, seq, ok := strings.Cut(id, "-")
			_, msErr := strconv.ParseUint(ms, 10, 64)
			_, seqErr := strconv.ParseUint(seq, 10, 64)
			if !ok || msErr != nil || seqErr != nil {
				s.T.Fatalf("XADD %s: %q", stream, id)
			}
			return id
		}
		for i, tc := range []struct {
			stream, label string
			fields        []string
		}{
			{commands, "method field", []string{"method", "publish", "payload", fmt.Sprintf(`{"channel":%q,"data":{"n":1}}`, channel)}},
			{commands, "command envelope", []string{"payload", fmt.Sprintf(`{"method":"publish","payload":{"channel":%q,"data":{"n":2}}}`, channel)}},
			{publications, "publication data", []string{"channels", channel, "tag-source", "queue", "payload", `{"n":3}`}},
		} {
			id := enqueue(tc.stream, tc.fields...)
			ws.Recv(1)
			var delivery protocol.Reply
			ws.Last(&delivery)
			push := delivery.GetPush()
			pub := push.GetPub()
			if push.GetChannel() != channel || pub.GetOffset() != uint64(i+1) || string(pub.GetData()) != fmt.Sprintf(`{"n":%d}`, i+1) {
				s.T.Fatalf("%s: unexpected delivery: %v", tc.label, &delivery)
			}
			if tc.stream == publications && pub.GetTags()["source"] != "queue" {
				s.T.Fatalf("missing consumer publication tag: %v", &delivery)
			}
			// Receipt proves this entry was read. It leaves the pending list after XACK.
			wait("ack for "+tc.label, func() string { return redis("XPENDING", tc.stream, group, id, id, "1") },
				func(out string) bool { return out == "" })
			s.Note("consumer acknowledged " + tc.label)
		}
		ws.Close()

		// An invalid boolean cannot be dispatched. Observe Redis redelivering
		// the same pending ID, then remove only this test entry.
		id := enqueue(publications, "channels", channel, "delta", "invalid", "payload", `{"n":4}`)
		defer func() {
			redis("XACK", publications, group, id)
			redis("XDEL", publications, id)
		}()
		wait("redelivery of the pending entry", func() string { return redis("XPENDING", publications, group, id, id, "1") },
			func(out string) bool {
				fields := strings.Fields(out)
				if len(fields) != 4 || fields[0] != id {
					return false
				}
				count, err := strconv.Atoi(fields[3])
				return err == nil && count >= 2
			})
		s.Note("failed publication pending after redelivery")
	})
}
