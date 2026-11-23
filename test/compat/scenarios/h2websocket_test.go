package scenarios

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_TRANS_07(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-TRANS-07", func(s *h.Session) {
		token := h.HS256(h.Claims{Sub: "h2-reader"})
		connect := fmt.Sprintf(`{"token":%q}`, token)
		var reply protocol.Reply
		if err := json.Unmarshal(s.H2WebSocketConnect("/connection/websocket", `{"id":1,"connect":`+connect+`}`), &reply); err != nil || reply.Id != 1 || reply.GetConnect().GetClient() == "" {
			s.T.Fatalf("bidirectional WebSocket connect: %v, %v", &reply, err)
		}
	})
}

// ADR-0032: exercise the fixed handler on Cedrazeka. The regular
// bidirectional comparison above remains common to both servers.
func TestCedrazekaUniWebSocketHTTP2(t *testing.T) {
	h.Profiles(t, "m")
	if h.Mode() == "record" {
		t.Skip("Cedrazeka-specific regression")
	}
	s := &h.Session{T: t, Target: h.SUT()}
	connect := fmt.Sprintf(`{"token":%q}`, h.HS256(h.Claims{Sub: "uni-h2-reader"}))
	var push protocol.Push
	if err := json.Unmarshal(s.H2WebSocketConnect("/connection/uni_websocket", connect), &push); err != nil || push.GetConnect().GetClient() == "" {
		t.Fatalf("unidirectional connect push: %v, %v", &push, err)
	}
	var reply protocol.Reply
	if err := json.Unmarshal(s.H2WebSocketConnect("/connection/websocket", `{"id":1,"connect":`+connect+`}`), &reply); err != nil || reply.GetConnect().GetClient() == "" {
		t.Fatalf("subsequent WebSocket connection: %v, %v", &reply, err)
	}
}
