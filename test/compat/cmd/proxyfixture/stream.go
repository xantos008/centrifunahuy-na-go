package main

import (
	"strings"
	"sync"

	p "github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// One signal per active bidirectional probe, keyed by the scenario's channel
// suffix. Its companion subscription opens after the client sees the publish
// reply. This keeps the echo from racing that reply on the WebSocket.
var streamReleases sync.Map

func streamKey(r *p.SubscribeRequest, prefix string) (string, bool) {
	key, ok := strings.CutPrefix(r.GetChannel(), prefix)
	return key, ok && key != "" && r.GetUser() == "proxy-user" && r.GetClient() != "" &&
		r.GetTransport() == "websocket" && r.GetProtocol() == "json" && r.GetEncoding() == "json"
}

func streamAck(data p.Raw) *p.StreamSubscribeResponse {
	return &p.StreamSubscribeResponse{SubscribeResponse: &p.SubscribeResponse{Result: &p.SubscribeResult{Data: data}}}
}

func streamDenied() *p.StreamSubscribeResponse {
	return &p.StreamSubscribeResponse{SubscribeResponse: &p.SubscribeResponse{Error: denied()}}
}

func streamPublication(data p.Raw, source string) *p.StreamSubscribeResponse {
	return &p.StreamSubscribeResponse{Publication: &p.Publication{Data: data, Tags: map[string]string{"source": source}}}
}

func (fixture) SubscribeUnidirectional(r *p.SubscribeRequest, stream p.CentrifugoProxy_SubscribeUnidirectionalServer) error {
	key, ok := streamKey(r, "streamuni:feed-")
	if !ok {
		return stream.Send(streamDenied())
	}
	if err := stream.Send(streamAck(r.Data)); err != nil {
		return err
	}
	if err := stream.Send(streamPublication(p.Raw(`{"event":"welcome"}`), "backend")); err != nil {
		return err
	}
	if release, ok := streamReleases.LoadAndDelete(key); ok {
		close(release.(chan struct{}))
	}
	<-stream.Context().Done()
	return stream.Context().Err()
}

func (fixture) SubscribeBidirectional(stream p.CentrifugoProxy_SubscribeBidirectionalServer) error {
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	r := first.GetSubscribeRequest()
	key, ok := streamKey(r, "streambi:feed-")
	if !ok || first.Publication != nil {
		return stream.Send(streamDenied())
	}
	release := make(chan struct{})
	if _, loaded := streamReleases.LoadOrStore(key, release); loaded {
		return stream.Send(streamDenied())
	}
	defer streamReleases.CompareAndDelete(key, release)
	if err := stream.Send(streamAck(r.Data)); err != nil {
		return err
	}
	next, err := stream.Recv()
	if err != nil {
		return err
	}
	if next.Publication == nil || next.SubscribeRequest != nil {
		return status.Error(codes.InvalidArgument, "expected a publication")
	}
	select {
	case <-release:
	case <-stream.Context().Done():
		return stream.Context().Err()
	}
	if err := stream.Send(streamPublication(next.Publication.Data, "client")); err != nil {
		return err
	}
	return status.Error(codes.Unavailable, "fixture stream interrupted")
}
