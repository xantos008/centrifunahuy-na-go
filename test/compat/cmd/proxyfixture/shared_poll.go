package main

import (
	"context"

	p "github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
)

func (fixture) SharedPollRefresh(_ context.Context, r *p.SharedPollRefreshRequest) (*p.SharedPollRefreshResponse, error) {
	if len(r.Items) != 1 || r.Items[0].GetKey() != "quote" {
		return &p.SharedPollRefreshResponse{Error: denied()}, nil
	}
	if r.Channel == "pollplain:compat" && r.Items[0].Version == 0 {
		return &p.SharedPollRefreshResponse{Result: &p.SharedPollRefreshResult{
			Items: []*p.SharedPollRefreshResultItem{{Key: "quote", Data: p.Raw(`{"source":"versionless"}`)}},
		}}, nil
	}
	if r.Channel != "poll:compat-versioned" {
		return &p.SharedPollRefreshResponse{Error: denied()}, nil
	}
	result := &p.SharedPollRefreshResult{}
	if r.Items[0].Version == 0 {
		result.Items = []*p.SharedPollRefreshResultItem{{Key: "quote", Version: 1, Data: p.Raw(`{"source":"refresh"}`)}}
	}
	return &p.SharedPollRefreshResponse{Result: result}, nil
}
