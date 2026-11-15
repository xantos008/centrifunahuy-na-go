package scenarios

import (
	"fmt"
	"testing"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// CS-PUB-01: client publish permissions and server API publish options.
// Publisher and subscriber are separate connections, so reply and push order
// never interleave on one socket.
func TestCS_PUB_01(t *testing.T) {
	h.Profiles(t, "m", "r", "rs", "rc", "rsh", "n", "p")
	h.Run(t, "CS-PUB-01", func(s *h.Session) {
		ch := s.Ch("pub:p1")
		sub := connect(s, h.HS256(h.Claims{Sub: "u2"}))
		sub.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
		sub.Recv(1)

		pub := connect(s, h.HS256(h.Claims{Sub: "u1"}))
		pub.Send(fmt.Sprintf(`{"id":2,"publish":{"channel":%q,"data":{"n":1}}}`, ch))
		pub.Recv(1)
		sub.Recv(1)
		pub.Send(fmt.Sprintf(`{"id":3,"publish":{"channel":%q,"data":{"n":2}}}`, s.Ch("news"))) // no publish permission
		pub.Next()

		hist := s.Ch("hist:p1")
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":3},"tags":{"k":"v"},"idempotency_key":"ik-1"}`, hist)})
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":3},"tags":{"k":"v"},"idempotency_key":"ik-1"}`, hist)})
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":4},"skip_history":true}`, hist)})
		s.API(h.APIRequest{Path: "/api/history", Body: fmt.Sprintf(`{"channel":%q,"limit":10}`, hist)})
	})
}
