package harness

import (
	"encoding/json"
	"net/http"

	"github.com/centrifugal/protocol"
)

// RequireHTTPStatus records the response, checks its status and returns the body.
func (s *Session) RequireHTTPStatus(r APIRequest, want int) []byte {
	s.T.Helper()
	status, body := s.call(r)
	s.record(r, status, body)
	if status != want {
		s.T.Fatalf("%s: %s %s: got HTTP %d, want %d", s.Target.Name, r.method(), r.Path, status, want)
	}
	return body
}

// OpenEmulation binds outgoing commands to the identifiers delivered in the
// connect reply. SSE connects with cf_connect. HTTP-stream connects by POST.
func (s *Session) OpenEmulation(path, connect string, sse bool) (read func() *protocol.Reply, send func(string)) {
	s.T.Helper()
	method := http.MethodPost
	if sse {
		method = http.MethodGet
	}
	next := s.openStream(method, path, connect, sse)
	var node, session string
	emulate := func(command string, record bool) {
		s.T.Helper()
		if node == "" || session == "" {
			s.T.Fatal("connect reply missing emulation node or session")
		}
		body, err := json.Marshal(map[string]string{"node": node, "session": session, "data": command})
		if err != nil {
			s.T.Fatal(err)
		}
		req := APIRequest{Path: "/emulation", Body: string(body), Headers: map[string]string{"Content-Type": "application/json"}, NoKey: true}
		if record {
			s.tr.add(Event{Kind: "emulation_send", Body: body})
			s.RequireHTTPStatus(req, http.StatusNoContent)
		} else if status, _ := s.call(req); status != http.StatusNoContent {
			s.T.Fatalf("%s: emulated pong returned %d", s.Target.Name, status)
		}
	}
	return func() *protocol.Reply {
		s.T.Helper()
		for {
			line := next()
			if line == nil {
				s.tr.add(Event{Kind: "stream_eof"})
				return nil
			}
			if string(line) == "{}" {
				emulate("{}", false) // Timing-dependent heartbeat.
				continue
			}
			reply := new(protocol.Reply)
			if err := json.Unmarshal(line, reply); err != nil {
				s.T.Fatalf("%s: invalid reply: %v", s.Target.Name, err)
			}
			s.tr.add(Event{Kind: "stream_reply", Body: line})
			if c := reply.Connect; c != nil {
				node, session = c.Node, c.Session
			}
			return reply
		}
	}, func(command string) { emulate(command, true) }
}
