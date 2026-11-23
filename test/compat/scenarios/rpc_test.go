package scenarios

import (
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_RPC_01(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-RPC-01", func(s *h.Session) {
		ws := connect(s, h.HS256(h.Claims{Sub: "u1"}))
		ws.Send(`{"id":2,"rpc":{"method":"compat-ping","data":{"probe":true}}}`)
		ws.Recv(1)
		var ping protocol.Reply
		ws.Last(&ping)
		if ping.Id != 2 || ping.Error != nil || ping.Rpc == nil || len(ping.Rpc.Data) != 0 {
			s.T.Fatalf("custom RPC ping: expected an empty RPC result, got %+v", &ping)
		}

		for i, method := range []string{"ping", "unknown"} {
			id := uint32(3 + i)
			ws.Send(fmt.Sprintf(`{"id":%d,"rpc":{"method":%q}}`, id, method))
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			if reply.Id != id || reply.Error.GetCode() != 104 || reply.Rpc != nil {
				s.T.Fatalf("%s: expected method not found, got %+v", method, &reply)
			}
		}
	})
}
