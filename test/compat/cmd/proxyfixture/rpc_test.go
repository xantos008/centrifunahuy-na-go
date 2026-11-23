package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	p "github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
)

type rpcProbeContext struct {
	context.Context
	waiting chan struct{}
	once    sync.Once
}

func (c *rpcProbeContext) Done() <-chan struct{} {
	// RPC evaluates Done after registering its timeout probe.
	c.once.Do(func() { close(c.waiting) })
	return c.Context.Done()
}

func TestRPCCancellationObservations(t *testing.T) {
	f := fixture{rpcTimeouts: &rpcTimeouts{}}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	request := func(client, method string) *p.RPCRequest {
		return &p.RPCRequest{
			Client: client, User: "proxy-user", Transport: "websocket",
			Protocol: "json", Encoding: "json", Method: method,
		}
	}
	start := func(client string) (context.CancelFunc, <-chan struct{}, <-chan error) {
		probe, cancelProbe := context.WithCancel(ctx)
		probeCtx := &rpcProbeContext{Context: probe, waiting: make(chan struct{})}
		done := make(chan error, 1)
		go func() {
			_, err := f.RPC(probeCtx, request(client, "probe:timeout"))
			done <- err
		}()
		return cancelProbe, probeCtx.waiting, done
	}
	consume := func(client string) {
		t.Helper()
		observed, err := f.RPC(ctx, request(client, "probe:was-canceled"))
		if err != nil || observed.GetError() != nil || string(observed.GetResult().GetData()) != "true" {
			t.Fatalf("%s: expected cancellation observation, got %v, %v", client, observed, err)
		}
		observed, err = f.RPC(ctx, request(client, "probe:was-canceled"))
		if err != nil || observed.GetError().GetCode() != 470 || observed.GetResult() != nil {
			t.Fatalf("%s: expected observation to be consumed, got %v, %v", client, observed, err)
		}
	}
	awaitCanceled := func(client string, done <-chan error) {
		t.Helper()
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("%s: timeout callback returned %v, want context.Canceled", client, err)
			}
		case <-ctx.Done():
			t.Fatalf("%s: timeout callback did not finish", client)
		}
	}

	cancelA, waitingA, doneA := start("client-a")
	cancelB, waitingB, doneB := start("client-b")
	for _, waiting := range []<-chan struct{}{waitingA, waitingB} {
		select {
		case <-waiting:
		case <-ctx.Done():
			t.Fatal("timeout callbacks did not overlap")
		}
	}

	cancelA()
	consume("client-a")
	awaitCanceled("client-a", doneA)
	cancelB()
	consume("client-b")
	awaitCanceled("client-b", doneB)
}
