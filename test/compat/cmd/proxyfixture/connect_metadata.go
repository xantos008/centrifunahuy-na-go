package main

import (
	"context"
	"encoding/json"
	"net/http"

	p "github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
)

const (
	connectMarker     = "x-compat-connect-marker"
	connectUnlisted   = "x-compat-connect-unlisted"
	connectCredential = "x-compat-connect-credential"
)

type connectHeadersKey struct{}
type connectCompressionKey struct{}

// connectStats records the actual request compressor. grpc-encoding is not
// exposed to the callback as incoming metadata.
type connectStats struct{}

func (connectStats) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	if info.FullMethodName == p.CentrifugoProxy_Connect_FullMethodName {
		return context.WithValue(ctx, connectCompressionKey{}, new(string))
	}
	return ctx
}

func (connectStats) HandleRPC(ctx context.Context, event stats.RPCStats) {
	if header, ok := event.(*stats.InHeader); ok {
		if compression, ok := ctx.Value(connectCompressionKey{}).(*string); ok {
			*compression = header.Compression
		}
	}
}

func (connectStats) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context { return ctx }
func (connectStats) HandleConn(context.Context, stats.ConnStats)                       {}

func metadataConnect(ctx context.Context, req *p.ConnectRequest) (*p.ConnectResponse, error) {
	if req.Client == "" || req.Transport != "uni_grpc" || req.Protocol != "protobuf" || req.Encoding != "json" {
		return &p.ConnectResponse{Error: denied()}, nil
	}
	type observation struct {
		Transport  string   `json:"transport"`
		Markers    []string `json:"markers"`
		Unlisted   bool     `json:"unlisted"`
		Credential bool     `json:"credential"`
		Gzip       bool     `json:"gzip"`
	}
	got := observation{Transport: "grpc"}
	if headers, ok := ctx.Value(connectHeadersKey{}).(http.Header); ok {
		got.Transport = "http"
		got.Markers = headers.Values(connectMarker)
		got.Unlisted = len(headers.Values(connectUnlisted)) != 0
	} else {
		got.Markers = metadata.ValueFromIncomingContext(ctx, connectMarker)
		got.Unlisted = len(metadata.ValueFromIncomingContext(ctx, connectUnlisted)) != 0
		credentials := metadata.ValueFromIncomingContext(ctx, connectCredential)
		got.Credential = len(credentials) == 1 && credentials[0] == "compat-test-proxy-credential"
		compression, _ := ctx.Value(connectCompressionKey{}).(*string)
		got.Gzip = compression != nil && *compression == gzip.Name
	}
	data, err := json.Marshal(got)
	if err != nil {
		return nil, err
	}
	return &p.ConnectResponse{Result: &p.ConnectResult{User: "metadata-proxy-user", Data: data}}, nil
}
