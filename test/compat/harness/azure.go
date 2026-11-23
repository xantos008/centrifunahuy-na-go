package harness

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

// AzureClient connects only to this project's published emulator port.
func AzureClient(target string) (*azservicebus.Client, error) {
	addr, err := compose("port", serviceFor("servicebus", target), "5672")
	if err != nil {
		return nil, err
	}
	addr = strings.TrimSpace(addr)
	host, _, err := net.SplitHostPort(addr)
	if err != nil || !net.ParseIP(host).IsLoopback() {
		return nil, fmt.Errorf("unexpected Service Bus binding %q", addr)
	}
	return azservicebus.NewClientFromConnectionString("Endpoint=sb://"+addr+
		";SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=SAS_KEY_VALUE;UseDevelopmentEmulator=true;", nil)
}

// AzurePeekFirst always includes the oldest message. The SDK's default peek
// cursor advances even when that message is locked and still unsettled.
func AzurePeekFirst(ctx context.Context, receiver *azservicebus.Receiver) ([]*azservicebus.ReceivedMessage, error) {
	first := int64(0)
	return receiver.PeekMessages(ctx, 1, &azservicebus.PeekMessagesOptions{FromSequenceNumber: &first})
}

func probeAzure() error {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	receivers := make(map[string]*azservicebus.Receiver)
	for _, side := range []string{"ref", "sut"} {
		c, err := AzureClient(side)
		if err != nil {
			return err
		}
		defer func() { _ = c.Close(ctx) }()
		tx, err := c.NewSender("compat-isolation", nil)
		if err != nil {
			return err
		}
		if err := tx.SendMessage(ctx, &azservicebus.Message{Body: []byte(side + runID)}, nil); err != nil {
			return fmt.Errorf("CS-ISO-01: send marker via %s: %w", side, err)
		}
		rx, err := c.NewReceiverForQueue("compat-isolation", nil)
		if err != nil {
			return err
		}
		receivers[side] = rx
	}
	// Keep both markers queued until both peeks finish. A shared emulator
	// would return the same first message on both sides.
	for side, rx := range receivers {
		msgs, err := AzurePeekFirst(ctx, rx)
		if err != nil {
			return fmt.Errorf("CS-ISO-01: peek via %s: %w", side, err)
		}
		if len(msgs) != 1 || string(msgs[0].Body) != side+runID {
			return fmt.Errorf("CS-ISO-01: %s did not see its own first marker", side)
		}
	}
	for side, rx := range receivers {
		msgs, err := rx.ReceiveMessages(ctx, 1, nil)
		if err != nil {
			return err
		}
		if len(msgs) != 1 || string(msgs[0].Body) != side+runID {
			return fmt.Errorf("CS-ISO-01: missing marker for %s", side)
		}
		if err := rx.CompleteMessage(ctx, msgs[0], nil); err != nil {
			return err
		}
	}
	return nil
}
