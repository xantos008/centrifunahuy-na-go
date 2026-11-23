// Command proxyfixture serves the application callbacks for the compatibility
// suite. Each side runs a separate instance inside its private Compose network.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	p "github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
	"google.golang.org/grpc"
)

type fixture struct {
	p.UnimplementedCentrifugoProxyServer
	rpcTimeouts *rpcTimeouts
	retryLeases *retryLeases
}

func denied() *p.Error {
	return &p.Error{Code: 470, Message: "fixture denied"}
}

func (fixture) Connect(ctx context.Context, r *p.ConnectRequest) (*p.ConnectResponse, error) {
	if r.Name == "proxy-metadata" {
		return metadataConnect(ctx, r)
	}
	if r.Name != "proxy-compat" {
		// Defer to the server's usual anonymous connection policy.
		return &p.ConnectResponse{}, nil
	}
	var data struct {
		Channel string `json:"channel"`
		Recover bool   `json:"recover"`
	}
	if r.Client == "" || r.Protocol != "json" || r.Encoding != "json" || r.Version != "1" ||
		(r.Transport != "websocket" && r.Transport != "uni_http_stream") ||
		json.Unmarshal(r.Data, &data) != nil || !strings.HasPrefix(data.Channel, "pcache:proxy-") {
		return &p.ConnectResponse{Error: denied()}, nil
	}
	return &p.ConnectResponse{Result: &p.ConnectResult{
		User: "proxy-user", Info: p.Raw(`{"source":"connect"}`),
		Subs: map[string]*p.SubscribeOptions{data.Channel: {
			CacheRecover: data.Recover,
			Info:         p.Raw(`{"source":"connect-sub"}`),
			Override: &p.SubscribeOptionOverride{
				ForceRecovery: &p.BoolValue{Value: true},
			},
		}},
	}}, nil
}

func (fixture) Subscribe(ctx context.Context, r *p.SubscribeRequest) (*p.SubscribeResponse, error) {
	if r.User == "lease-retry" && strings.HasPrefix(r.Channel, "lease:retry-") &&
		r.Client != "" && r.Transport == "websocket" && r.Protocol == "json" && r.Encoding == "json" {
		return &p.SubscribeResponse{Result: &p.SubscribeResult{ExpireAt: time.Now().Unix() + 2}}, nil
	}
	if r.User != "proxy-user" || r.Client == "" || r.Transport != "websocket" || r.Protocol != "json" ||
		r.Encoding != "json" {
		return &p.SubscribeResponse{Error: denied()}, nil
	}
	if route := callbackRoute(ctx); route != "" {
		if route == "alternate_sub" && strings.HasPrefix(r.Channel, "named:") && string(r.Data) == `"subscribe"` {
			return &p.SubscribeResponse{Result: &p.SubscribeResult{Data: p.Raw(`"alternate_sub"`)}}, nil
		}
		return &p.SubscribeResponse{Error: denied()}, nil
	}
	if strings.HasPrefix(r.Channel, "lease:renew-") || strings.HasPrefix(r.Channel, "lease:expire-") ||
		strings.HasPrefix(r.Channel, "namedlease:renew-") {
		return &p.SubscribeResponse{Result: &p.SubscribeResult{
			ExpireAt: time.Now().Unix() + 2, Info: p.Raw(`{"source":"subscribe"}`),
		}}, nil
	}
	if !strings.HasPrefix(r.Channel, "proxied:allowed-") {
		return &p.SubscribeResponse{Error: denied()}, nil
	}
	return &p.SubscribeResponse{Result: &p.SubscribeResult{
		Info: p.Raw(`{"source":"subscribe"}`), Data: r.Data,
	}}, nil
}

func (fixture) Publish(ctx context.Context, r *p.PublishRequest) (*p.PublishResponse, error) {
	if r.User != "proxy-user" || r.Client == "" || r.Transport != "websocket" || r.Protocol != "json" ||
		r.Encoding != "json" {
		return &p.PublishResponse{Error: denied()}, nil
	}
	if route := callbackRoute(ctx); route != "" {
		if route == "alternate_pub" && strings.HasPrefix(r.Channel, "named:") && string(r.Data) == `"publish"` {
			return &p.PublishResponse{Result: &p.PublishResult{Data: p.Raw(`"alternate_pub"`)}}, nil
		}
		return &p.PublishResponse{Error: denied()}, nil
	}
	if strings.HasPrefix(r.Channel, "proxystate:") {
		return statePublication(r), nil
	}
	if !strings.HasPrefix(r.Channel, "proxywrite:") {
		return &p.PublishResponse{Error: denied()}, nil
	}
	var action string
	if err := json.Unmarshal(r.Data, &action); err != nil || (action != "retain" && action != "live") {
		return &p.PublishResponse{Error: denied()}, nil
	}
	return &p.PublishResponse{Result: &p.PublishResult{
		Data: p.Raw(`{"source":"proxy"}`), Tags: map[string]string{"delivery": action}, SkipHistory: action == "live",
	}}, nil
}

func jsonHandler[Request, Response any](call func(context.Context, *Request) (*Response, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&req); err != nil {
			http.Error(w, "invalid callback", http.StatusBadRequest)
			return
		}
		reply, err := call(r.Context(), &req)
		if err != nil {
			http.Error(w, "callback failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(reply); err != nil {
			log.Print(err)
		}
	}
}

// connectHTTP selects an HTTP status only for the marked connect scenarios.
func connectHTTP(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("x-compat-connect") {
		case "busy":
			w.WriteHeader(http.StatusTooManyRequests)
		case "declined":
			w.WriteHeader(http.StatusForbidden)
		case "unavailable":
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			ctx := context.WithValue(r.Context(), connectHeadersKey{}, r.Header)
			next(w, r.WithContext(ctx))
		}
	}
}

func main() {
	f := fixture{rpcTimeouts: &rpcTimeouts{}, retryLeases: &retryLeases{}}
	listener, err := net.Listen("tcp", ":8081") //nolint:gosec // The peer runs in the same private test network. Compose publishes no fixture ports.
	if err != nil {
		log.Fatal(err)
	}
	server := grpc.NewServer(grpc.StatsHandler(connectStats{}))
	p.RegisterCentrifugoProxyServer(server, f)
	go func() { log.Fatal(server.Serve(listener)) }()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /connect", connectHTTP(jsonHandler(f.Connect)))
	mux.HandleFunc("POST /subscribe", jsonHandler(f.Subscribe))
	mux.HandleFunc("POST /refresh", jsonHandler(f.Refresh))
	mux.HandleFunc("POST /sub_refresh", jsonHandler(f.SubRefresh))
	mux.HandleFunc("POST /publish", jsonHandler(f.Publish))
	mux.HandleFunc("POST /map_publish", jsonHandler(f.MapPublish))
	mux.HandleFunc("POST /map_remove", jsonHandler(f.MapRemove))
	mux.HandleFunc("POST /rpc", jsonHandler(f.RPC))
	mux.HandleFunc("POST /marker/rpc", markerHTTP(jsonHandler(f.RPC)))
	mux.HandleFunc("POST /named/subscribe", namedHTTP("alternate_sub", jsonHandler(f.Subscribe)))
	mux.HandleFunc("POST /named/publish", namedHTTP("alternate_pub", jsonHandler(f.Publish)))
	mux.HandleFunc("POST /named/rpc", namedHTTP("alternate_rpc", jsonHandler(f.RPC)))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	httpServer := http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 3 * time.Second}
	log.Fatal(httpServer.ListenAndServe())
}
