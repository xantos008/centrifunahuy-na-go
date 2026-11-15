package scenarios

import (
	"fmt"
	"testing"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// CS-PRES-01: presence and presence_stats via client and API.
func TestCS_PRES_01(t *testing.T) {
	h.Profiles(t, "m", "r", "r-opt", "rs", "rc", "rsh", "n")
	h.Run(t, "CS-PRES-01", func(s *h.Session) {
		ch := s.Ch("presence")
		a := connect(s, h.HS256(h.Claims{Sub: "u1", Info: &h.Info{Name: "a"}}))
		a.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
		a.Recv(1)
		b := connect(s, h.HS256(h.Claims{Sub: "u2", Info: &h.Info{Name: "b"}}))
		b.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
		b.Recv(1)
		a.Send(fmt.Sprintf(`{"id":3,"presence":{"channel":%q}}`, ch))
		a.Recv(1)
		a.Send(fmt.Sprintf(`{"id":4,"presence_stats":{"channel":%q}}`, ch))
		a.Recv(1)
		s.API(h.APIRequest{Path: "/api/presence", Body: fmt.Sprintf(`{"channel":%q}`, ch)})
		s.API(h.APIRequest{Path: "/api/presence_stats", Body: fmt.Sprintf(`{"channel":%q}`, ch)})
		a.Send(fmt.Sprintf(`{"id":5,"presence":{"channel":%q}}`, s.Ch("pub:nopresence"))) // presence disabled
		a.Next()
	})
}
