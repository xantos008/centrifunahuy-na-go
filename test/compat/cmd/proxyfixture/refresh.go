package main

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	p "github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
)

// A malformed callback is a backend failure. Expiry scenarios therefore only
// observe an application rejection when the server forwards the expected data.
func validRefresh(client, transport, protocol, encoding string, meta p.Raw) bool {
	return client != "" && transport == "websocket" && protocol == "json" && encoding == "json" &&
		string(meta) == `{"name":"lease"}`
}

// Each retry scenario uses a fresh client and one expiring subscription.
// The two callbacks fail independently on their first attempt. Keep a bounded
// history in case the client closes before it can retry.
type retryLeases struct {
	mu     sync.Mutex
	next   int
	recent [32]struct {
		key   string
		first time.Time
	}
}

func (l *retryLeases) retry(key string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := range l.recent {
		attempt := &l.recent[i]
		if attempt.key == key {
			// Reject an immediate retry. Both handlers grant another minute
			// after a backend failure; subscription checks run on a timer.
			return time.Since(attempt.first) >= 55*time.Second, nil
		}
	}
	entry := &l.recent[l.next]
	entry.key, entry.first = key, time.Now()
	l.next = (l.next + 1) % len(l.recent)
	return false, errors.New("fixture temporarily unavailable")
}

func (f fixture) Refresh(_ context.Context, r *p.RefreshRequest) (*p.RefreshResponse, error) {
	if !validRefresh(r.Client, r.Transport, r.Protocol, r.Encoding, r.Meta) {
		return nil, errors.New("unexpected connection refresh fields")
	}
	switch r.User {
	case "lease-retry":
		ok, err := f.retryLeases.retry(r.Client)
		return &p.RefreshResponse{Result: &p.RefreshResult{
			Expired: !ok, ExpireAt: time.Now().Add(time.Minute).Unix(), Info: p.Raw(`{"retry":"connection"}`),
		}}, err
	case "lease-renew":
		return &p.RefreshResponse{Result: &p.RefreshResult{
			ExpireAt: time.Now().Add(time.Minute).Unix(), Info: p.Raw(`{"source":"refresh"}`),
		}}, nil
	case "lease-expire":
		return &p.RefreshResponse{Result: &p.RefreshResult{Expired: true}}, nil
	default:
		return nil, errors.New("unexpected refresh user")
	}
}

func (f fixture) SubRefresh(_ context.Context, r *p.SubRefreshRequest) (*p.SubRefreshResponse, error) {
	if !validRefresh(r.Client, r.Transport, r.Protocol, r.Encoding, r.Meta) {
		return nil, errors.New("unexpected subscription refresh fields")
	}
	if r.User == "lease-retry" && strings.HasPrefix(r.Channel, "lease:retry-") {
		ok, err := f.retryLeases.retry(r.Client + "/" + r.Channel)
		return &p.SubRefreshResponse{Result: &p.SubRefreshResult{
			Expired: !ok, ExpireAt: time.Now().Add(time.Minute).Unix(), Info: p.Raw(`{"retry":"subscription"}`),
		}}, err
	}
	if r.User != "proxy-user" {
		return nil, errors.New("unexpected subscription refresh user")
	}
	switch {
	case strings.HasPrefix(r.Channel, "lease:renew-"):
		return &p.SubRefreshResponse{Result: &p.SubRefreshResult{
			ExpireAt: time.Now().Add(time.Minute).Unix(), Info: p.Raw(`{"source":"sub_refresh"}`),
		}}, nil
	case strings.HasPrefix(r.Channel, "lease:expire-"):
		return &p.SubRefreshResponse{Result: &p.SubRefreshResult{Expired: true}}, nil
	default:
		return nil, errors.New("unexpected refresh channel")
	}
}
