package main

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	p "github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
	"google.golang.org/grpc/metadata"
)

// Each client has one timeout probe. Limit retained observations even if a
// scenario exits before its follow-up call. Ref and sut run separate fixtures.
type rpcTimeouts struct {
	mu         sync.Mutex
	next       int
	registered chan struct{}
	recent     [32]struct {
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
	if t.registered != nil {
		close(t.registered)
		t.registered = nil
	}
	return probe.canceled
}

// waitStarted also works when the observer's RPC arrives before the victim's.
func (t *rpcTimeouts) waitStarted(ctx context.Context, client string) bool {
	for {
		t.mu.Lock()
		for i := range t.recent {
			if probe := &t.recent[i]; probe.client == client {
				canceled := probe.canceled
				t.mu.Unlock()
				select {
				case <-canceled:
					return false
				default:
					return true
				}
			}
		}
		if t.registered == nil {
			t.registered = make(chan struct{})
		}
		registered := t.registered
		t.mu.Unlock()
		select {
		case <-registered:
		case <-ctx.Done():
			return false
		}
	}
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
	namespace, action, _ := strings.Cut(r.Method, ":")
	if namespace == "wait_http" || namespace == "wait_grpc" {
		_, grpc := metadata.FromIncomingContext(ctx)
		if grpc != (namespace == "wait_grpc") {
			return &p.RPCResponse{Error: denied()}, nil
		}
		if action == "hold" {
			return f.rpcTimeouts.hold(ctx, r.Client, 30*time.Second)
		}
		var victim struct{ Client string }
		if json.Unmarshal(r.Data, &victim) != nil || victim.Client == "" || victim.Client == r.Client {
			return &p.RPCResponse{Error: denied()}, nil
		}
		observe, cancel := context.WithTimeout(ctx, 4*time.Second)
		defer cancel()
		switch action {
		case "started":
			if f.rpcTimeouts.waitStarted(observe, victim.Client) {
				return &p.RPCResponse{Result: &p.RPCResult{Data: p.Raw(`true`)}}, nil
			}
		case "was-canceled":
			return f.rpcTimeouts.observe(observe, victim.Client)
		}
		return &p.RPCResponse{Error: denied()}, nil
	}
	if route := callbackRoute(ctx); route != "" {
		if route == "alternate_rpc" && r.Method == "named:echo" && string(r.Data) == `"rpc"` {
			return &p.RPCResponse{Result: &p.RPCResult{Data: p.Raw(`"alternate_rpc"`)}}, nil
		}
		return &p.RPCResponse{Error: denied()}, nil
	}
	switch r.Method {
	case "marker:read":
		return markerRPC(ctx)
	case "probe:echo":
		return &p.RPCResponse{Result: &p.RPCResult{Data: r.Data}}, nil
	case "probe:timeout":
		return f.rpcTimeouts.hold(ctx, r.Client, 4*time.Second)
	case "probe:was-canceled":
		return f.rpcTimeouts.observe(ctx, r.Client)
	}
	return &p.RPCResponse{Error: denied()}, nil
}

func (t *rpcTimeouts) hold(ctx context.Context, client string, guard time.Duration) (*p.RPCResponse, error) {
	canceled := t.start(client)
	defer close(canceled)
	select {
	case <-ctx.Done():
		canceled <- true
		return nil, ctx.Err()
	case <-time.After(guard):
		return &p.RPCResponse{Error: denied()}, nil
	}
}

func (t *rpcTimeouts) observe(ctx context.Context, client string) (*p.RPCResponse, error) {
	if canceled := t.take(client); canceled != nil {
		select {
		case observed := <-canceled:
			if observed {
				return &p.RPCResponse{Result: &p.RPCResult{Data: p.Raw(`true`)}}, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return &p.RPCResponse{Error: denied()}, nil
}
