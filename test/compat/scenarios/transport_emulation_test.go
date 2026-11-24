package scenarios

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"

	"github.com/xantos008/centrifunahuy-na-go/internal/unigrpc/unistream"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_TRANS_03(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-TRANS-03", func(s *h.Session) {
		user, ch := s.Ch("uni-grpc-user"), s.Ch("pub:uni-grpc")
		read := s.OpenUniGRPC(&unistream.ConnectRequest{Token: h.HS256(h.Claims{Sub: user, Channels: []string{ch}})})
		c := read().GetConnect()
		if c.GetClient() == "" || c.GetSession() == "" || c.GetNode() != "" || c.GetPong() || c.GetSubs()[ch] == nil {
			s.T.Fatalf("unexpected uni gRPC connect push: %v", c)
		}
		data := []byte{0, 0xff, 1, 0x80}
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"b64data":%q}`, ch, base64.StdEncoding.EncodeToString(data))})
		push := read()
		if push.GetChannel() != ch || !bytes.Equal(push.GetPub().GetData(), data) {
			s.T.Fatalf("unexpected uni gRPC binary publication: %v", push)
		}
		disconnectTransport(s, user)
		d := read().GetDisconnect()
		if d.GetCode() != 4001 || d.GetReason() != "test complete" {
			s.T.Fatalf("unexpected uni gRPC disconnect: %v", d)
		}
		if got := read(); got != nil {
			s.T.Fatalf("expected end of gRPC stream, got %v", got)
		}
	})
}

func TestCS_TRANS_04(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-TRANS-04", func(s *h.Session) {
		for _, transport := range []string{"sse", "http_stream"} {
			user, ch := s.Ch(transport+"-user"), s.Ch("pub:emulate-"+transport)
			connect := fmt.Sprintf(`{"id":1,"connect":{"token":%q}}`, h.HS256(h.Claims{Sub: user}))
			read, send := s.OpenEmulation("/connection/"+transport, connect, transport == "sse")
			reply := read()
			c := reply.GetConnect()
			if reply.GetId() != 1 || c.GetClient() == "" || c.GetSession() == "" || c.GetNode() == "" || !c.GetPong() {
				s.T.Fatalf("%s: unexpected emulation connect reply: %v", transport, reply)
			}
			send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
			if sub := read(); sub.GetId() != 2 || sub.GetSubscribe() == nil {
				s.T.Fatalf("%s: unexpected emulated subscribe reply: %v", transport, sub)
			}
			const data = `{"sequence":1}`
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":%s}`, ch, data)})
			push := read().GetPush()
			if push.GetChannel() != ch || string(push.GetPub().GetData()) != data {
				s.T.Fatalf("%s: unexpected publication: %v", transport, push)
			}
			send(fmt.Sprintf(`{"id":3,"unsubscribe":{"channel":%q}}`, ch))
			if unsub := read(); unsub.GetId() != 3 || unsub.GetUnsubscribe() == nil {
				s.T.Fatalf("%s: unexpected emulated unsubscribe reply: %v", transport, unsub)
			}
			disconnectTransport(s, user)
			d := read().GetPush().GetDisconnect()
			if d.GetCode() != 4001 || d.GetReason() != "test complete" {
				s.T.Fatalf("%s: unexpected disconnect: %v", transport, d)
			}
			if got := read(); got != nil {
				s.T.Fatalf("%s: expected end of HTTP stream, got %v", transport, got)
			}
		}
	})
}

func disconnectTransport(s *h.Session, user string) {
	s.T.Helper()
	s.API(h.APIRequest{Path: "/api/disconnect", Body: fmt.Sprintf(`{"user":%q,"disconnect":{"code":4001,"reason":"test complete"}}`, user)})
}

func TestCS_TRANS_05(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-TRANS-05", func(s *h.Session) {
		for _, path := range []string{"/connection/sockjs", "/connection/sockjs/info"} {
			s.RequireHTTPStatus(h.APIRequest{Method: http.MethodGet, Path: path, NoKey: true}, http.StatusNotFound)
		}
	})
}
