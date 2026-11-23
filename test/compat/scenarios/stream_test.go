package scenarios

import (
	"fmt"
	"testing"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_TRANS_01(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-TRANS-01", func(s *h.Session) {
		for _, transport := range []struct {
			name string
			sse  bool
		}{
			{"uni_sse", true},
			{"uni_http_stream", false},
		} {
			channel := s.Ch("pub:" + transport.name)
			token := h.HS256(h.Claims{Sub: "u1"})
			request := fmt.Sprintf(`{"token":%q,"subs":{%q:{}}}`, token, channel)
			next := s.OpenHTTPStream("/connection/"+transport.name, request, transport.sse)
			connected := next().Connect
			if connected == nil || connected.Client == "" || connected.Pong || connected.Subs[channel] == nil {
				s.T.Fatalf("%s: expected a unidirectional connect with the requested subscription", transport.name)
			}
			const payload = `{"probe":true}`
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":%s}`, channel, payload)})
			delivered := next()
			if delivered.Channel != channel || delivered.Pub == nil || string(delivered.Pub.Data) != payload {
				s.T.Fatalf("%s: expected the API publication on %s, got %+v", transport.name, channel, delivered)
			}
		}
	})
}
