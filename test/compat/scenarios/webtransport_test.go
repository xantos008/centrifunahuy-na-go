package scenarios

import (
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_TRANS_06(t *testing.T) {
	h.Profiles(t, "rsh")
	h.RunWebTransport(t, func(s *h.Session) {
		for _, protobuf := range []bool{false, true} {
			if s.DialWebTransport(protobuf) != nil {
				s.T.Fatal("expected TLS no_application_protocol on the pinned server")
			}
		}
	}, func(s *h.Session) {
		for _, mode := range []string{"json", "protobuf"} {
			connect := func(user string) *h.WT {
				s.T.Helper()
				client := s.DialWebTransport(mode == "protobuf")
				if client == nil {
					s.T.Fatalf("%s: Cedrazeka did not negotiate HTTP/3", mode)
				}
				client.Send(&protocol.Command{Id: 1, Connect: &protocol.ConnectRequest{
					Token: h.HS256(h.Claims{Sub: user}),
				}})
				if reply := client.Recv(); reply.Id != 1 || reply.GetConnect().GetClient() == "" {
					s.T.Fatalf("%s connect: %v", mode, reply)
				}
				return client
			}
			reader, writer := connect("reader"), connect("writer")
			channel := s.Ch("feed:" + mode)
			reader.Send(&protocol.Command{Id: 2, Subscribe: &protocol.SubscribeRequest{Channel: channel}})
			if reply := reader.Recv(); reply.Id != 2 || reply.Subscribe == nil {
				s.T.Fatalf("%s subscribe: %v", mode, reply)
			}
			const payload = `{"transport":"http3"}`
			writer.Send(&protocol.Command{Id: 2, Publish: &protocol.PublishRequest{Channel: channel, Data: []byte(payload)}})
			if reply := writer.Recv(); reply.Id != 2 || reply.Publish == nil {
				s.T.Fatalf("%s publish: %v", mode, reply)
			}
			push := reader.Recv().GetPush()
			if push.GetChannel() != channel || string(push.GetPub().GetData()) != payload || push.GetPub().GetInfo().GetUser() != "writer" {
				s.T.Fatalf("%s delivery: %v", mode, push)
			}
		}
	})
}
