package harness

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Deadline for any single expected event (spec §5: no sleeps, wait with a deadline).
const Deadline = 5 * time.Second

// Session records everything one scenario does against one target.
type Session struct {
	T      *testing.T
	Target Target
	tr     Transcript
}

// Transcript returns what was recorded so far.
func (s *Session) Transcript() Transcript { return s.tr }

// WS is a raw protocol client: frames are sent and received byte-exact (JSON format).
type WS struct {
	s    *Session
	conn *websocket.Conn
	buf  []json.RawMessage // received but not yet consumed messages
}

// Dial opens a JSON-protocol WebSocket connection.
func (s *Session) Dial() *WS {
	s.T.Helper()
	conn, resp, err := websocket.DefaultDialer.Dial(s.Target.WSURL(), nil)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		s.T.Fatalf("%s: dial: %v", s.Target.Name, err)
	}
	s.T.Cleanup(func() { _ = conn.Close() })
	return &WS{s: s, conn: conn}
}

// Send writes one frame; several commands may be joined with "\n".
func (c *WS) Send(frame string) {
	c.s.T.Helper()
	for _, line := range strings.Split(frame, "\n") {
		c.s.tr.add(Event{Kind: "ws_send", Body: json.RawMessage(line)})
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, []byte(frame)); err != nil {
		c.s.T.Fatalf("%s: write: %v", c.s.Target.Name, err)
	}
}

// Recv waits for n messages (replies or pushes). Server pings ("{}") are
// answered and not counted.
func (c *WS) Recv(n int) {
	c.s.T.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(Deadline))
	for len(c.buf) < n {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.s.T.Fatalf("%s: waiting for %d messages, got %d: %v", c.s.Target.Name, n, len(c.buf), err)
		}
		for _, line := range bytes.Split(data, []byte("\n")) {
			if len(bytes.TrimSpace(line)) == 0 {
				continue
			}
			if string(line) == "{}" {
				_ = c.conn.WriteMessage(websocket.TextMessage, []byte("{}"))
				continue
			}
			c.buf = append(c.buf, append(json.RawMessage(nil), line...))
		}
	}
	for _, m := range c.buf[:n] {
		c.s.tr.add(Event{Kind: "ws_recv", Body: m})
	}
	c.buf = c.buf[n:]
}

// ExpectClose waits for the server to close the connection and records code and reason.
func (c *WS) ExpectClose() {
	c.s.T.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(Deadline))
	for {
		_, _, err := c.conn.ReadMessage()
		if err == nil {
			continue
		}
		var ce *websocket.CloseError
		if errors.As(err, &ce) {
			c.s.tr.add(Event{Kind: "ws_close", Code: ce.Code, Text: ce.Text})
			return
		}
		c.s.T.Fatalf("%s: expected close frame, got %v", c.s.Target.Name, err)
	}
}

// APIRequest describes one server API HTTP call.
type APIRequest struct {
	Method  string            // HTTP method, default POST
	Path    string            // e.g. "/api/publish" (query allowed)
	Body    string            // raw body
	Headers map[string]string // extra headers
	NoKey   bool              // do not add the X-API-Key header
}

// API performs one HTTP call and records status and body.
func (s *Session) API(r APIRequest) {
	s.T.Helper()
	method := r.Method
	if method == "" {
		method = http.MethodPost
	}
	req, err := http.NewRequest(method, s.Target.BaseURL+r.Path, strings.NewReader(r.Body))
	if err != nil {
		s.T.Fatal(err)
	}
	if !r.NoKey {
		req.Header.Set("X-API-Key", APIKey)
	}
	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}
	resp, err := (&http.Client{Timeout: Deadline}).Do(req)
	if err != nil {
		s.T.Fatalf("%s: %s %s: %v", s.Target.Name, method, r.Path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.T.Fatal(err)
	}
	s.tr.add(Event{Kind: "http", Text: method + " " + r.Path, Status: resp.StatusCode, Body: asJSON(body)})
}

// asJSON keeps JSON bodies as they are and wraps anything else as a JSON string.
func asJSON(b []byte) json.RawMessage {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil
	}
	if json.Valid(b) {
		return b
	}
	q, _ := json.Marshal(string(b))
	return q
}
