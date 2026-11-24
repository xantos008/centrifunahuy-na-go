package scenarios

import (
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_PROXY_06(t *testing.T) {
	h.Profiles(t, "r", "p")
	h.Run(t, "CS-PROXY-06", func(s *h.Session) {
		token := h.HS256(h.Claims{Sub: "proxy-user"})
		reader := connect(s, token)
		channel := s.Ch("named:routing")
		reader.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"data":"subscribe"}}`, channel))
		reader.Recv(1)
		var sub protocol.Reply
		reader.Last(&sub)
		if sub.Id != 2 || string(sub.GetSubscribe().GetData()) != `"alternate_sub"` {
			s.T.Fatalf("named subscribe proxy: %v", &sub)
		}

		writer := connect(s, token)
		writer.Send(fmt.Sprintf(`{"id":2,"publish":{"channel":%q,"data":"publish"}}`, channel))
		writer.Recv(1)
		var ack protocol.Reply
		writer.Last(&ack)
		if ack.Id != 2 || ack.Publish == nil {
			s.T.Fatalf("named publish proxy: %v", &ack)
		}
		reader.Recv(1)
		var delivery protocol.Reply
		reader.Last(&delivery)
		if delivery.GetPush().GetChannel() != channel || string(delivery.GetPush().GetPub().GetData()) != `"alternate_pub"` {
			s.T.Fatalf("named proxy publication: %v", &delivery)
		}

		// The same input selects a different backend through the RPC namespace.
		for i, tc := range []struct{ method, want string }{
			{"named:echo", `"alternate_rpc"`},
			{"probe:echo", `"rpc"`},
		} {
			id := uint32(i + 3)
			writer.Send(fmt.Sprintf(`{"id":%d,"rpc":{"method":%q,"data":"rpc"}}`, id, tc.method))
			writer.Recv(1)
			var reply protocol.Reply
			writer.Last(&reply)
			if reply.Id != id || string(reply.GetRpc().GetData()) != tc.want {
				s.T.Fatalf("%s: unexpected RPC reply: %v", tc.method, &reply)
			}
		}
	})
}
