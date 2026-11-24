package harness

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/centrifugal/protocol"
	"github.com/gorilla/websocket"
)

// Deadline for any single expected event (spec §5: no sleeps, wait with a deadline).
const Deadline = 5 * time.Second

// runID makes channel names unique per test process, so long-running targets
// (memory history, presence) do not leak state between runs. It is replaced by
// "RUN" before normalization.
var runID = func() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}()

// Session records everything one scenario does against one target.
type Session struct {
	T      *testing.T
	Target Target
	tr     Transcript
}

// Transcript returns what was recorded so far.
func (s *Session) Transcript() Transcript { return s.tr }

// Ch returns a channel (or other identifier) unique to this test run.
func (s *Session) Ch(name string) string { return name + "-" + runID }

// Note records a marker event, e.g. a pause between phases of a scenario.
func (s *Session) Note(text string) { s.tr.add(Event{Kind: "note", Text: text}) }

// codec converts between wire frames and the JSON recorded in transcripts.
type codec interface {
	messages(frame []byte) ([]json.RawMessage, error) // replies/pushes as JSON
	pong() (int, []byte)                              // frame answering a server ping
	query() string
}

type jsonCodec struct{}

func (jsonCodec) query() string       { return "" }
func (jsonCodec) pong() (int, []byte) { return websocket.TextMessage, []byte("{}") }
func (jsonCodec) messages(frame []byte) ([]json.RawMessage, error) {
	var out []json.RawMessage
	for _, line := range bytes.Split(frame, []byte("\n")) {
		if len(bytes.TrimSpace(line)) > 0 {
			out = append(out, append(json.RawMessage(nil), line...))
		}
	}
	return out, nil
}

type protobufCodec struct{}

func (protobufCodec) query() string { return "?format=protobuf" }
func (protobufCodec) pong() (int, []byte) {
	b, _ := protocol.NewProtobufCommandEncoder().Encode(&protocol.Command{})
	return websocket.BinaryMessage, b
}
func (protobufCodec) messages(frame []byte) ([]json.RawMessage, error) {
	dec := protocol.NewProtobufReplyDecoder(frame)
	enc := protocol.NewJSONReplyEncoder()
	var out []json.RawMessage
	for {
		r, err := dec.Decode()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		b, err := enc.Encode(r)
		if err != nil {
			return nil, err
		}
		out = append(out, bytes.TrimSpace(b))
	}
}

// WS is a raw protocol client: commands are sent byte-exact.
type WS struct {
	s     *Session
	conn  *websocket.Conn
	codec codec
	buf   []json.RawMessage // received but not yet consumed messages
	last  json.RawMessage
	uni   bool // unidirectional connections do not send application pongs
}

// Dial opens a JSON-protocol WebSocket connection.
func (s *Session) Dial() *WS { return s.DialWithHeaders(nil) }

// DialWithHeaders adds transport headers to the JSON WebSocket handshake.
func (s *Session) DialWithHeaders(headers http.Header) *WS { return s.dial(jsonCodec{}, headers) }

// DialProtobuf opens a Protobuf-protocol WebSocket connection.
func (s *Session) DialProtobuf() *WS { return s.dial(protobufCodec{}, nil) }

func (s *Session) dial(c codec, headers http.Header) *WS {
	s.T.Helper()
	return s.dialURL(s.Target.WSURL()+c.query(), c, headers)
}

// DialUniWS supplies the connect request in cf_connect or the first text frame.
func (s *Session) DialUniWS(request string, inQuery bool) *WS {
	s.T.Helper()
	const path = "/connection/uni_websocket"
	endpoint := "ws" + strings.TrimPrefix(s.Target.BaseURL, "http") + path
	if inQuery {
		s.tr.add(Event{Kind: "ws_connect", Text: path + "?cf_connect", Body: json.RawMessage(request)})
		endpoint += "?cf_connect=" + url.QueryEscape(request)
	} else {
		s.tr.add(Event{Kind: "ws_connect", Text: path})
	}
	ws := s.dialURL(endpoint, jsonCodec{}, nil)
	ws.uni = true
	if !inQuery {
		ws.Send(request)
	}
	return ws
}

func (s *Session) dialURL(endpoint string, c codec, headers http.Header) *WS {
	s.T.Helper()
	conn, resp, err := websocket.DefaultDialer.Dial(endpoint, headers)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		s.T.Fatalf("%s: dial: %v", s.Target.Name, err)
	}
	s.T.Cleanup(func() { _ = conn.Close() })
	return &WS{s: s, conn: conn, codec: c}
}

// Close closes the connection from the client side (normal closure).
func (c *WS) Close() {
	_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	_ = c.conn.Close()
}

// Send writes one JSON frame; several commands may be joined with "\n".
func (c *WS) Send(frame string) { c.SendRaw(websocket.TextMessage, []byte(frame)) }

// SendRaw writes bytes as-is. Text frames are recorded per line; binary frames as their length.
func (c *WS) SendRaw(msgType int, frame []byte) {
	c.s.T.Helper()
	if msgType == websocket.TextMessage {
		for _, line := range strings.Split(string(frame), "\n") {
			body := json.RawMessage(line)
			if !json.Valid(body) {
				body = asJSON([]byte(strings.ToValidUTF8(line, "�")))
			}
			c.s.tr.add(Event{Kind: "ws_send", Body: body})
		}
	}
	if err := c.conn.WriteMessage(msgType, frame); err != nil {
		c.s.T.Fatalf("%s: write: %v", c.s.Target.Name, err)
	}
}

// SendCmd encodes commands with the Protobuf protocol into one frame. The
// commands are recorded in their JSON form.
func (c *WS) SendCmd(cmds ...*protocol.Command) {
	c.s.T.Helper()
	var frame []byte
	jenc, penc := protocol.NewJSONCommandEncoder(), protocol.NewProtobufCommandEncoder()
	for _, cmd := range cmds {
		j, err := jenc.Encode(cmd)
		if err != nil {
			c.s.T.Fatal(err)
		}
		c.s.tr.add(Event{Kind: "ws_send", Body: bytes.TrimSpace(j)})
		p, err := penc.Encode(cmd)
		if err != nil {
			c.s.T.Fatal(err)
		}
		frame = append(frame, p...)
	}
	if err := c.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		c.s.T.Fatalf("%s: write: %v", c.s.Target.Name, err)
	}
}

// read fills buf with at least one message, answering pings when answerPings
// is set. It returns a close error when the server closes the connection.
func (c *WS) read(answerPings bool) error {
	return c.readBefore(answerPings, time.Now().Add(Deadline))
}

func (c *WS) readBefore(answerPings bool, deadline time.Time) error {
	if err := c.conn.SetReadDeadline(deadline); err != nil {
		return err
	}
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return err
		}
		msgs, err := c.codec.messages(data)
		if err != nil {
			return fmt.Errorf("decode: %w", err)
		}
		added := false
		for _, m := range msgs {
			if answerPings && string(m) == "{}" {
				if !c.uni {
					t, b := c.codec.pong()
					if err := c.conn.WriteMessage(t, b); err != nil {
						return fmt.Errorf("pong: %w", err)
					}
				}
				continue
			}
			c.buf = append(c.buf, m)
			added = true
		}
		if added {
			return nil
		}
	}
}

// CaptureUntil reads pushes and answers pings while the scenario performs
// backend work. Call the returned function before using this WS again. It
// records the captured pushes at that point, after the backend actions.
func (c *WS) CaptureUntil(marker string, timeout time.Duration) func() {
	c.s.T.Helper()
	type result struct {
		count int
		err   error
	}
	done := make(chan result, 1)
	deadline := time.Now().Add(timeout)
	go func() {
		for i := 0; ; i++ {
			if i == len(c.buf) {
				if err := c.readBefore(true, deadline); err != nil {
					done <- result{err: err}
					return
				}
			}
			if strings.Contains(string(c.buf[i]), marker) {
				done <- result{count: i + 1}
				return
			}
		}
	}()
	return func() {
		c.s.T.Helper()
		captured := <-done
		if captured.err != nil {
			c.s.T.Fatalf("%s: waiting for push %s: %v", c.s.Target.Name, marker, captured.err)
		}
		c.Recv(captured.count)
	}
}

// Recv waits for n messages (replies or pushes). Server pings are answered and not counted.
func (c *WS) Recv(n int) {
	c.s.T.Helper()
	for len(c.buf) < n {
		if err := c.read(true); err != nil {
			c.s.T.Fatalf("%s: waiting for %d messages, got %d: %v", c.s.Target.Name, n, len(c.buf), err)
		}
	}
	for _, m := range c.buf[:n] {
		c.s.tr.add(Event{Kind: "ws_recv", Body: m})
		c.last = m
	}
	c.buf = c.buf[n:]
}

// RecvUntil records messages until one contains marker (e.g. the data of a
// final publication), so the number of messages before it may vary per engine.
func (c *WS) RecvUntil(marker string) {
	c.s.T.Helper()
	for {
		c.Recv(1)
		if strings.Contains(string(c.last), marker) {
			return
		}
	}
}

// Next records the next message or the close of the connection, whichever
// comes first. Use it where the server may answer with an error reply or a
// disconnect.
func (c *WS) Next() {
	c.s.T.Helper()
	if len(c.buf) == 0 {
		if err := c.read(true); err != nil {
			c.recordClose(err)
			return
		}
	}
	c.Recv(1)
}

// ExpectClose waits for the server's close frame, records it and returns its code.
func (c *WS) ExpectClose() int {
	c.s.T.Helper()
	for {
		if err := c.read(true); err != nil {
			return c.recordClose(err)
		}
		c.buf = nil
	}
}

func (c *WS) recordClose(err error) int {
	c.s.T.Helper()
	var ce *websocket.CloseError
	if errors.As(err, &ce) {
		c.s.tr.add(Event{Kind: "ws_close", Code: ce.Code, Text: ce.Text})
		return ce.Code
	}
	c.s.T.Fatalf("%s: expected a message or a close frame, got %v", c.s.Target.Name, err)
	return 0
}

// RecvPing waits for a server ping and records it. With answer=false no pong
// is sent, so the server should disconnect for the missing pong.
func (c *WS) RecvPing(answer bool) {
	c.s.T.Helper()
	if err := c.read(false); err != nil {
		c.s.T.Fatalf("%s: waiting for ping: %v", c.s.Target.Name, err)
	}
	if len(c.buf) == 0 || string(c.buf[0]) != "{}" {
		c.s.T.Fatalf("%s: expected ping, got %s", c.s.Target.Name, c.buf[0])
	}
	c.buf = c.buf[1:]
	c.s.tr.add(Event{Kind: "ws_recv", Text: "ping", Body: json.RawMessage("{}")})
	if answer {
		t, b := c.codec.pong()
		_ = c.conn.WriteMessage(t, b)
		c.s.tr.add(Event{Kind: "ws_send", Text: "pong", Body: json.RawMessage("{}")})
	}
}

// Last decodes the most recently received message into v.
func (c *WS) Last(v any) {
	c.s.T.Helper()
	if err := json.Unmarshal(c.last, v); err != nil {
		c.s.T.Fatalf("%s: decode %s: %v", c.s.Target.Name, c.last, err)
	}
}

// APIRequest describes one server API HTTP call.
type APIRequest struct {
	Method   string            // HTTP method, default POST
	Path     string            // e.g. "/api/publish" (query allowed)
	Body     string            // raw body
	Headers  map[string]string // extra headers
	NoKey    bool              // do not add the X-API-Key header
	KeysOnly bool              // record only the shape of the JSON body (process-specific values)
	Omit     []string          // object keys dropped anywhere in the body before recording
}

// API performs one HTTP call, records status and body, and returns the raw body.
func (s *Session) API(r APIRequest) []byte {
	s.T.Helper()
	status, body := s.call(r)
	s.record(r, status, body)
	return body
}

// APIUntil repeats the call without recording until done(body) holds or
// timeout passes, then records the last answer. It waits for a state that
// takes time to appear (TTL expiry, a backend coming back), never as a sleep.
func (s *Session) APIUntil(r APIRequest, timeout time.Duration, done func(body []byte) bool) []byte {
	s.T.Helper()
	deadline := time.Now().Add(timeout)
	for {
		status, body := s.call(r)
		if done(body) || time.Now().After(deadline) {
			s.record(r, status, body)
			return body
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *Session) call(r APIRequest) (int, []byte) {
	s.T.Helper()
	req, err := http.NewRequest(r.method(), s.Target.BaseURL+r.Path, strings.NewReader(r.Body))
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
		s.T.Fatalf("%s: %s %s: %v", s.Target.Name, r.method(), r.Path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.T.Fatal(err)
	}
	return resp.StatusCode, body
}

func (s *Session) record(r APIRequest, status int, body []byte) {
	rec := asJSON(body)
	if len(r.Omit) > 0 && rec != nil {
		rec = omit(rec, r.Omit)
	}
	if r.KeysOnly && rec != nil {
		rec = shape(rec)
	}
	s.tr.add(Event{Kind: "http", Text: r.method() + " " + r.Path, Status: status, Body: rec})
}

func (r APIRequest) method() string {
	if r.Method == "" {
		return http.MethodPost
	}
	return r.Method
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
