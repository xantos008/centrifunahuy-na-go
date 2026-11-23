package scenarios

import (
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_TRANS_02(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-TRANS-02", func(s *h.Session) {
		for _, method := range []string{"frame", "query"} {
			channel := s.Ch("pub:uni-ws-" + method)
			user := s.Ch("uni-ws-user-" + method)
			token := h.HS256(h.Claims{Sub: user, Channels: []string{channel}})
			ws := s.DialUniWS(fmt.Sprintf(`{"token":%q}`, token), method == "query")
			read := func() *protocol.Push {
				s.T.Helper()
				ws.Recv(1)
				var push protocol.Push
				ws.Last(&push)
				return &push
			}
			connected := read().Connect
			if connected == nil || connected.Client == "" || connected.Pong || connected.Subs[channel] == nil {
				s.T.Fatalf("%s: unexpected connect push: %v", method, connected)
			}
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":1}}`, channel)})
			delivered := read()
			if delivered.Channel != channel || string(delivered.GetPub().GetData()) != `{"n":1}` {
				s.T.Fatalf("%s: unexpected publication: %v", method, delivered)
			}
			s.API(h.APIRequest{Path: "/api/disconnect", Body: fmt.Sprintf(`{"user":%q,"disconnect":{"code":4001,"reason":"test complete"}}`, user)})
			disconnect := read().Disconnect
			if disconnect.GetCode() != 4001 || disconnect.GetReason() != "test complete" {
				s.T.Fatalf("%s: unexpected disconnect: %v", method, disconnect)
			}
			if code := ws.ExpectClose(); code != 4001 {
				s.T.Fatalf("%s: unexpected WebSocket close code: %d", method, code)
			}
		}
	})
}
