package scenarios

import (
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_PRES_03(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-PRES-03", func(s *h.Session) {
		ch := s.Ch("presence-permissions")
		ws := connect(s, h.HS256(h.Claims{Sub: "u1", Info: &h.Info{Name: "connection"}}))
		for i, method := range []string{"presence", "presence_stats"} {
			ws.Send(fmt.Sprintf(`{"id":%d,%q:{"channel":%q}}`, i+2, method, ch))
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			if reply.Error.GetCode() != 103 {
				s.T.Fatalf("%s before subscribing: got %v", method, &reply)
			}
		}
		token := h.HS256(h.Claims{Sub: "u1", Channel: ch, Info: &h.Info{Name: "subscription"}})
		ws.Send(fmt.Sprintf(`{"id":4,"subscribe":{"channel":%q,"token":%q}}`, ch, token))
		ws.Recv(1)
		ws.Send(fmt.Sprintf(`{"id":5,"presence":{"channel":%q}}`, ch))
		ws.Recv(1)
		var reply protocol.Reply
		ws.Last(&reply)
		members := reply.GetPresence().GetPresence()
		if len(members) != 1 {
			s.T.Fatalf("expected one subscriber: %v", &reply)
		}
		for _, member := range members {
			if member.User != "u1" || string(member.ConnInfo) != `{"name":"connection"}` || string(member.ChanInfo) != `{"name":"subscription"}` {
				s.T.Fatalf("unexpected presence entry: %v", member)
			}
		}
	})
}
