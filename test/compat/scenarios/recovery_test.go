package scenarios

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

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
	s.AwaitDelivery()
	ws = connect(s, h.HS256(h.Claims{Sub: "u1"}))
	ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"recoverable":true,"recover":true,"epoch":%q,"offset":%d}}`,
		ch, reply.Subscribe.Epoch, reply.Subscribe.Offset))
	ws.Next()
}

// CS-REC-01: the gap fits into history: recovered=true with exactly the missed publications.
func TestCS_REC_01(t *testing.T) {
	h.Profiles(t, "m", "r", "r-lists", "rs", "rc", "rsh", "p")
	h.Run(t, "CS-REC-01", func(s *h.Session) { recoverAfter(s, s.Ch("hist:r1"), 3) })
}

// CS-REC-02: the gap is larger than history_size (10): recovered=false.
func TestCS_REC_02(t *testing.T) {
	h.Profiles(t, "m", "r", "r-lists", "rs", "rc", "rsh", "p")
	h.Run(t, "CS-REC-02", func(s *h.Session) { recoverAfter(s, s.Ch("hist:r2"), 12) })
}

// CS-REC-03: the stream epoch changes when the history store loses its
// state: node restart on the memory engine, Redis restart without
// persistence on the Redis engine. Recovery from the old position must
// report recovered=false with the new epoch.
func TestCS_REC_03(t *testing.T) {
	h.Profiles(t, "m", "r")
	h.Run(t, "CS-REC-03", func(s *h.Session) {
		ch := s.Ch("hist:r3")
		ws := connect(s, h.HS256(h.Claims{Sub: "u1"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"recoverable":true}}`, ch))
		ws.Recv(1)
		var reply struct {
			Subscribe position `json:"subscribe"`
		}
		ws.Last(&reply)
		ws.Close()
		if h.Profile() == "m" {
			s.Compose("restart", "node")
		} else {
			s.Compose("restart", "redis")
		}
		if err := h.WaitReady(s.Target, 30*time.Second); err != nil {
			s.T.Fatal(err)
		}
		// The first publish after the restart may fail while the server reconnects.
		s.APIUntil(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":1}}`, ch)},
			15*time.Second, func(b []byte) bool { return strings.Contains(string(b), `"result"`) })
		waitSubscribable(s)
		ws = connect(s, h.HS256(h.Claims{Sub: "u1"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"recoverable":true,"recover":true,"epoch":%q,"offset":%d}}`,
			ch, reply.Subscribe.Epoch, reply.Subscribe.Offset))
		ws.Next()
	})
}

// CS-REC-04: cache recovery mode (namespace cache), auto_cache_recover
// (acache) and the positioning fields of the subscribe reply (pos).
func TestCS_REC_04(t *testing.T) {
	h.Profiles(t, "m", "r", "p")
	h.Run(t, "CS-REC-04", func(s *h.Session) {
		pub := func(ch string, n int) {
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":%d}}`, ch, n)})
		}
		cache, acache, pos := s.Ch("cache:r4"), s.Ch("acache:r4"), s.Ch("pos:r4")
		pub(cache, 1)
		pub(cache, 2)
		s.AwaitDelivery()
		subscribeFresh(s, "u1", fmt.Sprintf(`{"channel":%q,"recoverable":true,"recover":true}`, cache))                        // empty since: latest only
		subscribeFresh(s, "u1", fmt.Sprintf(`{"channel":%q}`, cache))                                                          // no recovery requested
		subscribeFresh(s, "u1", fmt.Sprintf(`{"channel":%q,"recoverable":true,"recover":true}`, s.Ch("cache:r4-empty")))       // nothing cached
		subscribeFresh(s, "u1", fmt.Sprintf(`{"channel":%q,"recoverable":true,"recover":true,"offset":1,"epoch":"x"}`, cache)) // stale position
		pub(acache, 1)
		s.AwaitDelivery()
		subscribeFresh(s, "u1", fmt.Sprintf(`{"channel":%q}`, acache)) // delivered without asking
		subscribeFresh(s, "u1", fmt.Sprintf(`{"channel":%q}`, s.Ch("acache:r4-empty")))
		pub(pos, 1)
		s.AwaitDelivery()
		subscribeFresh(s, "u1", fmt.Sprintf(`{"channel":%q}`, pos))                               // force_positioning
		subscribeFresh(s, "u1", fmt.Sprintf(`{"channel":%q,"positioned":true}`, s.Ch("hist:r4"))) // not allowed in hist
	})
}

// CS-REC-05: publications racing with a resubscribe-with-recovery, 200
// iterations on one connection: every publication must arrive exactly once,
// either in the recovered list or as a push. Only the outcome is recorded.
func TestCS_REC_05(t *testing.T) {
	h.Profiles(t, "r", "p")
	h.Run(t, "CS-REC-05", func(s *h.Session) {
		const iterations, perIteration = 200, 3
		ch := s.Ch("hist:r5")
		work := &h.Session{T: s.T, Target: s.Target} // transcript discarded
		ws := connect(work, h.HS256(h.Claims{Sub: "u1"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q,"recoverable":true}}`, ch))
		ws.Recv(1)
		var sub struct {
			Subscribe position `json:"subscribe"`
		}
		ws.Last(&sub)
		pos := sub.Subscribe
		lost, dup, unrecovered := 0, 0, 0
		id := 3
		for range iterations {
			ws.Send(fmt.Sprintf(`{"id":%d,"unsubscribe":{"channel":%q}}`, id, ch))
			for { // a push arriving after all offsets were seen is a duplicate
				var m struct {
					ID   int             `json:"id"`
					Push json.RawMessage `json:"push"`
				}
				ws.Recv(1)
				ws.Last(&m)
				if m.ID == id {
					break
				}
				if m.Push != nil {
					dup++
				}
			}
			errs := make(chan error, 1)
			go func() { errs <- publishRaw(s.Target, ch, perIteration) }()
			ws.Send(fmt.Sprintf(`{"id":%d,"subscribe":{"channel":%q,"recoverable":true,"recover":true,"epoch":%q,"offset":%d}}`,
				id+1, ch, pos.Epoch, pos.Offset))
			id += 2
			seen := map[uint64]int{}
			ws.Recv(1)
			var reply struct {
				Subscribe struct {
					Recovered    bool `json:"recovered"`
					Publications []struct {
						Offset uint64 `json:"offset"`
					} `json:"publications"`
				} `json:"subscribe"`
			}
			ws.Last(&reply)
			if !reply.Subscribe.Recovered {
				unrecovered++
			}
			for _, p := range reply.Subscribe.Publications {
				seen[p.Offset]++
			}
			if err := <-errs; err != nil {
				s.T.Fatal(err)
			}
			want := pos.Offset + perIteration
			for !hasAll(seen, pos.Offset+1, want) {
				var push struct {
					Push struct {
						Pub struct {
							Offset uint64 `json:"offset"`
						} `json:"pub"`
					} `json:"push"`
				}
				ws.Recv(1) // fails the scenario after the deadline if a publication is lost
				ws.Last(&push)
				seen[push.Push.Pub.Offset]++
			}
			for o := pos.Offset + 1; o <= want; o++ {
				switch n := seen[o]; {
				case n == 0:
					lost++
				case n > 1:
					dup += n - 1
				}
			}
			pos.Offset = want
		}
		s.Note(fmt.Sprintf("%d iterations x %d publications: lost=%d duplicated=%d unrecovered=%d",
			iterations, perIteration, lost, dup, unrecovered))
		if lost+dup+unrecovered > 0 {
			s.T.Errorf("lost=%d duplicated=%d unrecovered=%d", lost, dup, unrecovered)
		}
	})
}

// waitSubscribable retries a probe subscription without recording until it
// succeeds: after a Redis restart publishing works before the pub/sub
// connection is back, and both servers answer subscribes with 3004 until then.
func waitSubscribable(s *h.Session) {
	deadline := time.Now().Add(15 * time.Second)
	for {
		work := &h.Session{T: s.T, Target: s.Target} // transcript discarded
		ws := connect(work, h.HS256(h.Claims{Sub: "u1"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, s.Ch("hist:probe")))
		ws.Next()
		ws.Close()
		ev := work.Transcript().Events
		if last := ev[len(ev)-1]; last.Kind == "ws_recv" && strings.Contains(string(last.Body), `"subscribe"`) {
			return
		}
		if time.Now().After(deadline) {
			s.T.Fatalf("%s: subscriptions still failing after a backend restart", s.Target.Name)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func hasAll(seen map[uint64]int, from, to uint64) bool {
	for o := from; o <= to; o++ {
		if seen[o] == 0 {
			return false
		}
	}
	return true
}

// publishRaw publishes n messages through the server API without recording
// (it runs in a goroutine, where t.Fatal is not allowed).
func publishRaw(tg h.Target, ch string, n int) error {
	for i := 1; i <= n; i++ {
		req, err := http.NewRequest(http.MethodPost, tg.BaseURL+"/api/publish",
			strings.NewReader(fmt.Sprintf(`{"channel":%q,"data":{"n":%d}}`, ch, i)))
		if err != nil {
			return err
		}
		req.Header.Set("X-API-Key", h.APIKey)
		resp, err := (&http.Client{Timeout: h.Deadline}).Do(req)
		if err != nil {
			return err
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if !strings.Contains(string(b), `"result"`) {
			return fmt.Errorf("publish: %s", b)
		}
	}
	return nil
}
