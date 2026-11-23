package scenarios

import (
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// The hmeta namespace expires its stream metadata after five idle seconds.
// A positioned server-side subscriber must reconnect when the epoch changes.
func TestCS_REC_07(t *testing.T) {
	h.Profiles(t, "m", "r", "p")
	h.Run(t, "CS-REC-07", func(s *h.Session) {
		channel, user := s.Ch("hmeta:position-loss"), s.Ch("position-user")
		ws := connect(s, h.HS256(h.Claims{Sub: user}))
		s.API(h.APIRequest{Path: "/api/subscribe", Body: fmt.Sprintf(
			`{"user":%q,"channel":%q,"override":{"force_positioning":{"value":true}}}`, user, channel)})
		ws.Recv(1)
		var subscribed protocol.Reply
		ws.Last(&subscribed)
		if !subscribed.GetPush().GetSubscribe().GetPositioned() {
			s.T.Fatalf("expected positioned server-side subscription: %v", &subscribed)
		}
		publish := func(n int) {
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":%d}}`, channel, n)})
		}
		publish(1)
		ws.Recv(1)
		var delivered protocol.Reply
		ws.Last(&delivered)
		if delivered.GetPush().GetPub().GetOffset() != 1 {
			s.T.Fatalf("expected initial publication at offset 1: %v", &delivered)
		}
		// Each profile pings every two seconds. Five pings span at least
		// eight seconds, including the margin for whole-second TTL cleanup.
		// Reading history to poll for expiry would renew the metadata TTL.
		for range 5 {
			ws.RecvPing(true)
		}
		publish(2)
		if code := ws.ExpectClose(); code != 3010 {
			s.T.Fatalf("expected insufficient state after epoch change, got %d", code)
		}
	})
}
