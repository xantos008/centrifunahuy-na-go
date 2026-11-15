package scenarios

import (
	"fmt"
	"testing"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// recoverAfter subscribes with recoverable=true, disconnects, publishes n
// messages, then subscribes again with recover=true from the saved position.
func recoverAfter(s *h.Session, ch string, n int) {
	ws := connect(s, h.HS256(h.Claims{Sub: "u1"}))
	ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"recoverable":true}}`, ch))
	ws.Recv(1)
	var reply struct {
		Subscribe position `json:"subscribe"`
	}
	ws.Last(&reply)
	ws.Close()
	s.Note("client offline")
	for i := 1; i <= n; i++ {
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":%d}}`, ch, i)})
	}
	ws = connect(s, h.HS256(h.Claims{Sub: "u1"}))
	ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"recoverable":true,"recover":true,"epoch":%q,"offset":%d}}`,
		ch, reply.Subscribe.Epoch, reply.Subscribe.Offset))
	ws.Next()
}

// CS-REC-01: the gap fits into history: recovered=true with exactly the missed publications.
func TestCS_REC_01(t *testing.T) {
	h.Run(t, "CS-REC-01", func(s *h.Session) { recoverAfter(s, s.Ch("hist:r1"), 3) })
}

// CS-REC-02: the gap is larger than history_size (10): recovered=false.
func TestCS_REC_02(t *testing.T) {
	h.Run(t, "CS-REC-02", func(s *h.Session) { recoverAfter(s, s.Ch("hist:r2"), 12) })
}
