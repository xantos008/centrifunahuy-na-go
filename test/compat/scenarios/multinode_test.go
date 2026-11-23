package scenarios

import (
	"fmt"
	"testing"
	"time"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// Multi-node scenarios (phase 2, step 2.7): two server nodes per side share
// the side's backend; node A is Ref()/SUT(), node B is ref-b/sut-b.

// cleanup is the documented upper bound for a node to leave the info list.
const cleanup = 20 * time.Second

// bothNodes waits until node A and node B each list two nodes.
func bothNodes(s *h.Session) {
	s.AwaitNodes(2, cleanup)
	s.OnB(func() { s.AwaitNodes(2, cleanup) })
}

// startB starts node B again after a stop or kill and waits until it has
// rejoined, so later scenarios find two nodes.
func startB(s *h.Session) {
	s.Compose("start", "node-b")
	s.OnB(func() {
		if err := h.WaitReady(s.Target, 30*time.Second); err != nil {
			s.T.Fatal(err)
		}
	})
	bothNodes(s)
}

// CS-ENG-01: cross-node fan-out and join/leave. The subscriber is on node A;
// the client publish, the API publish and the second subscriber are on B.
func TestCS_ENG_01(t *testing.T) {
	h.Profiles(t, "r", "rs", "rc", "rsh", "n", "p")
	h.Run(t, "CS-ENG-01", func(s *h.Session) {
		bothNodes(s)
		ch, pub := s.Ch("fanout"), s.Ch("pub:fanout")
		a := connect(s, h.HS256(h.Claims{Sub: "u1"}))
		a.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"join_leave":true}}`, ch))
		a.Recv(1)
		a.Recv(1) // own join of u1
		a.Send(fmt.Sprintf(`{"id":3,"subscribe":{"channel":%q}}`, pub))
		a.Recv(1)
		if h.Profile() == "n" {
			s.AwaitNATSSubscriptions("centrifugo.client."+ch, "centrifugo.client."+pub)
		}
		s.OnB(func() {
			b := connect(s, h.HS256(h.Claims{Sub: "u2"}))
			b.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
			b.Recv(1)
			a.Recv(1) // join of u2
			b.Send(fmt.Sprintf(`{"id":3,"publish":{"channel":%q,"data":{"from":"client-b"}}}`, pub))
			b.Recv(1)
			a.Recv(1)
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"from":"api-b"}}`, ch)})
			b.RecvUntil("api-b")
			a.Recv(1)
			b.Send(fmt.Sprintf(`{"id":4,"unsubscribe":{"channel":%q}}`, ch))
			b.Recv(1)
			a.Recv(1) // leave of u2
		})
	})
}

// CS-ENG-02: presence and presence_stats across nodes, asked on both nodes.
// On p presence is the default memory manager, i.e. per node.
func TestCS_ENG_02(t *testing.T) {
	h.Profiles(t, "r", "rs", "rc", "rsh", "n", "p")
	h.Run(t, "CS-ENG-02", func(s *h.Session) {
		bothNodes(s)
		ch := s.Ch("xpresence")
		a := connect(s, h.HS256(h.Claims{Sub: "u1", Info: &h.Info{Name: "a"}}))
		a.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
		a.Recv(1)
		s.OnB(func() {
			b := connect(s, h.HS256(h.Claims{Sub: "u2", Info: &h.Info{Name: "b"}}))
			b.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
			b.Recv(1)
			b.Send(fmt.Sprintf(`{"id":3,"presence":{"channel":%q}}`, ch))
			b.RecvUntil(`"presence"`)
			s.API(h.APIRequest{Path: "/api/presence", Body: fmt.Sprintf(`{"channel":%q}`, ch)})
			s.API(h.APIRequest{Path: "/api/presence_stats", Body: fmt.Sprintf(`{"channel":%q}`, ch)})
		})
		a.Send(fmt.Sprintf(`{"id":3,"presence":{"channel":%q}}`, ch))
		a.RecvUntil(`"presence"`)
		a.Send(fmt.Sprintf(`{"id":4,"presence_stats":{"channel":%q}}`, ch))
		a.RecvUntil(`"presence_stats"`)
		s.API(h.APIRequest{Path: "/api/presence", Body: fmt.Sprintf(`{"channel":%q}`, ch)})
		s.API(h.APIRequest{Path: "/api/presence_stats", Body: fmt.Sprintf(`{"channel":%q}`, ch)})
	})
}

// CS-ENG-03: info lists both nodes; a stopped node disappears from node A's
// list within the documented cleanup.
func TestCS_ENG_03(t *testing.T) {
	h.Profiles(t, "r", "n", "p")
	h.Run(t, "CS-ENG-03", func(s *h.Session) {
		bothNodes(s)
		s.Compose("stop", "node-b")
		s.AwaitNodes(1, cleanup)
		startB(s)
	})
}

// CS-ENG-04: server API by user, called on node A, for a user connected to
// node B (= CS-API-04 across nodes).
func TestCS_ENG_04(t *testing.T) {
	h.Profiles(t, "r", "n", "p")
	h.Run(t, "CS-ENG-04", func(s *h.Session) {
		bothNodes(s)
		ch, ch2 := s.Ch("byuser"), s.Ch("byuser2")
		var b *h.WS
		s.OnB(func() {
			b = connect(s, h.HS256(h.Claims{Sub: "u4"}))
			b.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
			b.Recv(1)
		})
		s.API(h.APIRequest{Path: "/api/subscribe", Body: fmt.Sprintf(`{"user":"u4","channel":%q}`, ch2)})
		b.Recv(1)
		s.API(h.APIRequest{Path: "/api/unsubscribe", Body: fmt.Sprintf(`{"user":"u4","channel":%q}`, ch)})
		b.Recv(1)
		s.API(h.APIRequest{Path: "/api/refresh", Body: `{"user":"u4"}`})
		b.Recv(1)
		s.API(h.APIRequest{Path: "/api/disconnect", Body: `{"user":"u4","disconnect":{"code":4500,"reason":"compat"}}`})
		b.ExpectClose()
	})
}

// CS-ENG-05: history and recovery across nodes: position taken on node A,
// publications via the API of node A, history and recovery on node B.
func TestCS_ENG_05(t *testing.T) {
	h.Profiles(t, "r", "rc", "p")
	h.Run(t, "CS-ENG-05", func(s *h.Session) {
		bothNodes(s)
		ch := s.Ch("hist:xnode")
		a := connect(s, h.HS256(h.Claims{Sub: "u1"}))
		a.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"recoverable":true}}`, ch))
		a.Recv(1)
		var reply struct {
			Subscribe position `json:"subscribe"`
		}
		a.Last(&reply)
		a.Close()
		for i := 1; i <= 3; i++ {
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":%d}}`, ch, i)})
		}
		s.AwaitDelivery()
		s.OnB(func() {
			s.API(h.APIRequest{Path: "/api/history", Body: fmt.Sprintf(`{"channel":%q,"limit":10}`, ch)})
			b := connect(s, h.HS256(h.Claims{Sub: "u1"}))
			b.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"recoverable":true,"recover":true,"epoch":%q,"offset":%d}}`,
				ch, reply.Subscribe.Epoch, reply.Subscribe.Offset))
			b.Next()
		})
	})
}

// CS-PRES-02: node B crashes (docker kill): its client's presence entry
// vanishes from node A within presence_ttl (5 s) + presence_update_interval
// (2 s). The wait polls and records only the last answer.
func TestCS_PRES_02(t *testing.T) {
	h.Profiles(t, "r")
	h.Run(t, "CS-PRES-02", func(s *h.Session) {
		bothNodes(s)
		ch := s.Ch("crash")
		a := connect(s, h.HS256(h.Claims{Sub: "u1"}))
		a.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"join_leave":true}}`, ch))
		a.Recv(1)
		a.Recv(1) // own join of u1
		s.OnB(func() {
			b := connect(s, h.HS256(h.Claims{Sub: "u2"}))
			b.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
			b.Recv(1)
		})
		a.Recv(1) // join of u2
		stats := h.APIRequest{Path: "/api/presence_stats", Body: fmt.Sprintf(`{"channel":%q}`, ch)}
		s.API(stats)
		s.Compose("kill", "node-b")
		s.APIUntil(stats, 7*time.Second+3*time.Second, func(b []byte) bool {
			return string(b) == `{"result":{"num_clients":1,"num_users":1}}`
		})
		startB(s)
	})
}

// CS-ENG-03b (OQ-ENG-03): node B runs with the PostgreSQL broker but without
// the controller (configs/p-noctl). Neither node lists the other within a
// few info intervals, while publications still cross nodes via the broker.
func TestCS_ENG_03b(t *testing.T) {
	h.Profiles(t, "p")
	h.Run(t, "CS-ENG-03b", func(s *h.Session) {
		bothNodes(s)
		ch := s.Ch("noctl")
		restore := s.ReplaceB("p-noctl")
		s.AwaitNodes(1, cleanup)
		s.AwaitNodes(2, 5*time.Second)
		s.OnB(func() { s.AwaitNodes(2, 5*time.Second) })
		a := connect(s, h.HS256(h.Claims{Sub: "u1"}))
		a.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
		a.Recv(1)
		s.OnB(func() {
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"from":"b"}}`, ch)})
		})
		a.Recv(1)
		restore()
		startB(s)
	})
}
