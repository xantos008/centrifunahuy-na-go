package scenarios

import (
	"fmt"
	"testing"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// CS-PROTO-01: JSON connect, subscribe, server API publish, receive; then two
// commands batched in one frame.
func TestCS_PROTO_01(t *testing.T) {
	h.Run(t, "CS-PROTO-01", func(s *h.Session) {
		ws := s.Dial()
		ws.Send(fmt.Sprintf(`{"id":1,"connect":{"token":%q,"name":"compat"}}`, h.HS256(h.Claims{Sub: "u1"})))
		ws.Recv(1)
		ws.Send(`{"id":2,"subscribe":{"channel":"pub:proto01"}}`)
		ws.Recv(1)
		s.API(h.APIRequest{Path: "/api/publish", Body: `{"channel":"pub:proto01","data":{"text":"hello"}}`})
		ws.Recv(1)
		ws.Send(`{"id":3,"subscribe":{"channel":"pub:proto01-a"}}` + "\n" + `{"id":4,"subscribe":{"channel":"pub:proto01-b"}}`)
		ws.Recv(2)
	})
}
