package scenarios

import (
	"fmt"
	"testing"
	"time"

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

// Keep m on its original ping-only configuration. The named RPC backend
// runs over gRPC on r and HTTP on p.
func TestCS_RPC_02(t *testing.T) {
	h.Profiles(t, "r", "p")
	h.Run(t, "CS-RPC-02", func(s *h.Session) {
		ws := connect(s, h.HS256(h.Claims{Sub: "proxy-user"}))
		const data = `{"probe":true}`
		for i, tc := range []struct {
			method string
			code   uint32
			reason string
		}{
			{"probe:echo", 0, ""},
			{"probe:deny", 470, "fixture denied"},
			{"unknown", 108, "not available"},
			{"unknown:method", 104, "method not found"},
		} {
			id := uint32(i + 2)
			ws.Send(fmt.Sprintf(`{"id":%d,"rpc":{"method":%q,"data":%s}}`, id, tc.method, data))
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			if reply.Id != id || reply.Error.GetCode() != tc.code || reply.Error.GetMessage() != tc.reason {
				s.T.Fatalf("%s: unexpected RPC result: %v", tc.method, &reply)
			}
			if tc.code == 0 && string(reply.GetRpc().GetData()) != data {
				s.T.Fatalf("%s: expected echoed payload: %v", tc.method, &reply)
			}
		}
	})
}

func TestCS_RPC_03(t *testing.T) {
	h.Profiles(t, "r", "p")
	h.Run(t, "CS-RPC-03", func(s *h.Session) {
		ws := connect(s, h.HS256(h.Claims{Sub: "proxy-user"}))
		started := time.Now()
		ws.Send(`{"id":2,"rpc":{"method":"probe:timeout"}}`)
		ws.Recv(1)
		elapsed := time.Since(started)
		var timeout protocol.Reply
		ws.Last(&timeout)
		if timeout.Id != 2 || timeout.Error.GetCode() != 100 || timeout.Error.GetMessage() != "internal server error" ||
			!timeout.Error.GetTemporary() || timeout.Rpc != nil {
			s.T.Fatalf("expected temporary RPC error 100, got %v", &timeout)
		}
		// Both profiles configure a 1s RPC proxy timeout. Allow scheduling
		// variation while distinguishing an immediate failure or the fixture's 4s guard.
		if elapsed < 750*time.Millisecond || elapsed > 3*time.Second {
			s.T.Fatalf("configured 1s RPC timeout returned after %s", elapsed)
		}

		ws.Send(`{"id":3,"rpc":{"method":"probe:was-canceled"}}`)
		ws.Recv(1)
		var observed protocol.Reply
		ws.Last(&observed)
		if observed.Id != 3 || observed.Error != nil || string(observed.GetRpc().GetData()) != "true" {
			s.T.Fatalf("fixture did not observe callback cancellation for this client: %v", &observed)
		}

		ws.Send(`{"id":4,"rpc":{"method":"probe:echo","data":{"after":"timeout"}}}`)
		ws.Recv(1)
		var echo protocol.Reply
		ws.Last(&echo)
		if echo.Id != 4 || echo.Error != nil || string(echo.GetRpc().GetData()) != `{"after":"timeout"}` {
			s.T.Fatalf("RPC after timeout: expected echoed payload, got %v", &echo)
		}
	})
}
