package scenarios

import (
	"fmt"
	"testing"
	"time"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_RPC_04(t *testing.T) {
	h.Profiles(t, "r")
	h.Run(t, "CS-RPC-04", func(s *h.Session) {
		restore := s.ReplaceB("r-concurrent")
		defer func() {
			restore()
			startB(s)
		}()
		s.OnB(func() {
			token := h.HS256(h.Claims{Sub: "proxy-user"})
			for _, namespace := range []string{"wait_http", "wait_grpc"} {
				observer := connect(s, token)
				victim := connect(s, token)
				var connected protocol.Reply
				victim.Last(&connected)
				client := connected.GetConnect().GetClient()
				if client == "" {
					s.T.Fatalf("victim connect returned %v", &connected)
				}
				observe := func(id uint32, method string) {
					s.T.Helper()
					observer.Send(fmt.Sprintf(`{"id":%d,"rpc":{"method":%q,"data":{"client":%q}}}`, id, namespace+":"+method, client))
					observer.Recv(1)
					var reply protocol.Reply
					observer.Last(&reply)
					if reply.Id != id || reply.Error != nil || string(reply.GetRpc().GetData()) != "true" {
						s.T.Fatalf("%s %s observation returned %v", namespace, method, &reply)
					}
				}
				started := time.Now()
				victim.Send(fmt.Sprintf(`{"id":2,"rpc":{"method":%q}}`, namespace+":hold"))
				observe(2, "started")
				victim.Close()
				s.Note("closed peer with RPC pending")
				observe(3, "was-canceled")
				// The configured callback timeout is 25s. A deadline must never
				// satisfy the observation even if the test process is suspended.
				if elapsed := time.Since(started); elapsed >= 20*time.Second {
					s.T.Fatalf("%s cancellation observed after %s, too close to the 25s deadline", namespace, elapsed)
				}
				observer.Close()
			}
		})
	})
}
