// Package pubsubfixture addresses each compatibility target's local emulator.
package pubsubfixture

import (
	"context"
	"fmt"
	"os"

	pb "cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const Topic = "projects/compat-project/topics/commands"
const Subscription = "projects/compat-project/subscriptions/compat-sub"

// Dial uses the emulator protocol without SDK credential discovery.
func Dial(side string) (*grpc.ClientConn, error) {
	var addr string
	switch side {
	case "ref":
		addr = os.Getenv("COMPAT_REF_PUBSUB_ADDR")
		if addr == "" {
			addr = "127.0.0.1:18085"
		}
	case "sut":
		addr = os.Getenv("COMPAT_SUT_PUBSUB_ADDR")
		if addr == "" {
			addr = "127.0.0.1:28085"
		}
	default:
		return nil, fmt.Errorf("unknown Pub/Sub target %q", side)
	}
	return grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func Provision(ctx context.Context, side string) error {
	conn, err := Dial(side)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	_, err = pb.NewPublisherClient(conn).CreateTopic(ctx, &pb.Topic{Name: Topic}, grpc.WaitForReady(true))
	if err != nil && status.Code(err) != codes.AlreadyExists {
		return err
	}
	_, err = pb.NewSubscriberClient(conn).CreateSubscription(ctx, &pb.Subscription{
		Name: Subscription, Topic: Topic, AckDeadlineSeconds: 10, EnableMessageOrdering: true,
	})
	if status.Code(err) == codes.AlreadyExists {
		return nil
	}
	return err
}
