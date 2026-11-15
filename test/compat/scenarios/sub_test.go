package scenarios

import (
	"fmt"
	"testing"
	"time"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// subscribeFresh connects as user and sends one subscribe request on a fresh
// connection, recording the first reaction (reply or disconnect).
func subscribeFresh(s *h.Session, user, req string) {
	ws := connect(s, h.HS256(h.Claims{Sub: user}))
	ws.Send(`{"id":2,"subscribe":` + req + `}`)
	ws.Next()
}

// CS-SUB-01: namespace permission matrix.
func TestCS_SUB_01(t *testing.T) {
	h.Run(t, "CS-SUB-01", func(s *h.Session) {
		news := s.Ch("news")
		for _, ch := range []string{
			news,              // allowed: allow_subscribe_for_client
			s.Ch("closed:c"),  // namespace without client permission
			s.Ch("unknown:c"), // unknown namespace
			news + "#u1",      // user-limited channel, own user
			news + "#u2",      // user-limited channel, other user
			"",                // empty channel
		} {
			subscribeFresh(s, "u1", fmt.Sprintf(`{"channel":%q}`, ch))
		}
		// Subscribing twice on one connection.
		ws := connect(s, h.HS256(h.Claims{Sub: "u1"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, news))
		ws.Next()
		ws.Send(fmt.Sprintf(`{"id":3,"subscribe":{"channel":%q}}`, news))
		ws.Next()
	})
}

// CS-SUB-02: subscription tokens for a namespace that requires them.
func TestCS_SUB_02(t *testing.T) {
	h.Run(t, "CS-SUB-02", func(s *h.Session) {
		ch := s.Ch("closed:tok")
		for _, tok := range []string{
			h.HS256(h.Claims{Sub: "u1", Channel: ch}),                           // valid
			h.HS256(h.Claims{Sub: "u1", Channel: ch, Exp: h.ExpIn(-time.Hour)}), // expired
			h.HS256(h.Claims{Sub: "u1", Channel: ch + "-other"}),                // other channel
			h.HS256(h.Claims{Sub: "u2", Channel: ch}),                           // other user
			h.HS256WithSecret(h.Claims{Sub: "u1", Channel: ch}, "wrong-secret"), // bad signature
		} {
			subscribeFresh(s, "u1", fmt.Sprintf(`{"channel":%q,"token":%q}`, ch, tok))
		}
	})
}
