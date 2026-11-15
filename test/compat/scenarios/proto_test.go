package scenarios

import (
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	"github.com/gorilla/websocket"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// CS-PROTO-01: JSON connect, subscribe, server API publish, receive; then two
// commands batched in one frame.
func TestCS_PROTO_01(t *testing.T) {
	h.Run(t, "CS-PROTO-01", func(s *h.Session) {
		ch := s.Ch("pub:proto01")
		ws := s.Dial()
		ws.Send(fmt.Sprintf(`{"id":1,"connect":{"token":%q,"name":"compat"}}`, h.HS256(h.Claims{Sub: "u1"})))
		ws.Recv(1)
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
		ws.Recv(1)
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"text":"hello"}}`, ch)})
		ws.Recv(1)
		ws.Send(fmt.Sprintf(`{"id":3,"subscribe":{"channel":%q}}`+"\n"+`{"id":4,"subscribe":{"channel":%q}}`, ch+"-a", ch+"-b"))
		ws.Recv(2)
	})
}

// CS-PROTO-02: the same flow over the Protobuf protocol (varint-delimited frames).
func TestCS_PROTO_02(t *testing.T) {
	h.Run(t, "CS-PROTO-02", func(s *h.Session) {
		ch := s.Ch("pub:proto02")
		ws := s.DialProtobuf()
		ws.SendCmd(&protocol.Command{Id: 1, Connect: &protocol.ConnectRequest{Token: h.HS256(h.Claims{Sub: "u1"}), Name: "compat"}})
		ws.Recv(1)
		ws.SendCmd(&protocol.Command{Id: 2, Subscribe: &protocol.SubscribeRequest{Channel: ch}})
		ws.Recv(1)
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"text":"hello"}}`, ch)})
		ws.Recv(1)
		ws.SendCmd(
			&protocol.Command{Id: 3, Subscribe: &protocol.SubscribeRequest{Channel: ch + "-a"}},
			&protocol.Command{Id: 4, Subscribe: &protocol.SubscribeRequest{Channel: ch + "-b"}},
		)
		ws.Recv(2)
	})
}

// CS-PROTO-03: strict JSON decoding (upstream since 6.10.0). Each malformed
// frame goes on a fresh connection; the first server reaction is recorded.
func TestCS_PROTO_03(t *testing.T) {
	invalidUTF8 := append([]byte(`{"id":1,"connect":{"name":"`), 0xff, '"', '}', '}')
	cases := [][]byte{
		[]byte(`{"id":1,"id":2,"connect":{}}`),   // duplicate key
		invalidUTF8,                              // invalid UTF-8 inside a text frame
		[]byte(`{"id":1,"connect":{}} trailing`), // trailing data
		[]byte(`{"id":1,"no_such_command":{}}`),  // unknown field
		[]byte(`{"id":`),                         // truncated
	}
	h.Run(t, "CS-PROTO-03", func(s *h.Session) {
		for _, frame := range cases {
			ws := s.Dial()
			ws.SendRaw(websocket.TextMessage, frame)
			ws.Next()
		}
	})
}

// CS-PROTO-04: ping/pong (the suite config sets ping_interval 2s, pong_timeout 1s).
// The first ping is answered; the second is not, so the server disconnects.
func TestCS_PROTO_04(t *testing.T) {
	h.Run(t, "CS-PROTO-04", func(s *h.Session) {
		ws := s.Dial()
		ws.Send(fmt.Sprintf(`{"id":1,"connect":{"token":%q}}`, h.HS256(h.Claims{Sub: "u1"})))
		ws.Recv(1)
		ws.RecvPing(true)
		ws.RecvPing(false)
		ws.ExpectClose()
	})
}
