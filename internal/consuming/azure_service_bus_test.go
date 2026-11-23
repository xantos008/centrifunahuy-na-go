//go:build integration && azure_emulator

package consuming

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestAzureServiceBusConsumer_DrainAcksInFlight(t *testing.T) {
	connection := os.Getenv("COMPAT_AZURE_DRAIN_CONNECTION_STRING")
	if connection == "" {
		t.Skip("set COMPAT_AZURE_DRAIN_CONNECTION_STRING for a local Azure Service Bus emulator")
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	const queue = "compat-drain"
	const method = "publish"
	const body = `{"source":"drain"}`
	client, err := azservicebus.NewClientFromConnectionString(connection, nil)
	require.NoError(t, err)
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		_ = client.Close(closeCtx)
	}()
	sender, err := client.NewSender(queue, nil)
	require.NoError(t, err)
	receiver, err := client.NewReceiverForQueue(queue, nil)
	require.NoError(t, err)
	first := int64(0)
	peekFromStart := &azservicebus.PeekMessagesOptions{FromSequenceNumber: &first}
	pending, err := receiver.PeekMessages(ctx, 1, peekFromStart)
	require.NoError(t, err)
	require.Empty(t, pending, "use a fresh compat-drain queue")

	type command struct{ method, body string }
	dispatching := make(chan command, 1)
	release := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	consumer, err := NewAzureServiceBusConsumer(AzureServiceBusConsumerConfig{
		ConnectionString:   connection,
		Queues:             []string{queue},
		MaxConcurrentCalls: 1,
		MaxReceiveMessages: 1,
		MethodProperty:     "centrifugo-method",
	}, &MockDispatcher{
		onDispatchCommand: func(_ context.Context, method string, data []byte) error {
			select {
			case dispatching <- command{method, string(data)}:
			case <-ctx.Done():
				return ctx.Err()
			}
			// The test keeps the accepted delivery in flight past Run cancellation.
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}, testCommon(prometheus.NewRegistry()))
	require.NoError(t, err)
	runCtx, runCancel := context.WithCancel(ctx)
	defer func() {
		runCancel()
		unblock()
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		_ = consumer.Close(closeCtx)
	}()
	runDone := make(chan error, 1)
	go func() { runDone <- consumer.Run(runCtx) }()

	messageID := uuid.NewString()
	require.NoError(t, sender.SendMessage(ctx, &azservicebus.Message{
		MessageID:             &messageID,
		Body:                  []byte(body),
		ApplicationProperties: map[string]any{"centrifugo-method": method},
	}, nil))
	select {
	case got := <-dispatching:
		require.Equal(t, command{method, body}, got)
	case <-ctx.Done():
		t.Fatal("consumer did not accept the message")
	}

	runCancel()
	closeCtx, closeCancel := context.WithTimeout(ctx, 10*time.Second)
	defer closeCancel()
	closeStarted := make(chan struct{})
	closeDone := make(chan error, 1)
	go func() {
		close(closeStarted)
		closeDone <- consumer.Close(closeCtx)
	}()
	select {
	case <-closeStarted:
	case <-closeCtx.Done():
		t.Fatal("Close did not start")
	}
	pending, err = receiver.PeekMessages(closeCtx, 1, peekFromStart)
	require.NoError(t, err)
	require.Len(t, pending, 1, "AMQP should still expose the locked delivery while the handler is blocked")
	require.Equal(t, messageID, pending[0].MessageID)
	select {
	case err := <-closeDone:
		t.Fatalf("Close returned before the accepted handler finished: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	unblock()
	select {
	case err := <-closeDone:
		require.NoError(t, err)
	case <-closeCtx.Done():
		t.Fatal("Close did not finish after the handler was released")
	}
	select {
	case err := <-runDone:
		require.ErrorIs(t, err, context.Canceled)
	case <-ctx.Done():
		t.Fatal("Run did not return after Close")
	}

	pending, err = receiver.PeekMessages(ctx, 1, peekFromStart)
	require.NoError(t, err)
	require.Empty(t, pending, "the accepted message was not settled in AMQP")
	deadLetter, err := client.NewReceiverForQueue(queue, &azservicebus.ReceiverOptions{SubQueue: azservicebus.SubQueueDeadLetter})
	require.NoError(t, err)
	dead, err := deadLetter.PeekMessages(ctx, 1, peekFromStart)
	require.NoError(t, err)
	require.Empty(t, dead, "the accepted message was dead-lettered instead of completed")
}
