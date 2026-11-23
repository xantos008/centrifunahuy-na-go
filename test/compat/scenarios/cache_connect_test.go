package scenarios

import (
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// The token subscribes the client server-side. Neither transport sends subs,
// recover or a previous position in the connect request.
func TestCS_REC_06(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-REC-06", func(s *h.Session) {
		channel := s.Ch("acache:connect")
		for n := 1; n <= 2; n++ {
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":%d}}`, channel, n)})
		}
		token := h.HS256(h.Claims{Sub: "u1", Channels: []string{channel}})
		check := func(sub *protocol.SubscribeResult) {
			s.T.Helper()
			if sub == nil || !sub.WasRecovering || len(sub.Publications) != 1 {
				s.T.Fatalf("expected a single cached publication for %s: %v", channel, sub)
			}
			pub := sub.Publications[0]
			if string(pub.Data) != `{"n":2}` || pub.Offset != 2 {
				s.T.Fatalf("unexpected cached publication: %v", pub)
			}
		}
		ws := connect(s, token)
		var reply protocol.Reply
		ws.Last(&reply)
		check(reply.GetConnect().GetSubs()[channel])
		ws.Close()

		next := s.OpenHTTPStream("/connection/uni_http_stream", fmt.Sprintf(`{"token":%q}`, token), false)
		check(next().GetConnect().GetSubs()[channel])
	})
}
