package scenarios

import (
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_PROXY_07(t *testing.T) {
	h.Profiles(t, "r")
	h.Run(t, "CS-PROXY-07", func(s *h.Session) {
		token := h.HS256(h.Claims{Sub: "proxy-user"})
		bidi, uni := connect(s, token), connect(s, token)
		bidiChannel, uniChannel := s.Ch("streambi:feed"), s.Ch("streamuni:feed")

		next := func(ws *h.WS) *protocol.Reply {
			s.T.Helper()
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			return &reply
		}
		subscribe := func(ws *h.WS, channel, data string) {
			s.T.Helper()
			ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"data":%s,"positioned":true,"recoverable":true}}`, channel, data))
			reply := next(ws)
			sub := reply.GetSubscribe()
			if reply.Id != 2 || reply.Error != nil || sub == nil || string(sub.Data) != data ||
				sub.Positioned || sub.Recoverable || sub.Epoch != "" || sub.Offset != 0 {
				s.T.Fatalf("unexpected stream subscribe reply on %s: %v", channel, reply)
			}
		}
		publication := func(ws *h.WS, channel, source, data string) {
			s.T.Helper()
			reply := next(ws)
			push := reply.GetPush()
			pub := push.GetPub()
			if reply.Id != 0 || push.GetChannel() != channel || pub.GetTags()["source"] != source ||
				string(pub.GetData()) != data || pub.GetOffset() != 0 {
				s.T.Fatalf("unexpected stream publication on %s: %v", channel, reply)
			}
		}
		subscribe(bidi, bidiChannel, `{"mode":"bidirectional"}`)
		const echo = `{"event":"from-client"}`
		bidi.Send(fmt.Sprintf(`{"id":3,"publish":{"channel":%q,"data":%s}}`, bidiChannel, echo))
		if reply := next(bidi); reply.Id != 3 || reply.Publish == nil || reply.Error != nil {
			s.T.Fatalf("bidirectional stream publish failed: %v", reply)
		}

		// Opening the companion stream releases the echo on the first socket.
		subscribe(uni, uniChannel, `{"mode":"unidirectional"}`)
		publication(uni, uniChannel, "backend", `{"event":"welcome"}`)
		publication(bidi, bidiChannel, "client", echo)
		reply := next(bidi)
		push := reply.GetPush()
		if reply.Id != 0 || push.GetChannel() != bidiChannel || push.GetUnsubscribe().GetCode() != 2500 ||
			push.GetUnsubscribe().GetReason() != "insufficient state" {
			s.T.Fatalf("expected insufficient state after interrupted stream: %v", reply)
		}

		uni.Send(fmt.Sprintf(`{"id":3,"publish":{"channel":%q,"data":%s}}`, uniChannel, echo))
		if reply := next(uni); reply.Id != 3 || reply.Error.GetCode() != 108 || reply.Error.GetMessage() != "not available" || reply.Publish != nil {
			s.T.Fatalf("expected publish to be unavailable on unidirectional stream: %v", reply)
		}
		for _, tc := range []struct {
			ws      *h.WS
			channel string
		}{{bidi, s.Ch("streambi:denied")}, {uni, s.Ch("streamuni:denied")}} {
			tc.ws.Send(fmt.Sprintf(`{"id":4,"subscribe":{"channel":%q}}`, tc.channel))
			if reply := next(tc.ws); reply.Id != 4 || reply.Error.GetCode() != 470 || reply.Error.GetMessage() != "fixture denied" || reply.Subscribe != nil {
				s.T.Fatalf("expected application rejection on %s: %v", tc.channel, reply)
			}
		}
	})
}
