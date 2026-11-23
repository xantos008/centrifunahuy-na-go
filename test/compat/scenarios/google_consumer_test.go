package scenarios

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	pb "cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
	"github.com/xantos008/centrifunahuy-na-go/test/compat/internal/pubsubfixture"
)

func TestCS_CONS_05(t *testing.T) {
	h.Profiles(t, "cloud")
	h.Run(t, "CS-CONS-05", func(s *h.Session) {
		channel := s.Ch("hist:google")
		ws := connect(s, h.HS256(h.Claims{Sub: "google-reader"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, channel))
		ws.Recv(1)
		var sub protocol.Reply
		ws.Last(&sub)
		if sub.Subscribe == nil {
			s.T.Fatalf("reader setup: %v", &sub)
		}
		conn, err := pubsubfixture.Dial(s.Target.Name)
		if err != nil {
			s.T.Fatal(err)
		}
		defer func() { _ = conn.Close() }()
		ctx, cancel := context.WithTimeout(s.T.Context(), h.Deadline)
		defer cancel()
		result, err := pb.NewPublisherClient(conn).Publish(ctx, &pb.PublishRequest{
			Topic: pubsubfixture.Topic,
			Messages: []*pb.PubsubMessage{
				{OrderingKey: channel, Attributes: map[string]string{"centrifugo-method": "publish"},
					Data: []byte(fmt.Sprintf(`{"channel":%q,"data":{"n":1}}`, channel))},
				{OrderingKey: channel, Data: []byte(fmt.Sprintf(`{"method":"publish","payload":{"channel":%q,"data":{"n":2}}}`, channel))},
			},
		})
		if err != nil || len(result.GetMessageIds()) != 2 || result.MessageIds[0] == "" || result.MessageIds[1] == "" || result.MessageIds[0] == result.MessageIds[1] {
			s.T.Fatalf("emulator did not accept both messages: %v, %v", result, err)
		}
		s.Note("Pub/Sub accepted two messages for the same ordering key")
		for n := 1; n <= 2; n++ {
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			push := reply.GetPush()
			pub := push.GetPub()
			var data struct{ N int }
			if push.GetChannel() != channel || pub.GetOffset() != uint64(n) || json.Unmarshal(pub.GetData(), &data) != nil || data.N != n {
				s.T.Fatalf("expected ordered Pub/Sub message %d: %v", n, &reply)
			}
		}
	})
}
