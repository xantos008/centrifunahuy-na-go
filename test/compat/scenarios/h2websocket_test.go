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
