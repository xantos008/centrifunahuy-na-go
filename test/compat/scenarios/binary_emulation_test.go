package scenarios

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// CS-TRANS-09 checks binary emulation through the published client protocol.
func TestCS_TRANS_09(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-TRANS-09", func(s *h.Session) {
		ch := s.Ch("pub:binary-stream")
		encode := func(cmd *protocol.Command) []byte {
			s.T.Helper()
			data, err := protocol.NewProtobufCommandEncoder().Encode(cmd)
			if err != nil {
				s.T.Fatal(err)
			}
			return data
		}
		post := func(path string, data []byte) *http.Response {
			s.T.Helper()
			req, err := http.NewRequestWithContext(s.T.Context(), http.MethodPost, s.Target.BaseURL+path, bytes.NewReader(data))
			if err != nil {
				s.T.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/octet-stream")
			client := http.Client{Timeout: 3 * h.Deadline}
			res, err := client.Do(req)
			if err != nil {
				s.T.Fatal(err)
			}
			return res
		}
		ws := post("/connection/http_stream", encode(&protocol.Command{Id: 1, Connect: &protocol.ConnectRequest{Token: h.HS256(h.Claims{Sub: "binary-stream"})}}))
		defer func() { _ = ws.Body.Close() }()
		if ws.StatusCode != http.StatusOK {
			s.T.Fatalf("binary stream HTTP %d", ws.StatusCode)
		}
		r := bufio.NewReader(ws.Body)
		read := func() *protocol.Reply {
			s.T.Helper()
			n, err := binary.ReadUvarint(r)
			if err != nil || n > 65536 {
				s.T.Fatalf("invalid message length %d: %v", n, err)
			}
			data := make([]byte, n)
			if _, err := io.ReadFull(r, data); err != nil {
				s.T.Fatal(err)
			}
			var reply protocol.Reply
			if err := reply.UnmarshalCF(data); err != nil {
				s.T.Fatal(err)
			}
			return &reply
		}
		reply := read()
		conn := reply.GetConnect()
		if reply.Id != 1 || conn.GetClient() == "" || conn.GetNode() == "" || conn.GetSession() == "" {
			s.T.Fatalf("connect: %v", reply)
		}
		s.Note("binary HTTP stream connected with a client, node and session")
		send := func(cmd *protocol.Command) {
			s.T.Helper()
			envelope := &protocol.EmulationRequest{Node: conn.Node, Session: conn.Session, Data: encode(cmd)}
			data, err := envelope.MarshalCF()
			if err != nil {
				s.T.Fatal(err)
			}
			res := post("/emulation", data)
			defer func() { _ = res.Body.Close() }()
			if res.StatusCode != http.StatusNoContent {
				s.T.Fatalf("emulated command %d: HTTP %d", cmd.Id, res.StatusCode)
			}
		}
		next := func() *protocol.Reply {
			s.T.Helper()
			for {
				reply := read()
				if reply.Id != 0 || reply.Push != nil {
					return reply
				}
				send(&protocol.Command{})
			}
		}
		send(&protocol.Command{Id: 2, Subscribe: &protocol.SubscribeRequest{Channel: ch}})
		if reply := next(); reply.Id != 2 || reply.Subscribe == nil {
			s.T.Fatalf("binary subscribe: %v", reply)
		}
		s.Note("binary subscribe accepted through emulation")
		data := []byte{0, 255, 128, 1}
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"b64data":%q}`, ch, base64.StdEncoding.EncodeToString(data))})
		push := next().GetPush()
		if push.GetChannel() != ch || !bytes.Equal(push.GetPub().GetData(), data) {
			s.T.Fatalf("binary delivery: %v", push)
		}
		s.Note(fmt.Sprintf("received binary publication %x", push.Pub.Data))
		send(&protocol.Command{Id: 3, Unsubscribe: &protocol.UnsubscribeRequest{Channel: ch}})
		if reply := next(); reply.Id != 3 || reply.Unsubscribe == nil {
			s.T.Fatalf("binary unsubscribe: %v", reply)
		}
		s.Note("binary unsubscribe accepted through emulation")
	})
}
