package main

import (
	"context"
	"net/http"

	"google.golang.org/grpc/metadata"
)

const routeHeader = "x-compat-route"

type httpRouteKey struct{}

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
