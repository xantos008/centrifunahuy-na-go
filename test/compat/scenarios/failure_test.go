package scenarios

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// Failure scenarios (phase 2, step 2.8). Backend actions address the
// session's own side only (C2), so ref and sut go through the same sequence.

// recovery is the deadline for a server to work again after its backend is
// back.
const recovery = 30 * time.Second

// subscribeOnce records one subscribe attempt on a fresh connection.
func subscribeOnce(s *h.Session, ch string) {
	s.T.Helper()
	ws := connect(s, h.HS256(h.Claims{Sub: "u1"}))
	ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
	ws.Next()
	ws.Close()
}

// subscribeUntil polls subscribe attempts in throw-away sessions until one
// succeeds or the deadline passes, then records one more attempt.
// Polling is not recorded: when the server notices the backend state is
// timing, the stable state afterwards is what is compared.
func subscribeUntil(s *h.Session, ch string) {
	s.T.Helper()
	deadline := time.Now().Add(recovery)
	for {
		work := &h.Session{T: s.T, Target: s.Target}
		ws := connect(work, h.HS256(h.Claims{Sub: "u1"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
		ws.Next()
		ws.Close()
		ev := work.Transcript().Events
		last := ev[len(ev)-1]
		succeeded := last.Kind == "ws_recv" && strings.Contains(string(last.Body), `"subscribe"`)
		if succeeded || time.Now().After(deadline) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	subscribeOnce(s, ch)
}

// delivered reports whether a fresh connection subscribed to a fresh channel
// receives an API publication within a second. Nothing is recorded.
func delivered(s *h.Session, i int) bool {
	ch := fmt.Sprintf("%s-%d", s.Ch("probe"), i)
	c, resp, err := websocket.DefaultDialer.Dial(s.Target.WSURL(), nil)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return false
	}
	defer func() { _ = c.Close() }()
	cmds := fmt.Sprintf(`{"id":1,"connect":{"token":%q}}`+"\n"+`{"id":2,"subscribe":{"channel":%q}}`, h.HS256(h.Claims{Sub: "u1"}), ch)
	if c.WriteMessage(websocket.TextMessage, []byte(cmds)) != nil {
		return false
	}
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	for got := ""; !strings.Contains(got, `"id":2`); {
		_, m, err := c.ReadMessage()
		if err != nil {
			return false
		}
		got += string(m)
		if strings.Contains(got, `"error"`) {
			return false
		}
	}
	work := &h.Session{T: s.T, Target: s.Target} // transcript discarded
	if !hasResult(work.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{}}`, ch)})) {
		return false
	}
	_ = c.SetReadDeadline(time.Now().Add(time.Second))
	_, m, err := c.ReadMessage()
	return err == nil && strings.Contains(string(m), `"push"`)
}

// awaitStableDelivery waits until delivered() has held for quiet without a
// single miss. After a Sentinel failover, from ~3.5 s to ~17.5 s, one
// delivery in about every 1.07 s is lost, on ref and sut alike; a single
// success inside that window proves nothing. 5 s without a miss cannot fit
// between the failover and the end of that window (3 s could, before the
// first loss at ~3.5 s).
func awaitStableDelivery(s *h.Session, quiet, timeout time.Duration) {
	s.T.Helper()
	deadline := time.Now().Add(timeout)
	since := time.Now()
	for i := 0; time.Since(since) < quiet; i++ {
		if time.Now().After(deadline) {
			s.T.Fatalf("%s: delivery not stable for %s within %s", s.Target.Name, quiet, timeout)
		}
		if !delivered(s, i) {
			since = time.Now()
		}
		// ~20 probes/s: unthrottled, hundreds of connections per second
		// themselves cause losses and exhaust the host's ephemeral ports.
		time.Sleep(50 * time.Millisecond)
	}
	s.Note(fmt.Sprintf("delivery stable for %s", quiet))
}

func hasResult(b []byte) bool { return strings.Contains(string(b), `"result"`) }
func hasError(b []byte) bool  { return strings.Contains(string(b), `"error"`) }

// CS-ENG-06: the broker backend is stopped: API publish and client
// subscribe fail; after it is started again both work within a deadline.
func TestCS_ENG_06(t *testing.T) {
	h.Profiles(t, "r", "n", "p")
	backend := map[string]string{"r": "redis", "n": "nats", "p": "postgres"}[h.Profile()]
	h.Run(t, "CS-ENG-06", func(s *h.Session) {
		ch := s.Ch("down")
		publish := h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":1}}`, ch)}
		s.Compose("stop", backend)
		s.APIUntil(publish, recovery, hasError)
		// Recorded once, after the server has reported the backend down:
		// on NATS the client library buffers subscribes while it
		// reconnects, so a subscribe may still succeed.
		subscribeOnce(s, ch)
		s.Compose("start", backend)
		s.APIUntil(publish, recovery, hasResult)
		subscribeUntil(s, ch)
		// A fresh channel: on PG the publish that succeeded above may still
		// be in the outbox and reach a new subscriber of ch as a push.
		ch2 := s.Ch("up")
		ws := connect(s, h.HS256(h.Claims{Sub: "u1"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch2))
		ws.Recv(1)
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":2}}`, ch2)})
		ws.Recv(1)
	})
}

// CS-ENG-07: SENTINEL FAILOVER: a replica becomes master; publishing works
// again within a deadline and the history written before the failover is
// on the new master.
func TestCS_ENG_07(t *testing.T) {
	h.Profiles(t, "rs")
	h.Run(t, "CS-ENG-07", func(s *h.Session) {
		ch := s.Ch("hist:failover")
		for i := 1; i <= 2; i++ {
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":%d}}`, ch, i)})
		}
		master := func() string {
			return strings.TrimSpace(s.Exec("sentinel", "redis-cli", "-p", "26379", "sentinel", "get-master-addr-by-name", "mymaster"))
		}
		old := master()
		s.Exec("sentinel", "redis-cli", "-p", "26379", "sentinel", "failover", "mymaster")
		s.Note("sentinel failover mymaster")
		deadline := time.Now().Add(recovery)
		for master() == old && time.Now().Before(deadline) {
			time.Sleep(200 * time.Millisecond)
		}
		if master() == old {
			s.T.Fatalf("%s: master unchanged after failover", s.Target.Name)
		}
		s.Note("new master elected")
		s.APIUntil(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":3}}`, ch)}, recovery, hasResult)
		s.API(h.APIRequest{Path: "/api/history", Body: fmt.Sprintf(`{"channel":%q,"limit":10}`, ch)})
		awaitStableDelivery(s, 5*time.Second, 60*time.Second)
		ws := connect(s, h.HS256(h.Claims{Sub: "u1"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, s.Ch("after-failover")))
		ws.Recv(1)
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":4}}`, s.Ch("after-failover"))})
		ws.Recv(1)
	})
}
