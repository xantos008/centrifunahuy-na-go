package scenarios

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/centrifugal/protocol"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// CS-CONS-03: a JetStream durable hands off an unacknowledged API command to
// node B. The subscriber on A observes it and a subsequent, fresh command.
func TestCS_CONS_03(t *testing.T) {
	h.Profiles(t, "n")
	h.Run(t, "CS-CONS-03", func(s *h.Session) {
		const streamName, durable, subject = "COMPAT_COMMANDS", "compat_dispatch", "compat.commands"
		js := s.JetStream()
		stream, err := js.CreateStream(s.T.Context(), jetstream.StreamConfig{
			Name: streamName, Subjects: []string{subject}, Storage: jetstream.MemoryStorage,
			Description: s.Ch("CS-CONS-03"), MaxMsgs: 8,
		})
		if err != nil {
			s.T.Fatalf("create stream: %v", err)
		}
		// ReplaceB registers cleanup after this one, so the replacement stops
		// and the ordinary B restarts before deleting its stream and durable.
		s.T.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), h.Deadline)
			defer cancel()
			if err := js.DeleteStream(ctx, streamName); err != nil {
				s.T.Errorf("delete stream: %v", err)
			}
		})
		consumer, err := stream.CreateConsumer(s.T.Context(), jetstream.ConsumerConfig{
			Durable: durable, AckPolicy: jetstream.AckExplicitPolicy, AckWait: 10 * time.Minute, MaxAckPending: 1,
		})
		if err != nil {
			s.T.Fatalf("create durable: %v", err)
		}

		channel := s.Ch("jetstream-dispatch")
		publish := func(sequence uint64, method, body string) {
			s.T.Helper()
			msg := nats.NewMsg(subject)
			msg.Data = []byte(body)
			if method != "" {
				msg.Header.Set("centrifugo-method", method)
			}
			ack, err := js.PublishMsg(s.T.Context(), msg)
			if err != nil || ack.Stream != streamName || ack.Sequence != sequence {
				s.T.Fatalf("publish stream sequence %d: ack=%v error=%v", sequence, ack, err)
			}
		}
		publish(1, "publish", fmt.Sprintf(`{"channel":%q,"data":{"n":1}}`, channel))
		first, err := consumer.Next(jetstream.FetchMaxWait(h.Deadline))
		if err != nil {
			s.T.Fatalf("first delivery: %v", err)
		}
		md, err := first.Metadata()
		if err != nil || md.Sequence.Stream != 1 || md.NumDelivered != 1 {
			s.T.Fatalf("expected first delivery of sequence 1: metadata=%v error=%v", md, err)
		}
		info, err := consumer.Info(s.T.Context())
		if err != nil || info.AckFloor.Stream != 0 || info.NumAckPending != 1 {
			s.T.Fatalf("expected one pending message: info=%v error=%v", info, err)
		}
		s.Note("first delivery pending on durable at stream sequence 1")

		restore := s.ReplaceB("n-consumer")
		ws := connect(s, h.HS256(h.Claims{Sub: "jetstream-reader"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, channel))
		ws.Recv(1)
		var sub protocol.Reply
		ws.Last(&sub)
		if sub.Id != 2 || sub.Subscribe == nil {
			s.T.Fatalf("subscribe: %v", &sub)
		}
		s.AwaitNATSSubscriptions("centrifugo.client." + channel)
		// Release the held delivery only after node A is listening. JetStream
		// can now redeliver sequence 1 to the application consumer on B.
		if err := first.Nak(); err != nil {
			s.T.Fatalf("release pending delivery: %v", err)
		}
		delivery := func(streamSeq, deliverySeq uint64) {
			s.T.Helper()
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			push := reply.GetPush()
			if push.GetChannel() != channel || string(push.GetPub().GetData()) != fmt.Sprintf(`{"n":%d}`, streamSeq) {
				s.T.Fatalf("unexpected delivery for sequence %d: %v", streamSeq, &reply)
			}
			ctx, cancel := context.WithTimeout(s.T.Context(), h.Deadline)
			defer cancel()
			for {
				info, err := consumer.Info(ctx)
				if err != nil {
					s.T.Fatalf("observe ack floor for stream sequence %d: %v", streamSeq, err)
				}
				if info.AckFloor.Stream == streamSeq && info.AckFloor.Consumer == deliverySeq && info.NumAckPending == 0 {
					break
				}
				select {
				case <-ctx.Done():
					s.T.Fatalf("durable ack floor %v, pending %d", info.AckFloor, info.NumAckPending)
				case <-time.After(25 * time.Millisecond):
				}
			}
			s.Note(fmt.Sprintf("durable acknowledged stream sequence %d at delivery sequence %d", streamSeq, deliverySeq))
		}
		delivery(1, 2)
		publish(2, "", fmt.Sprintf(`{"method":"publish","payload":{"channel":%q,"data":{"n":2}}}`, channel))
		delivery(2, 3)
		ws.Close()
		restore()
		startB(s)
	})
}
