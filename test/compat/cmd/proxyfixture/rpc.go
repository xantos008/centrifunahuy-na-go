package main

import (
	"context"
	"sync"
	"time"

	p "github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
)

// Each client has one timeout probe. Limit retained observations even if a
// scenario exits before its follow-up call. Ref and sut run separate fixtures.
type rpcTimeouts struct {
	mu     sync.Mutex
	next   int
	recent [32]struct {
		client   string
		canceled chan bool
	}
}

func (t *rpcTimeouts) start(client string) chan bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	probe := &t.recent[t.next]
	t.next = (t.next + 1) % len(t.recent)
	probe.client = client
	probe.canceled = make(chan bool, 1)
	return probe.canceled
}

func (t *rpcTimeouts) take(client string) <-chan bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := range t.recent {
		probe := &t.recent[i]
		if probe.client == client {
			probe.client = ""
			return probe.canceled
		}
	}
	return nil
}

func (f fixture) RPC(ctx context.Context, r *p.RPCRequest) (*p.RPCResponse, error) {
	if r.User != "proxy-user" || r.Client == "" || r.Transport != "websocket" || r.Protocol != "json" || r.Encoding != "json" {
		return &p.RPCResponse{Error: denied()}, nil
	}
	if route := callbackRoute(ctx); route != "" {
		if route == "alternate_rpc" && r.Method == "named:echo" && string(r.Data) == `"rpc"` {
			return &p.RPCResponse{Result: &p.RPCResult{Data: p.Raw(`"alternate_rpc"`)}}, nil
		}
		return &p.RPCResponse{Error: denied()}, nil
	}
	switch r.Method {
	case "probe:echo":
		return &p.RPCResponse{Result: &p.RPCResult{Data: r.Data}}, nil
	case "probe:timeout":
		canceled := f.rpcTimeouts.start(r.Client)
		defer close(canceled)
		select {
		case <-ctx.Done():
			canceled <- true
			return nil, ctx.Err()
		case <-time.After(4 * time.Second):
			// A missing proxy deadline must produce a different wire result.
			return &p.RPCResponse{Error: denied()}, nil
		}
	case "probe:was-canceled":
		canceled := f.rpcTimeouts.take(r.Client)
		if canceled != nil {
			// The client can receive error 100 before the canceled handler runs.
			select {
			case observed := <-canceled:
				if observed {
					return &p.RPCResponse{Result: &p.RPCResult{Data: p.Raw(`true`)}}, nil
				}
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	return &p.RPCResponse{Error: denied()}, nil
}
