package main

import (
	"context"
	"errors"
	"strings"
	"time"

	p "github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
)

// A malformed callback is a backend failure. Expiry scenarios therefore only
// observe an application rejection when the server forwards the expected data.
func validRefresh(client, transport, protocol, encoding string, meta p.Raw) bool {
	return client != "" && transport == "websocket" && protocol == "json" && encoding == "json" &&
		string(meta) == `{"name":"lease"}`
}

func (fixture) Refresh(_ context.Context, r *p.RefreshRequest) (*p.RefreshResponse, error) {
	if !validRefresh(r.Client, r.Transport, r.Protocol, r.Encoding, r.Meta) {
		return nil, errors.New("unexpected connection refresh fields")
	}
	switch r.User {
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

func (fixture) SubRefresh(_ context.Context, r *p.SubRefreshRequest) (*p.SubRefreshResponse, error) {
	if r.User != "proxy-user" || !validRefresh(r.Client, r.Transport, r.Protocol, r.Encoding, r.Meta) {
		return nil, errors.New("unexpected subscription refresh fields")
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
