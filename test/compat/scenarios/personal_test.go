package scenarios

import (
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_SUB_06(t *testing.T) {
	h.Profiles(t, "r")
	h.Run(t, "CS-SUB-06", func(s *h.Session) {
		restore := s.ReplaceB("r-personal")
		defer func() {
			restore()
			startB(s)
		}()

		login := func(user string) *h.WS {
			s.T.Helper()
			var ws *h.WS
			s.OnB(func() {
				ws = connect(s, h.HS256(h.Claims{Sub: user}))
				var reply protocol.Reply
				ws.Last(&reply)
				subs := reply.GetConnect().GetSubs()
				if reply.Id != 1 || len(subs) != 1 || subs["#"+user] == nil {
					s.T.Fatalf("expected automatic personal subscription for %s, got %v", user, &reply)
				}
			})
			return ws
		}
		publish := func(user, data string) {
			s.T.Helper()
			body := fmt.Sprintf(`{"channel":%q,"data":%q}`, "#"+user, data)
			result := s.API(h.APIRequest{Path: "/api/publish", Body: body})
			if string(result) != `{"result":{}}` {
				s.T.Fatalf("personal publish for %s returned %s", user, result)
			}
		}
		receive := func(ws *h.WS, user, data string) {
			s.T.Helper()
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			if reply.Id != 0 || reply.GetPush().GetChannel() != "#"+user ||
				string(reply.GetPush().GetPub().GetData()) != fmt.Sprintf("%q", data) {
				s.T.Fatalf("expected %q on %s's personal channel, got %v", data, user, &reply)
			}
		}

		alice, bob := s.Ch("personal-alice"), s.Ch("personal-bob")
		first := login(alice)
		defer first.Close()
		other := login(bob)
		defer other.Close()
		other.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, "#"+alice))
		other.Recv(1)
		var denied protocol.Reply
		other.Last(&denied)
		if denied.Id != 2 || denied.GetError().GetCode() != 103 {
			s.T.Fatalf("other user subscribing to personal channel got %v", &denied)
		}
		publish(alice, "first")
		publish(bob, "second")
		receive(first, alice, "first")
		receive(other, bob, "second")

		replacement := login(alice)
		defer replacement.Close()
		if code := first.ExpectClose(); code != 3504 {
			s.T.Fatalf("previous personal connection closed with %d, want 3504", code)
		}
		publish(alice, "replacement")
		publish(bob, "still connected")
		receive(replacement, alice, "replacement")
		receive(other, bob, "still connected")
	})
}
