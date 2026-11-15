package scenarios

import (
	"net/http"
	"testing"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// CS-API-01: server API authentication variants and HTTP-level errors.
func TestCS_API_01(t *testing.T) {
	const pub = `{"channel":"pub:api01","data":{"x":1}}`
	h.Run(t, "CS-API-01", func(s *h.Session) {
		s.API(h.APIRequest{Path: "/api/publish", Body: pub})
		s.API(h.APIRequest{Path: "/api/publish?api_key=" + h.APIKey, Body: pub, NoKey: true})
		s.API(h.APIRequest{Path: "/api/publish", Body: pub, NoKey: true, Headers: map[string]string{"Authorization": "apikey " + h.APIKey}})
		s.API(h.APIRequest{Path: "/api/publish", Body: pub, NoKey: true, Headers: map[string]string{"X-API-Key": "wrong"}})
		s.API(h.APIRequest{Path: "/api/publish", Body: pub, NoKey: true})
		s.API(h.APIRequest{Method: http.MethodGet, Path: "/api/publish"})
		s.API(h.APIRequest{Path: "/api/publish", Body: `{"channel":`})
		s.API(h.APIRequest{Path: "/api/no_such_method", Body: `{}`})
	})
}
