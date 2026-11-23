package scenarios

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_CONS_02(t *testing.T) {
	h.Profiles(t, "p-notify")
	h.Run(t, "CS-CONS-02", func(s *h.Session) {
		channel := s.Ch("hist:outbox-consumer")
		ws := connect(s, h.HS256(h.Claims{Sub: "outbox-reader"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, channel))
		ws.Recv(1)
		var sub protocol.Reply
		ws.Last(&sub)
		if sub.Subscribe == nil {
			s.T.Fatalf("subscriber setup: %v", &sub)
		}
		insert := func(method string, n int) string {
			return fmt.Sprintf(`INSERT INTO compat_consumer_outbox (method,payload) VALUES ('%s','{"channel":%q,"data":{"n":%d}}')`, method, channel, n)
		}
		s.SQL("BEGIN", insert("publish", 99), "ROLLBACK")
		s.SQL("BEGIN", insert("publish", 1), insert("unknown", 77), insert("publish", 2), "COMMIT")

		check := func(pub *protocol.Publication, n int) {
			s.T.Helper()
			var data struct{ N int }
			if pub.GetOffset() != uint64(n) || json.Unmarshal(pub.GetData(), &data) != nil || data.N != n {
				s.T.Fatalf("expected publication %d: %v", n, pub)
			}
		}
		for n := 1; n <= 2; n++ {
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			if reply.GetPush().GetChannel() != channel {
				s.T.Fatalf("unexpected delivery: %v", &reply)
			}
			check(reply.GetPush().GetPub(), n)
		}
		ws.Close()
		// Dispatch precedes the consumer's DELETE and COMMIT. Wait for the
		// acknowledgement of this run's rows before inspecting final history.
		query := fmt.Sprintf("SELECT count(*) FROM compat_consumer_outbox WHERE payload->>'channel' = '%s'", channel)
		deadline := time.Now().Add(h.Deadline)
		for {
			count := strings.TrimSpace(s.Exec("postgres", "psql", "-XAt", "-v", "ON_ERROR_STOP=1", "-U", "compat", "-d", "compat", "-c", query))
			if count == "0" {
				break
			}
			if time.Now().After(deadline) {
				s.T.Fatalf("outbox still contains %s rows for %s", count, channel)
			}
			time.Sleep(100 * time.Millisecond)
		}
		s.Note("outbox rows acknowledged")
		body := s.API(h.APIRequest{Path: "/api/history", Body: fmt.Sprintf(`{"channel":%q,"limit":10}`, channel)})
		var history struct {
			Result struct{ Publications []*protocol.Publication }
		}
		if err := json.Unmarshal(body, &history); err != nil || len(history.Result.Publications) != 2 {
			s.T.Fatalf("unexpected outbox history: %s", body)
		}
		for i, pub := range history.Result.Publications {
			check(pub, i+1)
		}
	})
}
