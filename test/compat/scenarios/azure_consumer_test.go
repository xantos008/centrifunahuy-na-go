package scenarios

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_CONS_07(t *testing.T) {
	h.Profiles(t, "azure")
	h.Run(t, "CS-CONS-07", func(s *h.Session) {
		channel := s.Ch("hist:azure")
		ws := connect(s, h.HS256(h.Claims{Sub: "azure-reader"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, channel))
		ws.Recv(1)
		var sub protocol.Reply
		ws.Last(&sub)
		if sub.Subscribe == nil {
			s.T.Fatalf("reader setup: %v", &sub)
		}

		ctx, cancel := context.WithTimeout(s.T.Context(), 45*time.Second)
		defer cancel()
		client, err := h.AzureClient(s.Target.Name)
		if err != nil {
			s.T.Fatal(err)
		}
		defer func() { _ = client.Close(ctx) }()
		const queue = "compat-commands"
		tx, err := client.NewSender(queue, nil)
		if err != nil {
			s.T.Fatal(err)
		}
		msgs := []*azservicebus.Message{
			{ApplicationProperties: map[string]any{"centrifugo-method": "publish"},
				Body: []byte(fmt.Sprintf(`{"channel":%q,"data":{"n":1}}`, channel))},
			{Body: []byte(fmt.Sprintf(`{"method":"publish","payload":{"channel":%q,"data":{"n":2}}}`, channel))},
		}
		for i, msg := range msgs {
			if err := tx.SendMessage(ctx, msg, nil); err != nil {
				s.T.Fatal(err)
			}
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			push := reply.GetPush()
			pub := push.GetPub()
			var data struct{ N int }
			if push.GetChannel() != channel || pub.GetOffset() != uint64(i+1) || json.Unmarshal(pub.GetData(), &data) != nil || data.N != i+1 {
				s.T.Fatalf("unexpected Service Bus publication %d: %v", i+1, &reply)
			}
		}
		rx, err := client.NewReceiverForQueue(queue, nil)
		if err != nil {
			s.T.Fatal(err)
		}
		for {
			pending, err := h.AzurePeekFirst(ctx, rx)
			if err != nil {
				s.T.Fatal(err)
			}
			if len(pending) == 0 {
				break
			}
			select {
			case <-ctx.Done():
				s.T.Fatal("command queue still contains an unsettled delivery")
			case <-time.After(100 * time.Millisecond):
			}
		}
		dlq, err := client.NewReceiverForQueue(queue, &azservicebus.ReceiverOptions{SubQueue: azservicebus.SubQueueDeadLetter})
		if err != nil {
			s.T.Fatal(err)
		}
		dead, err := h.AzurePeekFirst(ctx, dlq)
		if err != nil || len(dead) != 0 {
			s.T.Fatalf("dead-letter queue: %d messages, %v", len(dead), err)
		}
		s.Note("Service Bus command queue and dead-letter queue are empty after both deliveries")
	})
}
