package scenarios

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func proxyMember(s *h.Session, channel, client, connInfo, chanInfo string) {
	s.T.Helper()
	body := s.API(h.APIRequest{Path: "/api/presence", Body: fmt.Sprintf(`{"channel":%q}`, channel)})
	var reply struct {
		Result struct {
			Presence map[string]*protocol.ClientInfo `json:"presence"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &reply); err != nil {
		s.T.Fatal(err)
	}
	member := reply.Result.Presence[client]
	if len(reply.Result.Presence) != 1 || member.GetClient() != client || member.GetUser() != "proxy-user" ||
		string(member.GetConnInfo()) != connInfo || string(member.GetChanInfo()) != chanInfo {
		s.T.Fatalf("unexpected presence for %s: %s", channel, body)
	}
}

func TestCS_PROXY_01(t *testing.T) {
	h.Profiles(t, "m", "r", "p")
	h.Run(t, "CS-PROXY-01", func(s *h.Session) {
		// Both transports exercise both override values on fresh channels.
		// Neither connect request includes a subscription or a previous position.
		for _, recoverCache := range []bool{false, true} {
			channel := s.Ch(fmt.Sprintf("pcache:proxy-%t", recoverCache))
			for n := 1; n <= 2; n++ {
				s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":%d}}`, channel, n)})
			}
			s.AwaitDelivery()
			request := fmt.Sprintf(`{"name":"proxy-compat","version":"1","data":{"channel":%q,"recover":%t}}`, channel, recoverCache)
			ws := s.Dial()
			ws.Send(fmt.Sprintf(`{"id":1,"connect":%s}`, request))
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			checkProxyRecovery(s, reply.GetConnect().GetSubs()[channel], recoverCache)
			proxyMember(s, channel, reply.GetConnect().GetClient(), `{"source":"connect"}`, `{"source":"connect-sub"}`)
			ws.Close()

			next := s.OpenHTTPStream("/connection/uni_http_stream", request, false)
			checkProxyRecovery(s, next().GetConnect().GetSubs()[channel], recoverCache)
		}
	})
}

func checkProxyRecovery(s *h.Session, sub *protocol.SubscribeResult, recoverCache bool) {
	s.T.Helper()
	if sub == nil || !sub.Recoverable || sub.WasRecovering != recoverCache {
		s.T.Fatalf("cache override %t: unexpected subscription: %v", recoverCache, sub)
	}
	if !recoverCache {
		if len(sub.Publications) != 0 {
			s.T.Fatalf("unexpected cached publications: %v", sub.Publications)
		}
		return
	}
	if len(sub.Publications) != 1 {
		s.T.Fatalf("expected one cached publication: %v", sub.Publications)
	}
	pub := sub.Publications[0]
	var payload struct{ N int }
	if err := json.Unmarshal(pub.GetData(), &payload); err != nil || payload.N != 2 || pub.GetOffset() != 2 {
		s.T.Fatalf("unexpected cached publication: %v", pub)
	}
}

func TestCS_PROXY_02(t *testing.T) {
	h.Profiles(t, "m", "r", "p")
	h.Run(t, "CS-PROXY-02", func(s *h.Session) {
		ws := connect(s, h.HS256(h.Claims{Sub: "proxy-user", Info: &h.Info{Name: "jwt"}}))
		var connected protocol.Reply
		ws.Last(&connected)
		call := func(frame string) *protocol.Reply {
			s.T.Helper()
			ws.Send(frame)
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			return &reply
		}
		channel := s.Ch("proxied:allowed")
		const data = `{"probe":true}`
		subscribed := call(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"data":%s}}`, channel, data))
		if subscribed.Id != 2 || string(subscribed.GetSubscribe().GetData()) != data {
			s.T.Fatalf("unexpected subscribe callback result: %v", subscribed)
		}
		proxyMember(s, channel, connected.GetConnect().GetClient(), `{"name":"jwt"}`, `{"source":"subscribe"}`)
		reply := call(fmt.Sprintf(`{"id":3,"subscribe":{"channel":%q}}`, s.Ch("proxied:denied")))
		if reply.Error.GetCode() != 470 || reply.Error.GetMessage() != "fixture denied" {
			s.T.Fatalf("unexpected subscribe proxy error: %v", reply)
		}
	})
}

func TestCS_PROXY_03(t *testing.T) {
	h.Profiles(t, "m", "r", "p")
	h.Run(t, "CS-PROXY-03", func(s *h.Session) {
		channel := s.Ch("proxywrite:delivery")
		reader := connect(s, h.HS256(h.Claims{Sub: "reader"}))
		reader.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, channel))
		reader.Recv(1)
		writer := connect(s, h.HS256(h.Claims{Sub: "proxy-user"}))
		for i, action := range []string{"retain", "live", "reject"} {
			writer.Send(fmt.Sprintf(`{"id":%d,"publish":{"channel":%q,"data":%q}}`, i+2, channel, action))
			writer.Recv(1)
			var ack protocol.Reply
			writer.Last(&ack)
			if action == "reject" {
				if ack.Error.GetCode() != 470 {
					s.T.Fatalf("expected publish proxy rejection: %v", &ack)
				}
				continue
			}
			if ack.Publish == nil {
				s.T.Fatalf("publish failed: %v", &ack)
			}
			reader.Recv(1)
			var delivery protocol.Reply
			reader.Last(&delivery)
			pub := delivery.GetPush().GetPub()
			if pub.GetTags()["delivery"] != action || !proxyPublication(pub) {
				s.T.Fatalf("unexpected proxy publication: %v", &delivery)
			}
		}
		body := s.API(h.APIRequest{Path: "/api/history", Body: fmt.Sprintf(`{"channel":%q,"limit":4}`, channel)})
		var reply struct {
			Result struct {
				Publications []*protocol.Publication `json:"publications"`
			} `json:"result"`
		}
		if err := json.Unmarshal(body, &reply); err != nil {
			s.T.Fatal(err)
		}
		stored := reply.Result.Publications
		want := []string{"retain"}
		if h.Profile() == "p" {
			// The PostgreSQL stream broker persists even skip_history publications.
			want = append(want, "live")
		}
		if len(stored) != len(want) {
			s.T.Fatalf("unexpected proxy history: %s", body)
		}
		for i, action := range want {
			if stored[i].GetOffset() != uint64(i+1) || stored[i].GetTags()["delivery"] != action || !proxyPublication(stored[i]) {
				s.T.Fatalf("unexpected publication at index %d: %s", i, body)
			}
		}
	})
}

func proxyPublication(pub *protocol.Publication) bool {
	var data struct{ Source string }
	return json.Unmarshal(pub.GetData(), &data) == nil && data.Source == "proxy"
}

func TestCS_PROXY_12(t *testing.T) {
	h.Profiles(t, "m", "r")
	h.Run(t, "CS-PROXY-12", func(s *h.Session) {
		const user = "proxy-user"
		ws := connect(s, h.HS256(h.Claims{Sub: user, Info: &h.Info{Name: "connection"}}))
		var connected protocol.Reply
		ws.Last(&connected)
		client := connected.GetConnect().GetClient()

		// All three channel names match the ordinary callback, which echoes
		// request data and supplies its own channel info when selected.
		signed := s.Ch("proxied:allowed-signed")
		token := h.HS256(h.Claims{Sub: user, Channel: signed, Info: &h.Info{Name: "subscription"}})
		const probe = `"route-probe"`
		for i, tc := range []struct{ channel, token, replyData, chanInfo string }{
			{signed, token, "", `{"name":"subscription"}`},
			{s.Ch("proxied:allowed-personal") + "#" + user, "", "", ""},
			{s.Ch("proxied:allowed-control"), "", probe, `{"source":"subscribe"}`},
		} {
			id := uint32(i + 2)
			cmd := protocol.Command{Id: id, Subscribe: &protocol.SubscribeRequest{
				Channel: tc.channel, Token: tc.token, Data: protocol.Raw(probe),
			}}
			frame, err := json.Marshal(&cmd)
			if err != nil {
				s.T.Fatal(err)
			}
			ws.Send(string(frame))
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			if reply.Id != id || reply.Error != nil || reply.Subscribe == nil || string(reply.Subscribe.Data) != tc.replyData {
				s.T.Fatalf("unexpected subscribe reply on %s: %v", tc.channel, &reply)
			}
			proxyMember(s, tc.channel, client, `{"name":"connection"}`, tc.chanInfo)
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":"delivery"}`, tc.channel)})
			ws.Recv(1)
			var delivery protocol.Reply
			ws.Last(&delivery)
			push := delivery.GetPush()
			if delivery.Id != 0 || push.GetChannel() != tc.channel || string(push.GetPub().GetData()) != `"delivery"` {
				s.T.Fatalf("missing publication on %s: %v", tc.channel, &delivery)
			}
		}
	})
}
