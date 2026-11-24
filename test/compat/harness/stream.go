package harness

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/centrifugal/protocol"
)

// OpenHTTPStream posts a unidirectional connect request and returns a reader
// for JSON pushes. SSE frames must contain one data line and a blank separator.
func (s *Session) OpenHTTPStream(path, body string, sse bool) func() *protocol.Push {
	s.T.Helper()
	next := s.openStream(http.MethodPost, path, body, sse)
	return func() *protocol.Push {
		s.T.Helper()
		for {
			line := next()
			if line == nil {
				s.T.Fatalf("%s: stream ended before expected push", s.Target.Name)
			}
			if string(line) == "{}" {
				continue // Unidirectional keepalive, no pong expected.
			}
			push := new(protocol.Push)
			if err := json.Unmarshal(line, push); err != nil {
				s.T.Fatalf("%s: invalid stream push: %v", s.Target.Name, err)
			}
			s.tr.add(Event{Kind: "stream_push", Body: line})
			return push
		}
	}
}

// openStream reads one JSON line (or SSE data event) at a time and returns nil on EOF.
func (s *Session) openStream(method, path, body string, sse bool) func() json.RawMessage {
	s.T.Helper()
	s.tr.add(Event{Kind: "stream_connect", Text: method + " " + path, Body: json.RawMessage(body)})
	endpoint := s.Target.BaseURL + path
	client := http.Client{Timeout: Deadline}
	var resp *http.Response
	var err error
	if method == http.MethodGet {
		resp, err = client.Get(endpoint + "?cf_connect=" + url.QueryEscape(body))
	} else {
		resp, err = client.Post(endpoint, "application/json", strings.NewReader(body))
	}
	if err != nil {
		// A GET transport carries the test token in its URL.
		if urlErr, ok := err.(*url.Error); ok {
			err = urlErr.Err
		}
		s.T.Fatalf("%s: open %s: %v", s.Target.Name, path, err)
	}
	s.T.Cleanup(func() { _ = resp.Body.Close() })
	contentType := resp.Header.Get("Content-Type")
	s.tr.add(Event{Kind: "stream_status", Status: resp.StatusCode, Text: contentType})
	if resp.StatusCode != http.StatusOK || (sse && !strings.HasPrefix(contentType, "text/event-stream")) {
		s.T.Fatalf("%s: %s returned %s with Content-Type %q", s.Target.Name, path, resp.Status, contentType)
	}

	scanner := bufio.NewScanner(resp.Body)
	return func() json.RawMessage {
		s.T.Helper()
		for scanner.Scan() {
			line := scanner.Text()
			if sse {
				if line == "" {
					continue // The handler flushes an empty line when the SSE stream opens.
				}
				var ok bool
				line, ok = strings.CutPrefix(line, "data: ")
				if !ok || !scanner.Scan() || scanner.Text() != "" {
					s.T.Fatalf("%s: invalid SSE event framing", s.Target.Name)
				}
			}
			return json.RawMessage(line)
		}
		if err := scanner.Err(); err != nil {
			s.T.Fatalf("%s: %s: %v", s.Target.Name, path, err)
		}
		return nil
	}
}
