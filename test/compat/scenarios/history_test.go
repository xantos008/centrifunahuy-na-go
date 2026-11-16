package scenarios

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

type position struct {
	Offset uint64 `json:"offset"`
	Epoch  string `json:"epoch"`
}

// CS-HIST-01: history API: position only, limit, since, reverse, oversized limit.
func TestCS_HIST_01(t *testing.T) {
	h.Profiles(t, "m", "r", "r-lists", "rs", "rc", "rsh", "p")
	h.Run(t, "CS-HIST-01", func(s *h.Session) {
		ch := s.Ch("hist:h1")
		var last struct {
			Result position `json:"result"`
		}
		for i := 1; i <= 3; i++ {
			body := s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":%d}}`, ch, i)})
			if err := json.Unmarshal(body, &last); err != nil {
				s.T.Fatal(err)
			}
		}
		hist := func(extra string) {
			s.API(h.APIRequest{Path: "/api/history", Body: fmt.Sprintf(`{"channel":%q%s}`, ch, extra)})
		}
		hist(``)            // position only
		hist(`,"limit":2`)  // first two
		hist(`,"limit":-1`) // all
		hist(fmt.Sprintf(`,"limit":10,"since":{"offset":1,"epoch":%q}`, last.Result.Epoch))
		hist(`,"limit":2,"reverse":true`)
		hist(`,"limit":1000`) // above any page limit
		hist(`,"limit":10,"since":{"offset":1,"epoch":"wrong-epoch"}`)
		s.API(h.APIRequest{Path: "/api/history_remove", Body: fmt.Sprintf(`{"channel":%q}`, ch)})
		hist(`,"limit":10`) // empty, top position kept
	})
}

// CS-HIST-02: history_ttl and history_meta_ttl expiry (namespace hmeta:
// history_ttl 1s, history_meta_ttl 5s). Publications vanish first, the
// stream position later (offset 0, new epoch). Every history read extends
// the meta TTL (centrifuge: Redis broker_history_stream.lua, memory
// historyHub), so the scenario must not poll: it waits out the TTLs. The
// memory engine counts TTLs in whole seconds and cleans up once a second, so
// each wait leaves a margin of more than one second on both sides (the documented exception to
// "no sleeps", compatibility-suite.md §5). history_remove keeping the top
// position is covered by CS-HIST-01.
func TestCS_HIST_02(t *testing.T) {
	h.Profiles(t, "m", "r", "rc", "p")
	h.Run(t, "CS-HIST-02", func(s *h.Session) {
		ch := s.Ch("hmeta:h2")
		for i := 1; i <= 2; i++ {
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":%d}}`, ch, i)})
		}
		hist := h.APIRequest{Path: "/api/history", Body: fmt.Sprintf(`{"channel":%q,"limit":10}`, ch)}
		s.API(hist)
		s.Note("wait 2.5s: history_ttl 1s passed, meta TTL 5s not")
		time.Sleep(2500 * time.Millisecond)
		s.API(hist)
		s.Note("wait 6.5s: meta TTL 5s passed since the last read")
		time.Sleep(6500 * time.Millisecond)
		s.API(hist)
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":3}}`, ch)})
	})
}

// CS-HIST-04: history and recovery publication limits, global
// (client.history_max_publication_limit / recovery_max_publication_limit = 8)
// and per namespace (hlim: 3 / 2), and permission gating of client history.
func TestCS_HIST_04(t *testing.T) {
	h.Profiles(t, "m", "r", "p")
	h.Run(t, "CS-HIST-04", func(s *h.Session) {
		clientHistory := func(ch string, n int) {
			for i := 1; i <= n; i++ {
				s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":%d}}`, ch, i)})
			}
			ws := connect(s, h.HS256(h.Claims{Sub: "u1"}))
			ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
			ws.Next()
			for i, limit := range []int{100, 5, -1, 0} {
				ws.Send(fmt.Sprintf(`{"id":%d,"history":{"channel":%q,"limit":%d}}`, 3+i, ch, limit))
				ws.Next()
			}
		}
		clientHistory(s.Ch("hist:h4"), 10) // global limit 8
		clientHistory(s.Ch("hlim:h4"), 4)  // namespace limit 3

		recoverAfter(s, s.Ch("hist:h4-8"), 8) // = global recovery limit
		recoverAfter(s, s.Ch("hist:h4-9"), 9) // above it
		recoverAfter(s, s.Ch("hlim:h4-2"), 2) // = namespace recovery limit
		recoverAfter(s, s.Ch("hlim:h4-3"), 3) // above it

		// Permission gating.
		ws := connect(s, h.HS256(h.Claims{Sub: "u1"}))
		ws.Send(fmt.Sprintf(`{"id":2,"history":{"channel":%q,"limit":10}}`, s.Ch("hist:h4"))) // not subscribed
		ws.Next()
		for i, ch := range []string{s.Ch("hnoperm:h4"), s.Ch("pub:h4")} { // no history permission; no history
			ws.Send(fmt.Sprintf(`{"id":%d,"subscribe":{"channel":%q}}`, 3+2*i, ch))
			ws.Next()
			ws.Send(fmt.Sprintf(`{"id":%d,"history":{"channel":%q,"limit":10}}`, 4+2*i, ch))
			ws.Next()
		}
	})
}

// CS-HIST-03: history-based features on a NATS broker, which has no history
// (docs: at most once PUB/SUB). The outcome is observed on ref and compared:
// API history and history_remove, client history, subscribe with recovery,
// cache recovery, auto cache recovery, forced positioning and fossil delta.
func TestCS_HIST_03(t *testing.T) {
	h.Profiles(t, "n")
	h.Run(t, "CS-HIST-03", func(s *h.Session) {
		ch := s.Ch("hist:h3")
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":1}}`, ch)})
		s.API(h.APIRequest{Path: "/api/history", Body: fmt.Sprintf(`{"channel":%q,"limit":10}`, ch)})
		s.API(h.APIRequest{Path: "/api/history_remove", Body: fmt.Sprintf(`{"channel":%q}`, ch)})

		ws := connect(s, h.HS256(h.Claims{Sub: "u1"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
		ws.Next()
		ws.Send(fmt.Sprintf(`{"id":3,"history":{"channel":%q,"limit":10}}`, ch))
		ws.Next()

		recoverAfter(s, s.Ch("hist:h3r"), 2)
		subscribeFresh(s, "u1", fmt.Sprintf(`{"channel":%q,"recoverable":true,"recover":true}`, s.Ch("cache:h3")))
		subscribeFresh(s, "u1", fmt.Sprintf(`{"channel":%q}`, s.Ch("acache:h3")))
		subscribeFresh(s, "u1", fmt.Sprintf(`{"channel":%q}`, s.Ch("pos:h3")))
		subscribeFresh(s, "u1", fmt.Sprintf(`{"channel":%q,"delta":"fossil"}`, s.Ch("delta:h3")))
	})
}
