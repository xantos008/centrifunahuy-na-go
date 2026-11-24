package scenarios

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// Use a fixed channel and non-expiring test signature so the recorded track
// command can be replayed verbatim. The last client explicitly untracks.
func TestCS_POLL_01(t *testing.T) {
	h.Profiles(t, "r")
	h.Run(t, "CS-POLL-01", func(s *h.Session) {
		const channel, key, user = "poll:compat-versioned", "quote", "poll-reader"
		signature := pollSignature(user, channel, key)
		open := func() *h.WS {
			s.T.Helper()
			ws := connect(s, h.HS256(h.Claims{Sub: user}))
			ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"type":4}}`, channel))
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			if reply.Id != 2 || reply.GetSubscribe().GetType() != 4 || reply.Subscribe.Recoverable {
				s.T.Fatalf("unexpected shared poll subscription: %v", &reply)
			}
			return ws
		}
		track := func(ws *h.WS, id int, sig string, version int) {
			ws.Send(fmt.Sprintf(`{"id":%d,"sub_refresh":{"channel":%q,"type":1,"track":[{"signature":%q,"items":[{"key":%q,"version":%d}]}]}}`,
				id, channel, sig, key, version))
			ws.Recv(1)
		}
		checkRefresh := func(ws *h.WS, id uint32) {
			s.T.Helper()
			var reply protocol.Reply
			ws.Last(&reply)
			refresh := reply.GetSubRefresh()
			if reply.Id != id || reply.Error != nil || refresh == nil || refresh.Expires || len(refresh.Items) != 0 {
				s.T.Fatalf("unexpected poll refresh acknowledgement: %v", &reply)
			}
		}
		push := func(ws *h.WS, version uint64, data string) {
			s.T.Helper()
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			pub := reply.GetPush().GetPub()
			if reply.Id != 0 || reply.GetPush().GetChannel() != channel ||
				pub.GetKey() != key || pub.GetVersion() != version || string(pub.GetData()) != data {
				s.T.Fatalf("expected %s version %d with %s, got %v", key, version, data, &reply)
			}
		}
		publish := func(version int, data string) {
			s.T.Helper()
			body := fmt.Sprintf(`{"channel":%q,"key":%q,"version":%d,"data":%s}`, channel, key, version, data)
			result := s.API(h.APIRequest{Path: "/api/shared_poll_publish", Body: body})
			if string(result) != `{"result":{}}` {
				s.T.Fatalf("shared poll publish returned %s", result)
			}
		}

		first := open()
		track(first, 3, "invalid", 0)
		var denied protocol.Reply
		first.Last(&denied)
		if denied.Id != 3 || denied.GetError().GetCode() != 103 {
			s.T.Fatalf("invalid signature returned %v", &denied)
		}
		track(first, 4, signature, 0)
		checkRefresh(first, 4)
		push(first, 1, `{"source":"refresh"}`)
		s.OnB(func() { publish(2, `{"source":"direct"}`) })
		push(first, 2, `{"source":"direct"}`)
		first.Close()

		// Move the socket to the other node and replay the signed keys with
		// the last seen version. A duplicate update must precede the newer
		// one on the broker and be discarded before reaching the client.
		var reconnected *h.WS
		s.OnB(func() {
			reconnected = open()
			track(reconnected, 3, signature, 2)
			checkRefresh(reconnected, 3)
		})
		defer reconnected.Close()
		publish(2, `{"source":"duplicate"}`)
		publish(3, `{"source":"next"}`)
		push(reconnected, 3, `{"source":"next"}`)
		reconnected.Send(fmt.Sprintf(`{"id":4,"sub_refresh":{"channel":%q,"type":2,"untrack":[%q]}}`, channel, key))
		reconnected.Recv(1)
		checkRefresh(reconnected, 4)
	})
}

func pollSignature(user, channel, key string) string {
	mac := hmac.New(sha256.New, []byte("compat-test-poll-secret"))
	_, _ = fmt.Fprintf(mac, "1\x000\x00%s\x00%s\x00%x", user, channel, sha256.Sum256([]byte(key)))
	return fmt.Sprintf("1:0:%x", mac.Sum(nil))
}

func TestCS_POLL_02(t *testing.T) {
	h.Profiles(t, "r")
	h.Run(t, "CS-POLL-02", func(s *h.Session) {
		const ch, user, key = "pollplain:compat", "plain-reader", "quote"
		ws := connect(s, h.HS256(h.Claims{Sub: user}))
		defer ws.Close()
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"type":4}}`, ch))
		ws.Recv(1)
		var subscribed protocol.Reply
		ws.Last(&subscribed)
		if subscribed.Id != 2 || subscribed.GetSubscribe().GetType() != 4 {
			s.T.Fatalf("versionless subscribe: %v", &subscribed)
		}
		ws.Send(fmt.Sprintf(`{"id":3,"sub_refresh":{"channel":%q,"type":1,"track":[{"signature":%q,"items":[{"key":%q}]}]}}`, ch, pollSignature(user, ch, key), key))
		ws.Recv(1)
		var ack protocol.Reply
		ws.Last(&ack)
		if ack.Id != 3 || ack.SubRefresh == nil {
			s.T.Fatalf("versionless track: %v", &ack)
		}
		ws.Recv(1)
		var delivered protocol.Reply
		ws.Last(&delivered)
		push := delivered.GetPush()
		pub := push.GetPub()
		if push.GetChannel() != ch || pub.GetKey() != key || pub.GetVersion() == 0 || string(pub.GetData()) != `{"source":"versionless"}` {
			s.T.Fatalf("expected a server-assigned version for callback data: %v", &delivered)
		}
		body := s.API(h.APIRequest{Path: "/api/shared_poll_publish", Body: fmt.Sprintf(`{"channel":%q,"key":%q,"data":true,"version":1}`, ch, key)})
		var response struct{ Error struct{ Code int } }
		if err := json.Unmarshal(body, &response); err != nil || response.Error.Code != 108 {
			s.T.Fatalf("versionless direct publish: %s", body)
		}
		ws.Send(fmt.Sprintf(`{"id":4,"sub_refresh":{"channel":%q,"type":2,"untrack":[%q]}}`, ch, key))
		ws.Recv(1)
		var removed protocol.Reply
		ws.Last(&removed)
		if removed.Id != 4 || removed.SubRefresh == nil {
			s.T.Fatalf("versionless untrack: %v", &removed)
		}
	})
}
