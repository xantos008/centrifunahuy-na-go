package main

import (
	"encoding/json"

	p "github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
)

// Route options returned by the backend are independent of the client publish command.
func statePublication(r *p.PublishRequest) *p.PublishResponse {
	var request struct {
		Revision uint64 `json:"revision"`
		Epoch    string `json:"epoch"`
		Key      string `json:"key"`
	}
	if err := json.Unmarshal(r.Data, &request); err != nil {
		return &p.PublishResponse{Error: denied()}
	}
	return &p.PublishResponse{Result: &p.PublishResult{
		Data: r.Data, Delta: true, Version: request.Revision,
		VersionEpoch: request.Epoch, IdempotencyKey: request.Key,
	}}
}
