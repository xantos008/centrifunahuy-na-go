package scenarios

import (
	"fmt"
	"testing"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

type apiCase struct{ method, valid, invalid string }

func apiCases(s *h.Session) []apiCase {
	news, hist, pub := s.Ch("news"), s.Ch("hist:a"), s.Ch("pub:a")
	return []apiCase{
		{"publish", fmt.Sprintf(`{"channel":%q,"data":{}}`, pub), `{"channel":""}`},
		{"broadcast", fmt.Sprintf(`{"channels":[%q,%q],"data":{}}`, pub, pub+"2"), `{"channels":[]}`},
		{"subscribe", fmt.Sprintf(`{"user":"nobody","channel":%q}`, news), `{}`},
		{"unsubscribe", fmt.Sprintf(`{"user":"nobody","channel":%q}`, news), `{}`},
		{"disconnect", `{"user":"nobody"}`, `{}`},
		{"refresh", `{"user":"nobody"}`, `{}`},
		{"presence", fmt.Sprintf(`{"channel":%q}`, news), fmt.Sprintf(`{"channel":%q}`, pub)},
		{"presence_stats", fmt.Sprintf(`{"channel":%q}`, news), fmt.Sprintf(`{"channel":%q}`, pub)},
		{"history", fmt.Sprintf(`{"channel":%q}`, hist), fmt.Sprintf(`{"channel":%q}`, news)},
		{"history_remove", fmt.Sprintf(`{"channel":%q}`, hist), `{}`},
		{"channels", fmt.Sprintf(`{"pattern":%q}`, s.Ch("none")+"*"), `{"pattern":"["}`},
		{"batch", fmt.Sprintf(`{"commands":[{"publish":{"channel":%q,"data":{}}},{"presence_stats":{"channel":%q}}]}`, pub, news), `{"commands":[{}]}`},
		{"rpc", `{"method":"no_such_rpc"}`, `{}`},
		{"map_publish", fmt.Sprintf(`{"channel":%q,"key":"k","data":{}}`, pub), `{}`},
		{"map_remove", fmt.Sprintf(`{"channel":%q,"key":"k"}`, pub), `{}`},
		{"map_read_state", fmt.Sprintf(`{"channel":%q}`, pub), `{}`},
		{"map_read_stream", fmt.Sprintf(`{"channel":%q}`, pub), `{}`},
		{"map_stats", fmt.Sprintf(`{"channel":%q}`, pub), `{}`},
		{"map_clear", fmt.Sprintf(`{"channel":%q}`, pub), `{}`},
		{"shared_poll_publish", fmt.Sprintf(`{"channel":%q,"key":"k","data":{}}`, pub), `{}`},
	}
}

// CS-API-02: every HTTP server API route with one minimal valid request and
// one invalid request. Results are recorded as returned (including errors for
// features this config does not enable, e.g. map and shared poll channels).
func TestCS_API_02(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-API-02", func(s *h.Session) {
		for _, c := range apiCases(s) {
			s.API(h.APIRequest{Path: "/api/" + c.method, Body: c.valid})
			s.API(h.APIRequest{Path: "/api/" + c.method, Body: c.invalid})
		}
		// info carries per-process numbers (counts): compare its shape only. uptime
		// is dropped: it is omitted while a node is younger than one second.
		// metrics.items keys appear as metrics are first observed, so they depend on
		// the process history and are dropped.
		s.API(h.APIRequest{Path: "/api/info", Body: `{}`, KeysOnly: true, Omit: []string{"metrics", "uptime"}})
	})
}
