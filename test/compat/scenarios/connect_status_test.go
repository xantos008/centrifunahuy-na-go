package scenarios

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func markedConnect(marker string) string {
	return fmt.Sprintf(`{"headers":{"x-compat-connect":%q}}`, marker)
}

func TestCS_PROXY_10(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-PROXY-10", func(s *h.Session) {
		for _, tc := range []struct {
			marker  string
			code    uint32
			message string
		}{
			{"busy", 471, "try later"},
			{"unavailable", 100, "internal server error"},
		} {
			ws := s.Dial()
			ws.Send(fmt.Sprintf(`{"id":1,"connect":%s}`, markedConnect(tc.marker)))
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			e := reply.GetError()
			if reply.Id != 1 || e.GetCode() != tc.code || e.GetMessage() != tc.message || !e.GetTemporary() {
				s.T.Fatalf("%s: unexpected connect error: %v", tc.marker, &reply)
			}
			ws.Close()
		}
		ws := s.Dial()
		ws.Send(fmt.Sprintf(`{"id":1,"connect":%s}`, markedConnect("declined")))
		if code := ws.ExpectClose(); code != 4472 {
			s.T.Fatalf("unexpected callback disconnect: %d", code)
		}
	})
}

func TestCS_TRANS_08(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-TRANS-08", func(s *h.Session) {
		body := s.RequireHTTPStatus(h.APIRequest{Method: http.MethodGet, Path: "/connection/init", NoKey: true}, http.StatusOK)
		if string(body) != "{}" {
			s.T.Fatalf("unexpected connection init response: %q", body)
		}
		for _, tc := range []struct {
			marker string
			code   int
			status int
			body   string
		}{
			{"busy", 3471, http.StatusTooManyRequests, "retry this connection"},
			{"declined", 4472, http.StatusForbidden, "access declined"},
			{"unavailable", 3004, http.StatusInternalServerError, "Internal Server Error"},
		} {
			request := markedConnect(tc.marker)
			ws := s.DialUniWS(request, false)
			if code := ws.ExpectClose(); code != tc.code {
				s.T.Fatalf("%s: got WebSocket close %d, want %d", tc.marker, code, tc.code)
			}
			for _, transport := range []string{"uni_sse", "uni_http_stream"} {
				req := h.APIRequest{Path: "/connection/" + transport, Body: request, NoKey: true,
					Headers: map[string]string{"Content-Type": "application/json"}}
				if got := s.RequireHTTPStatus(req, tc.status); string(got) != tc.body {
					s.T.Fatalf("%s %s: got %q, want %q", transport, tc.marker, got, tc.body)
				}
			}
		}
	})
}
