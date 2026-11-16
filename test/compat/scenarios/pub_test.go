package scenarios

import (
	"fmt"
	"strings"
	"testing"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// CS-PUB-01: client publish permissions and server API publish options.
// Publisher and subscriber are separate connections, so reply and push order
// never interleave on one socket.
func TestCS_PUB_01(t *testing.T) {
	h.Profiles(t, "m", "r", "rs", "rc", "rsh", "n", "p")
	h.Run(t, "CS-PUB-01", func(s *h.Session) {
		ch := s.Ch("pub:p1")
		sub := connect(s, h.HS256(h.Claims{Sub: "u2"}))
		sub.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
		sub.Recv(1)

		pub := connect(s, h.HS256(h.Claims{Sub: "u1"}))
		pub.Send(fmt.Sprintf(`{"id":2,"publish":{"channel":%q,"data":{"n":1}}}`, ch))
		pub.Recv(1)
		sub.Recv(1)
		pub.Send(fmt.Sprintf(`{"id":3,"publish":{"channel":%q,"data":{"n":2}}}`, s.Ch("news"))) // no publish permission
		pub.Next()

		hist := s.Ch("hist:p1")
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":3},"tags":{"k":"v"},"idempotency_key":"ik-1"}`, hist)})
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":3},"tags":{"k":"v"},"idempotency_key":"ik-1"}`, hist)})
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":4},"skip_history":true}`, hist)})
		s.API(h.APIRequest{Path: "/api/history", Body: fmt.Sprintf(`{"channel":%q,"limit":10}`, hist)})
	})
}

// subscribeOn connects as u1 and subscribes with the given request body,
// recording the reply.
func subscribeOn(s *h.Session, req string) *h.WS {
	ws := connect(s, h.HS256(h.Claims{Sub: "u1"}))
	ws.Send(`{"id":2,"subscribe":` + req + `}`)
	ws.Next()
	return ws
}

// CS-PUB-03: fossil delta compression. Namespace delta allows fossil and
// forces positioning; deltap also sets delta_publish. The fossil delta of a
// given pair of payloads is deterministic, so pushes are compared byte-exact.
func TestCS_PUB_03(t *testing.T) {
	h.Profiles(t, "m", "r", "rc", "p")
	h.Run(t, "CS-PUB-03", func(s *h.Session) {
		pub := func(ch, text, extra string) {
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"text":%q}%s}`, ch, text, extra)})
		}
		const long = "the quick brown fox jumps over the lazy dog, again and again"
		ch := s.Ch("delta:p3")
		d := subscribeOn(s, fmt.Sprintf(`{"channel":%q,"delta":"fossil"}`, ch))
		plain := subscribeOn(s, fmt.Sprintf(`{"channel":%q}`, ch))
		pub(ch, long+" 1", `,"delta":true`)
		pub(ch, long+" 2", `,"delta":true`)
		pub(ch, long+" 3", ``) // no delta flag
		d.Recv(3)
		plain.Recv(3)
		s.API(h.APIRequest{Path: "/api/history", Body: fmt.Sprintf(`{"channel":%q,"limit":10}`, ch)}) // never delta

		chp := s.Ch("deltap:p3")
		dp := subscribeOn(s, fmt.Sprintf(`{"channel":%q,"delta":"fossil"}`, chp))
		pub(chp, long+" 1", ``)
		pub(chp, long+" 2", ``) // delta_publish
		dp.Recv(2)

		subscribeOn(s, fmt.Sprintf(`{"channel":%q,"delta":"fossil"}`, s.Ch("hist:p3"))) // fossil not allowed
		subscribeOn(s, fmt.Sprintf(`{"channel":%q,"delta":"zstd"}`, s.Ch("delta:p3b"))) // unknown type
	})
}

// CS-PUB-04: publication tags filter (tf) in namespace tags: eq, in, and,
// or, no filter; invalid filter; filter where it is not allowed.
func TestCS_PUB_04(t *testing.T) {
	h.Profiles(t, "m", "r", "n", "p")
	h.Run(t, "CS-PUB-04", func(s *h.Session) {
		ch := s.Ch("tags:p4")
		subs := []struct {
			tf   string
			want int
		}{
			{`{"key":"k","cmp":"eq","val":"a"}`, 1},
			{`{"key":"k","cmp":"in","vals":["a","c"]}`, 2},
			{`{"op":"and","nodes":[{"key":"k","cmp":"neq","val":"a"},{"key":"n","cmp":"gte","val":"2"}]}`, 2},
			{`{"op":"or","nodes":[{"key":"k","cmp":"eq","val":"b"},{"key":"n","cmp":"lt","val":"2"}]}`, 2},
			{``, 3},
		}
		var wss []*h.WS
		for _, sub := range subs {
			req := fmt.Sprintf(`{"channel":%q}`, ch)
			if sub.tf != "" {
				req = fmt.Sprintf(`{"channel":%q,"tf":%s}`, ch, sub.tf)
			}
			wss = append(wss, subscribeOn(s, req))
		}
		for i, k := range []string{"a", "b", "c"} {
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"k":%q},"tags":{"k":%q,"n":"%d"}}`, ch, k, k, i+1)})
		}
		for i, sub := range subs {
			wss[i].Recv(sub.want)
		}
		subscribeOn(s, fmt.Sprintf(`{"channel":%q,"tf":{"key":"k","cmp":"in","vals":[]}}`, s.Ch("tags:p4b"))) // invalid
		subscribeOn(s, fmt.Sprintf(`{"channel":%q,"tf":{"key":"k","cmp":"eq","val":"a"}}`, s.Ch("pub:p4")))   // not allowed
	})
}

// CS-PUB-05: version / version_epoch suppression and idempotency_key in
// namespace ver (history on). The subscriber reads until a final marker
// publication, so suppressed publications show as missing pushes.
func TestCS_PUB_05(t *testing.T) {
	h.Profiles(t, "m", "r", "n", "p")
	h.Run(t, "CS-PUB-05", func(s *h.Session) {
		ch := s.Ch("ver:p5")
		ws := subscribeOn(s, fmt.Sprintf(`{"channel":%q}`, ch))
		for i, extra := range []string{
			`,"version":1,"version_epoch":"e1"`,
			`,"version":2,"version_epoch":"e1"`,
			`,"version":2,"version_epoch":"e1"`, // same version: suppressed
			`,"version":1,"version_epoch":"e1"`, // older: suppressed
			`,"version":1,"version_epoch":"e2"`, // new epoch: accepted
			`,"idempotency_key":"p5"`,
			`,"idempotency_key":"p5"`, // repeated key
		} {
			s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":%d}%s}`, ch, i+1, extra)})
		}
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"marker":"end"}}`, ch)})
		ws.RecvUntil(`"marker"`)
		s.API(h.APIRequest{Path: "/api/history", Body: fmt.Sprintf(`{"channel":%q,"limit":20}`, ch)})
	})
}

// CS-PUB-06: NATS raw mode with channel_replacements {":": "."} and
// allow_wildcards. A raw NATS message reaches channel subscribers as is; API
// and client publications go to the raw subject without wrapping; a wildcard
// channel receives messages of matching subjects.
func TestCS_PUB_06(t *testing.T) {
	h.Profiles(t, "n-raw")
	h.Run(t, "CS-PUB-06", func(s *h.Session) {
		ch := s.Ch("pub:raw")
		subj := strings.ReplaceAll(ch, ":", ".")
		raw := s.NATSSubscribe(subj)
		ws := subscribeOn(s, fmt.Sprintf(`{"channel":%q}`, ch))
		s.NATSPublish(subj, []byte(`{"from":"nats"}`))
		ws.Next()
		raw()
		s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"from":"api"},"tags":{"k":"v"}}`, ch)})
		ws.Next()
		raw()
		p := connect(s, h.HS256(h.Claims{Sub: "u2"})) // separate publisher: no reply/push ordering
		p.Send(fmt.Sprintf(`{"id":2,"publish":{"channel":%q,"data":{"from":"client"}}}`, ch))
		p.Next()
		ws.Next()
		raw()

		wild := s.Ch("wild") + ".*"
		w := subscribeOn(s, fmt.Sprintf(`{"channel":%q}`, wild))
		s.NATSPublish(s.Ch("wild")+".a", []byte(`{"n":1}`))
		w.Next()
		s.API(h.APIRequest{Path: "/api/presence", Body: fmt.Sprintf(`{"channel":%q}`, wild)})
		subscribeOn(s, fmt.Sprintf(`{"channel":%q}`, s.Ch("pub:wild")+".*")) // wildcard after replacement
	})
}
