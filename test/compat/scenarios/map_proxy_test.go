package scenarios

import (
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// The proxy receives base64 data over either transport, can reject a client
// write, and controls the key used both to publish and to remove the value.
func TestCS_MAP_02(t *testing.T) {
	h.Profiles(t, "r", "p")
	h.Run(t, "CS-MAP-02", func(s *h.Session) {
		ch := s.Ch("mapproxy:callbacks")
		reader := connect(s, h.HS256(h.Claims{Sub: "map-reader"}))
		writer := connect(s, h.HS256(h.Claims{Sub: "map-writer", Meta: &h.Info{Name: "map-meta"}}))
		reader.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"type":1,"phase":2}}`, ch))
		reader.Recv(1)
		var reply protocol.Reply
		reader.Last(&reply)
		if reply.GetSubscribe().GetType() != 1 || len(reply.Subscribe.State) != 0 {
			s.T.Fatalf("initial map subscription: %v", &reply)
		}
		write := func(id int, key string, remove, allow bool) {
			s.T.Helper()
			body := `"data":"client"`
			if remove {
				body = `"removed":true`
			}
			writer.Send(fmt.Sprintf(`{"id":%d,"publish":{"channel":%q,"type":1,"key":%q,%s}}`, id, ch, key, body))
			writer.Recv(1)
			var ack protocol.Reply
			writer.Last(&ack)
			if int(ack.Id) != id {
				s.T.Fatalf("unexpected reply id: %v", &ack)
			}
			if !allow {
				if ack.GetError().GetCode() != 470 {
					s.T.Fatalf("callback rejection: %v", &ack)
				}
				return
			}
			// On v6.10.3 the handler applies the write, but its reply omits
			// the key required by the client library. Verify both wire effects.
			if ack.GetError().GetCode() != 107 || ack.GetError().GetMessage() != "bad request" {
				s.T.Fatalf("unexpected reply for an applied map write: %v", &ack)
			}
			reader.Recv(1)
			var delivered protocol.Reply
			reader.Last(&delivered)
			push := delivered.GetPush()
			pub := push.GetPub()
			if push.GetChannel() != ch || pub.GetKey() != "stored" || pub.GetRemoved() != remove {
				s.T.Fatalf("unexpected map delivery: %v", &delivered)
			}
			if remove {
				if pub.GetOffset() != 2 || pub.GetTags()["action"] != "remove" {
					s.T.Fatalf("unexpected removal: %v", pub)
				}
			} else if pub.GetOffset() != 1 || string(pub.GetData()) != `"callback"` || pub.GetTags()["action"] != "put" {
				s.T.Fatalf("unexpected publication: %v", pub)
			}
		}
		write(2, "denied", false, false)
		write(3, "input", false, true)
		write(4, "denied", true, false)
		write(5, "input", true, true)
	})
}
