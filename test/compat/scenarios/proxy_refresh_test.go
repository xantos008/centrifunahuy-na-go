package scenarios

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func leaseClient(s *h.Session, claims h.Claims) (*h.WS, string) {
	s.T.Helper()
	claims.Meta = &h.Info{Name: "lease"}
	ws := connect(s, h.HS256(claims))
	var reply protocol.Reply
	ws.Last(&reply)
	result := reply.GetConnect()
	if result.GetClient() == "" || result.GetExpires() || result.GetTtl() != 0 {
		s.T.Fatalf("expected server-managed connection: %v", &reply)
	}
	return ws, result.Client
}

// Keep answering WebSocket pings while the presence manager picks up the
// callback's new metadata. Record just the observed state and its delivery.
func awaitLeaseInfo(s *h.Session, ws *h.WS, channel, client, user, connInfo, chanInfo string) {
	s.T.Helper()
	const marker = "lease-refreshed"
	finish := ws.CaptureUntil(marker, 20*time.Second)
	matches := func(body []byte) bool {
		var reply struct {
			Result struct {
				Presence map[string]*protocol.ClientInfo `json:"presence"`
			} `json:"result"`
		}
		if json.Unmarshal(body, &reply) != nil {
			return false
		}
		member := reply.Result.Presence[client]
		return len(reply.Result.Presence) == 1 && member.GetUser() == user && member.GetClient() == client &&
			string(member.GetConnInfo()) == connInfo && string(member.GetChanInfo()) == chanInfo
	}
	body := s.APIUntil(h.APIRequest{Path: "/api/presence", Body: fmt.Sprintf(`{"channel":%q}`, channel)}, 15*time.Second, matches)
	if !matches(body) {
		s.T.Fatalf("timed out waiting for refreshed presence on %s: %s", channel, body)
	}
	s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":%q}`, channel, marker)})
	finish()
	var publication protocol.Reply
	ws.Last(&publication)
	push := publication.GetPush()
	if push.GetChannel() != channel || string(push.GetPub().GetData()) != `"lease-refreshed"` {
		s.T.Fatalf("expected a publication after renewal: %v", &publication)
	}
}

func TestCS_PROXY_04(t *testing.T) {
	h.Profiles(t, "r", "p")
	h.Run(t, "CS-PROXY-04", func(s *h.Session) {
		channel := s.Ch("connection-lease")
		ws, client := leaseClient(s, h.Claims{Sub: "lease-renew", Exp: h.ExpIn(2 * time.Second),
			Info: &h.Info{Name: "initial"}, Channels: []string{channel}})
		awaitLeaseInfo(s, ws, channel, client, "lease-renew", `{"source":"refresh"}`, "")
		ws.Close()

		expiring, _ := leaseClient(s, h.Claims{Sub: "lease-expire", Exp: h.ExpIn(2 * time.Second)})
		if code := expiring.ExpectClose(); code != 3005 {
			s.T.Fatalf("expected connection expired (3005), got %d", code)
		}
	})
}

func subscribeLease(s *h.Session, ws *h.WS, id int, channel string) {
	s.T.Helper()
	ws.Send(fmt.Sprintf(`{"id":%d,"subscribe":{"channel":%q}}`, id, channel))
	ws.Recv(1)
	var reply protocol.Reply
	ws.Last(&reply)
	sub := reply.GetSubscribe()
	if int(reply.Id) != id || sub == nil || sub.Expires || sub.Ttl != 0 {
		s.T.Fatalf("expected server-managed subscription to %s: %v", channel, &reply)
	}
}

func TestCS_PROXY_05(t *testing.T) {
	h.Profiles(t, "r", "p")
	h.Run(t, "CS-PROXY-05", func(s *h.Session) {
		ws, client := leaseClient(s, h.Claims{Sub: "proxy-user", Info: &h.Info{Name: "initial"}})
		renewed, expired := s.Ch("lease:renew"), s.Ch("lease:expire")
		subscribeLease(s, ws, 2, renewed)
		awaitLeaseInfo(s, ws, renewed, client, "proxy-user", `{"name":"initial"}`, `{"source":"sub_refresh"}`)
		subscribeLease(s, ws, 3, expired)
		ws.CaptureUntil("subscription expired", 15*time.Second)()
		var reply protocol.Reply
		ws.Last(&reply)
		push := reply.GetPush()
		if push.GetChannel() != expired || push.GetUnsubscribe().GetCode() != 2501 ||
			push.GetUnsubscribe().GetReason() != "subscription expired" {
			s.T.Fatalf("expected subscription expired (2501) on %s: %v", expired, &reply)
		}
		// The other subscription still receives publications on this connection.
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":"still-subscribed"}`, renewed)})
		ws.Recv(1)
		var delivery protocol.Reply
		ws.Last(&delivery)
		if delivery.GetPush().GetChannel() != renewed || string(delivery.GetPush().GetPub().GetData()) != `"still-subscribed"` {
			s.T.Fatalf("missing delivery on renewed subscription: %v", &delivery)
		}
	})
}
