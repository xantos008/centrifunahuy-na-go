package scenarios

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// CS-MAP-01 follows a recoverable map from its initial state through live
// writes, a disconnected interval, two stream pages and a fresh state read.
func TestCS_MAP_01(t *testing.T) {
	h.Profiles(t, "r-lists")
	h.Run(t, "CS-MAP-01", func(s *h.Session) {
		channel := s.Ch("board:map01")
		token := h.HS256(h.Claims{Sub: "map-reader"})
		defer func() {
			body := s.API(h.APIRequest{Path: "/api/map_clear", Body: fmt.Sprintf(`{"channel":%q}`, channel)})
			if string(body) != `{"result":{}}` {
				s.T.Errorf("map_clear returned %s", body)
			}
		}()

		var epoch string
		write := func(key, data string, offset uint64) {
			s.T.Helper()
			method := "/api/map_remove"
			body := fmt.Sprintf(`{"channel":%q,"key":%q}`, channel, key)
			if data != "" {
				method = "/api/map_publish"
				body = fmt.Sprintf(`{"channel":%q,"key":%q,"data":%s}`, channel, key, data)
			}
			result := s.API(h.APIRequest{Path: method, Body: body})
			var reply struct{ Result position }
			if err := json.Unmarshal(result, &reply); err != nil || reply.Result.Offset != offset || reply.Result.Epoch == "" {
				s.T.Fatalf("%s at offset %d returned %s", method, offset, result)
			}
			if epoch == "" {
				epoch = reply.Result.Epoch
			} else if reply.Result.Epoch != epoch {
				s.T.Fatalf("%s changed the epoch at offset %d", method, offset)
			}
		}
		subscribe := func(ws *h.WS, id uint32, params string, phase int32, offset uint64) *protocol.SubscribeResult {
			s.T.Helper()
			ws.Send(fmt.Sprintf(`{"id":%d,"subscribe":{"channel":%q,"type":1,%s}}`, id, channel, params))
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			sub := reply.GetSubscribe()
			if reply.Id != id || reply.Error != nil || sub.GetType() != 1 || sub.Phase != phase ||
				sub.Epoch != epoch || sub.Offset != offset || sub.Cursor != "" {
				s.T.Fatalf("expected map subscribe phase %d at offset %d, got %v", phase, offset, &reply)
			}
			return sub
		}
		checkPub := func(pub *protocol.Publication, key, data string, offset uint64) {
			s.T.Helper()
			if pub.GetKey() != key || pub.Offset != offset || pub.Removed != (data == "") ||
				(data != "" && string(pub.Data) != data) {
				s.T.Fatalf("expected publication for %q at offset %d with data %q, got %v", key, offset, data, pub)
			}
		}
		push := func(ws *h.WS, key, data string, offset uint64) {
			s.T.Helper()
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			if reply.Id != 0 || reply.GetPush().GetChannel() != channel {
				s.T.Fatalf("expected map push on %s, got %v", channel, &reply)
			}
			checkPub(reply.Push.Pub, key, data, offset)
		}
		state := func(ws *h.WS, data string, offset uint64) {
			s.T.Helper()
			sub := subscribe(ws, 2, `"phase":2`, 0, offset)
			if !sub.Recoverable || sub.WasRecovering || len(sub.State) != 1 || len(sub.Publications) != 0 {
				s.T.Fatalf("unexpected initial map state: %v", sub)
			}
			checkPub(sub.State[0], "a", data, offset)
		}

		write("a", `{"v":1}`, 1)
		ws := connect(s, token)
		state(ws, `{"v":1}`, 1)
		write("b", `{"v":2}`, 2)
		push(ws, "b", `{"v":2}`, 2)
		write("b", `{"v":3}`, 3)
		push(ws, "b", `{"v":3}`, 3)
		write("b", "", 4)
		push(ws, "b", "", 4)
		ws.Close()

		write("a", `{"v":5}`, 5)
		write("a", `{"v":6}`, 6)
		reconnected := connect(s, token)
		recoverFrom := func(id uint32, since uint64, phase int32, data string) {
			s.T.Helper()
			params := fmt.Sprintf(`"phase":1,"recover":true,"offset":%d,"epoch":%q,"limit":1`, since, epoch)
			sub := subscribe(reconnected, id, params, phase, since+1)
			if len(sub.State) != 0 || len(sub.Publications) != 1 {
				s.T.Fatalf("unexpected map recovery page: %v", sub)
			}
			if phase == 0 && (!sub.Recoverable || !sub.WasRecovering || !sub.Recovered) {
				s.T.Fatalf("final page did not report recovery: %v", sub)
			}
			checkPub(sub.Publications[0], "a", data, since+1)
		}
		recoverFrom(2, 4, 1, `{"v":5}`)
		recoverFrom(3, 5, 0, `{"v":6}`)
		write("a", `{"v":7}`, 7)
		push(reconnected, "a", `{"v":7}`, 7)
		reconnected.Close()

		fresh := connect(s, token)
		state(fresh, `{"v":7}`, 7)
		fresh.Close()
	})
}
