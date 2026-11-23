// Modified by the Cedrazeka authors, 2026. See NOTICE.

package api

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	. "github.com/xantos008/centrifunahuy-na-go/internal/apiproto"
	"github.com/xantos008/centrifunahuy-na-go/internal/config"
	"github.com/xantos008/centrifunahuy-na-go/internal/configtypes"

	"github.com/centrifugal/centrifuge"
	"github.com/stretchr/testify/require"
)

func TestToAPIErr(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want *Error
	}{
		{"broker failure", errors.New("connection refused"), ErrorInternal},
		{"API error", ErrorConflict, ErrorConflict},
		{"unknown channel", centrifuge.ErrorUnknownChannel, ErrorUnknownChannel},
		{"bad request, wrapped", fmt.Errorf("%w: channel name is not supported", centrifuge.ErrorBadRequest), ErrorBadRequest},
		{"not available", centrifuge.ErrorNotAvailable, ErrorNotAvailable},
		{"too many requests", centrifuge.ErrorTooManyRequests, ErrorTooManyRequests},
		{"unrecoverable position", centrifuge.ErrorUnrecoverablePosition, ErrorUnrecoverablePosition},
		// 113 means another thing in the client protocol than in the API.
		{"concurrent pagination, temporary", centrifuge.ErrorConcurrentPagination, ErrorInternal},
		{"no API equivalent, temporary", &centrifuge.Error{Code: 4000, Temporary: true}, ErrorInternal},
		{"no API equivalent, permanent", centrifuge.ErrorLimitExceeded, ErrorBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, toAPIErr(tc.err))
		})
	}
}

// Embedding the Broker interface routes each publication through Publish,
// including publications made by a broadcast.
type failingBroker struct {
	centrifuge.Broker
	err error
}

func (b *failingBroker) Publish(ch string, data []byte, opts centrifuge.PublishOptions) (centrifuge.PublishResult, error) {
	if ch == "failed" && b.err != nil {
		return centrifuge.PublishResult{}, b.err
	}
	return b.Broker.Publish(ch, data, opts)
}

func TestNodeErrorsReachAPI(t *testing.T) {
	for _, tc := range []struct {
		name  string
		err   error
		want  *Error
		retry bool
	}{
		{"refused", fmt.Errorf("%w: channel name is not supported", centrifuge.ErrorBadRequest), ErrorBadRequest, false},
		{"broker failure", errors.New("connection refused"), ErrorInternal, true},
		{"rate limited", centrifuge.ErrorTooManyRequests, ErrorTooManyRequests, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			node, err := centrifuge.New(centrifuge.Config{})
			require.NoError(t, err)
			memory, err := centrifuge.NewMemoryBroker(node, centrifuge.MemoryBrokerConfig{})
			require.NoError(t, err)
			broker := &failingBroker{Broker: memory, err: tc.err}
			node.SetBroker(broker)
			require.NoError(t, node.Run())
			defer func() { _ = node.Shutdown(context.Background()) }()

			cfg := config.DefaultConfig()
			cfg.Channel.WithoutNamespace.HistorySize = 4
			cfg.Channel.WithoutNamespace.HistoryTTL = configtypes.Duration(time.Minute)
			cfgContainer, err := config.NewContainer(cfg)
			require.NoError(t, err)
			executor := NewExecutor(node, cfgContainer, &testSurveyCaller{}, ExecutorConfig{Protocol: "test"})

			resp := executor.Publish(ctx, &PublishRequest{Channel: "failed", Data: []byte("{}")})
			require.Equal(t, tc.want, resp.Error)
			result := executor.Broadcast(ctx, &BroadcastRequest{Channels: []string{"api-ok", "failed"}, Data: []byte("{}")})
			require.Nil(t, result.Error)
			require.Len(t, result.GetResult().GetResponses(), 2)
			require.Nil(t, result.Result.Responses[0].Error)
			require.EqualValues(t, 1, result.Result.Responses[0].GetResult().GetOffset())
			require.Equal(t, tc.want, result.Result.Responses[1].Error)

			dispatcher := NewDispatcher(NewConsumingHandler(node, executor, ConsumingHandlerConfig{}))
			publishErr := dispatcher.DispatchCommand(ctx, "publish", []byte(`{"channel":"failed","data":{}}`))
			payload := []byte(`{"channels":["delivered","failed"],"data":{"n":1}}`)
			broadcastErr := dispatcher.DispatchCommand(ctx, "broadcast", payload)
			wantDeliveries := 1
			if tc.retry {
				require.ErrorIs(t, publishErr, tc.want)
				require.ErrorIs(t, broadcastErr, tc.want)

				// A consumer redelivers the original message once the broker recovers.
				broker.err = nil
				require.NoError(t, dispatcher.DispatchCommand(ctx, "broadcast", payload))
				wantDeliveries = 2
			} else {
				require.NoError(t, publishErr)
				require.NoError(t, broadcastErr)
			}

			history, err := node.History("delivered", centrifuge.WithLimit(-1))
			require.NoError(t, err)
			require.Len(t, history.Publications, wantDeliveries)
			for _, pub := range history.Publications {
				require.JSONEq(t, `{"n":1}`, string(pub.Data))
			}
			recovered, err := node.History("failed", centrifuge.WithLimit(-1))
			require.NoError(t, err)
			require.Len(t, recovered.Publications, wantDeliveries-1)
		})
	}
}
