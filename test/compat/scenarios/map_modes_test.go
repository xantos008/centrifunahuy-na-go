package scenarios

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func withMapModes(s *h.Session, run func()) {
	s.T.Helper()
	restore := s.ReplaceB("r-map-modes")
	defer func() {
		restore()
		startB(s)
	}()
	s.OnB(run)
}

func subscribeMapState(s *h.Session, ws *h.WS, id uint32, channel string, subType int) *protocol.SubscribeResult {
	s.T.Helper()
	ws.Send(fmt.Sprintf(`{"id":%d,"subscribe":{"channel":%q,"type":%d,"phase":2}}`, id, channel, subType))
	ws.Recv(1)
	var reply protocol.Reply
	ws.Last(&reply)
	sub := reply.GetSubscribe()
	if reply.Id != id || sub.GetType() != 1 || sub.Phase != 0 || sub.Cursor != "" || len(sub.Publications) != 0 {
		s.T.Fatalf("expected complete map state on %s, got %v", channel, &reply)
	}
	return sub
}

// awaitMapKeys reads the complete page. Presence keys for individual connections
// are server-generated IDs, so compare the raw keys before recording their shape.
func awaitMapKeys(s *h.Session, channel string, clientIDs bool, keys ...string) {
	s.T.Helper()
	request := h.APIRequest{Path: "/api/map_read_state", Body: fmt.Sprintf(`{"channel":%q,"limit":-1}`, channel), KeysOnly: clientIDs}
	slices.Sort(keys)
	matches := func(data []byte) bool {
		var reply struct {
			Result *struct {
				Entries []struct{ Key string }
				Cursor  string
			}
		}
		if json.Unmarshal(data, &reply) != nil || reply.Result == nil || reply.Result.Cursor != "" {
			return false
		}
		var actual []string
		for _, entry := range reply.Result.Entries {
			actual = append(actual, entry.Key)
		}
		slices.Sort(actual)
		return slices.Equal(actual, keys)
	}
	if body := s.APIUntil(request, 12*time.Second, matches); !matches(body) {
		s.T.Fatalf("%s: expected keys %v, got %s", channel, keys, body)
	}
}

func TestCS_MAP_03(t *testing.T) {
	h.Profiles(t, "r")
	h.Run(t, "CS-MAP-03", func(s *h.Session) {
		withMapModes(s, func() {
			short, long := s.Ch("shortmap:map03"), s.Ch("longmap:map03")
			defer func() {
				body := s.API(h.APIRequest{Path: "/api/map_clear", Body: fmt.Sprintf(`{"channel":%q}`, long)})
				if string(body) != `{"result":{}}` {
					s.T.Errorf("persistent map cleanup returned %s", body)
				}
			}()

			token := h.HS256(h.Claims{Sub: "map-reader"})
			ws := connect(s, token)
			for i, channel := range []string{short, long} {
				persistent := channel == long
				sub := subscribeMapState(s, ws, uint32(i+2), channel, 1)
				if len(sub.State) != 0 || sub.Recoverable != persistent {
					s.T.Fatalf("unexpected initial state on %s: %v", channel, sub)
				}
				body := s.API(h.APIRequest{Path: "/api/map_publish", Body: fmt.Sprintf(`{"channel":%q,"key":"item","data":{"v":1}}`, channel)})
				var reply struct{ Result *position }
				if err := json.Unmarshal(body, &reply); err != nil || reply.Result == nil {
					s.T.Fatalf("map_publish on %s returned %s", channel, body)
				}
				var wantOffset uint64
				if persistent {
					wantOffset = 1
				}
				if reply.Result.Offset != wantOffset || reply.Result.Epoch == "" {
					s.T.Fatalf("unexpected map_publish position on %s: %s", channel, body)
				}
				ws.Recv(1)
				var push protocol.Reply
				ws.Last(&push)
				pub := push.GetPush().GetPub()
				if push.GetPush().GetChannel() != channel || pub.GetKey() != "item" || pub.Offset != wantOffset || string(pub.Data) != `{"v":1}` {
					s.T.Fatalf("unexpected publication on %s: %v", channel, &push)
				}
			}
			ws.Close()
			awaitMapKeys(s, short, false, "item")
			awaitMapKeys(s, short, false)
			fresh := connect(s, token)
			defer fresh.Close()
			if sub := subscribeMapState(s, fresh, 2, short, 1); sub.Recoverable || len(sub.State) != 0 {
				s.T.Fatalf("expected expired ephemeral entry, got %v", sub)
			}
			sub := subscribeMapState(s, fresh, 3, long, 1)
			if !sub.Recoverable || sub.Offset != 1 || len(sub.State) != 1 || sub.State[0].Key != "item" || sub.State[0].Offset != 1 || string(sub.State[0].Data) != `{"v":1}` {
				s.T.Fatalf("unexpected persistent state after reconnect: %v", sub)
			}
		})
	})
}

func TestCS_MAP_04(t *testing.T) {
	h.Profiles(t, "r")
	h.Run(t, "CS-MAP-04", func(s *h.Session) {
		withMapModes(s, func() {
			room := s.Ch("members:map04")
			clients, users := "clientsmap:"+room, "usersmap:"+room
			observer := connect(s, h.HS256(h.Claims{Sub: "observer"}))
			defer observer.Close()
			for i, channel := range []string{clients, users} {
				sub := subscribeMapState(s, observer, uint32(i+2), channel, i+2)
				if sub.Recoverable || len(sub.State) != 0 {
					s.T.Fatalf("expected empty ephemeral presence map on %s, got %v", channel, sub)
				}
			}
			// Watch the user map live and read individual connection IDs through the API.
			observer.Send(fmt.Sprintf(`{"id":4,"unsubscribe":{"channel":%q}}`, clients))
			observer.Recv(1)
			var unsub protocol.Reply
			observer.Last(&unsub)
			if unsub.Id != 4 || unsub.GetUnsubscribe() == nil {
				s.T.Fatalf("unsubscribe from client map returned %v", &unsub)
			}
			join := func() (*h.WS, string) {
				s.T.Helper()
				ws := connect(s, h.HS256(h.Claims{Sub: "member"}))
				var connected protocol.Reply
				ws.Last(&connected)
				id := connected.GetConnect().GetClient()
				ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, room))
				ws.Recv(1)
				var reply protocol.Reply
				ws.Last(&reply)
				if id == "" || reply.Id != 2 || reply.GetSubscribe() == nil {
					s.T.Fatalf("member subscribe returned %v", &reply)
				}
				return ws, id
			}
			first, a := join()
			observer.Recv(1)
			var joined protocol.Reply
			observer.Last(&joined)
			pub := joined.GetPush().GetPub()
			if joined.GetPush().GetChannel() != users || pub.GetKey() != "member" || pub.Removed {
				s.T.Fatalf("expected user presence publication, got %v", &joined)
			}
			second, b := join()
			if a == b {
				s.T.Fatal("two connections received the same client ID")
			}
			awaitMapKeys(s, clients, true, a, b)
			awaitMapKeys(s, users, false, "member")
			first.Close()
			awaitMapKeys(s, clients, true, b)
			awaitMapKeys(s, users, false, "member")
			second.Close()
			awaitMapKeys(s, clients, true)
			awaitMapKeys(s, users, false, "member")
			awaitMapKeys(s, users, false)
		})
	})
}
