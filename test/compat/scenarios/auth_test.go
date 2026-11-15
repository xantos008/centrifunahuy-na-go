package scenarios

import (
	"fmt"
	"testing"
	"time"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func connect(s *h.Session, token string) *h.WS {
	ws := s.Dial()
	ws.Send(fmt.Sprintf(`{"id":1,"connect":{"token":%q}}`, token))
	ws.Next()
	return ws
}

// CS-AUTH-01: valid connection tokens for every supported algorithm family.
func TestCS_AUTH_01(t *testing.T) {
	h.Run(t, "CS-AUTH-01", func(s *h.Session) {
		c := h.Claims{Sub: "u1", Info: &h.Info{Name: "compat"}}
		for _, tok := range []string{h.HS256(c), h.RS256(c), h.ES256(c), h.EdDSA(c)} {
			connect(s, tok).Close()
		}
	})
}

// CS-AUTH-02: rejected connection tokens.
func TestCS_AUTH_02(t *testing.T) {
	h.Run(t, "CS-AUTH-02", func(s *h.Session) {
		c := h.Claims{Sub: "u1"}
		for _, tok := range []string{
			h.HS256(h.Claims{Sub: "u1", Exp: h.ExpIn(-time.Hour)}), // expired
			h.HS256WithSecret(c, "wrong-secret"),                   // bad signature
			h.AlgNone(c),                                           // alg=none
			h.AlgConfusion(c),                                      // HS256 keyed with the RSA public key
			"not-a-jwt",                                            // garbage
			"",                                                     // no token, anonymous not allowed
		} {
			connect(s, tok)
		}
	})
}
