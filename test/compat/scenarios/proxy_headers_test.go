package scenarios

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_PROXY_09(t *testing.T) {
	h.Profiles(t, "r", "p")
	h.Run(t, "CS-PROXY-09", func(s *h.Session) {
		token := h.HS256(h.Claims{Sub: "proxy-user"})
		wantTransport := "http"
		if h.Profile() == "r" {
			wantTransport = "grpc"
		}

		for _, tc := range []struct {
			frame   bool
			upgrade bool
			want    string
		}{
			{want: "static"},
			{frame: true, want: "frame"},
			{frame: true, upgrade: true, want: "upgrade"},
		} {
			headers := http.Header{}
			if tc.upgrade {
				headers.Set("X-Compat-Marker", "upgrade")
			}
			ws := s.DialWithHeaders(headers)
			frameHeaders := ""
			if tc.frame {
				frameHeaders = `,"headers":{"x-compat-marker":"frame"}`
			}
			ws.Send(fmt.Sprintf(`{"id":1,"connect":{"token":%q%s}}`, token, frameHeaders))
			ws.Recv(1)
			var connected protocol.Reply
			ws.Last(&connected)
			if connected.Id != 1 || connected.Error != nil || connected.Connect == nil {
				s.T.Fatalf("connect for %s marker: %v", tc.want, &connected)
			}
			ws.Send(`{"id":2,"rpc":{"method":"marker:read"}}`)
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			var observed struct{ Transport, Marker string }
			if reply.Id != 2 || reply.Error != nil || json.Unmarshal(reply.GetRpc().GetData(), &observed) != nil ||
				observed.Transport != wantTransport || observed.Marker != tc.want {
				s.T.Fatalf("expected %s callback with %s marker, got %v", wantTransport, tc.want, &reply)
			}
			ws.Close()
		}
	})
}
