package main

import (
	"context"
	"encoding/base64"
	"strings"

	p "github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
)

func mapWriteAllowed(user, client, transport, protocol, encoding, channel, key string, meta p.Raw) bool {
	return user == "map-writer" && client != "" && transport == "websocket" && protocol == "json" &&
		encoding == "binary" && strings.HasPrefix(channel, "mapproxy:") && key == "input" && string(meta) == `{"name":"map-meta"}`
}

func (fixture) MapPublish(_ context.Context, req *p.MapPublishRequest) (*p.MapPublishResponse, error) {
	data, err := base64.StdEncoding.DecodeString(req.B64Data)
	if !mapWriteAllowed(req.User, req.Client, req.Transport, req.Protocol, req.Encoding, req.Channel, req.Key, req.Meta) ||
		err != nil || len(req.Data) != 0 || string(data) != `"client"` {
		return &p.MapPublishResponse{Error: denied()}, nil
	}
	return &p.MapPublishResponse{Result: &p.MapPublishResult{
		Key: "stored", B64Data: base64.StdEncoding.EncodeToString([]byte(`"callback"`)), Tags: map[string]string{"action": "put"},
	}}, nil
}

func (fixture) MapRemove(_ context.Context, req *p.MapRemoveRequest) (*p.MapRemoveResponse, error) {
	if !mapWriteAllowed(req.User, req.Client, req.Transport, req.Protocol, req.Encoding, req.Channel, req.Key, req.Meta) {
		return &p.MapRemoveResponse{Error: denied()}, nil
	}
	return &p.MapRemoveResponse{Result: &p.MapRemoveResult{Key: "stored", Tags: map[string]string{"action": "remove"}}}, nil
}
