package main

import (
	"context"
	"encoding/json"
	"net/http"

	p "github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
	"google.golang.org/grpc/metadata"
)

const routeHeader = "x-compat-route"

type httpRouteKey struct{}

const markerHeader = "x-compat-marker"

type httpMarkerKey struct{}

func markerHTTP(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), httpMarkerKey{}, r.Header.Values(markerHeader))
		next(w, r.WithContext(ctx))
	}
}

func markerRPC(ctx context.Context) (*p.RPCResponse, error) {
	transport := "http"
	values, ok := ctx.Value(httpMarkerKey{}).([]string)
	if !ok {
		transport = "grpc"
		values = metadata.ValueFromIncomingContext(ctx, markerHeader)
	}
	if len(values) != 1 || (values[0] != "static" && values[0] != "frame" && values[0] != "upgrade") {
		return &p.RPCResponse{Error: denied()}, nil
	}
	data, err := json.Marshal(map[string]string{"transport": transport, "marker": values[0]})
	if err != nil {
		return nil, err
	}
	return &p.RPCResponse{Result: &p.RPCResult{Data: data}}, nil
}

// The HTTP path and configured static header must select the same backend.
func namedHTTP(route string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(routeHeader) != route {
			http.Error(w, "wrong proxy route", http.StatusBadRequest)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), httpRouteKey{}, route)))
	}
}

func callbackRoute(ctx context.Context) string {
	if route, ok := ctx.Value(httpRouteKey{}).(string); ok {
		return route
	}
	if values := metadata.ValueFromIncomingContext(ctx, routeHeader); len(values) == 1 {
		return values[0]
	}
	return ""
}
