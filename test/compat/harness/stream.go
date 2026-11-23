package harness

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/centrifugal/protocol"
)

// OpenHTTPStream posts a unidirectional connect request and returns a reader
// for JSON pushes. SSE frames must contain one data line and a blank separator.
func (s *Session) OpenHTTPStream(path, body string, sse bool) func() *protocol.Push {
	s.T.Helper()
	s.tr.add(Event{Kind: "stream_connect", Text: "POST " + path, Body: json.RawMessage(body)})
	client := http.Client{Timeout: Deadline}
	resp, err := client.Post(s.Target.BaseURL+path, "application/json", strings.NewReader(body))
	if err != nil {
		s.T.Fatalf("%s: open %s: %v", s.Target.Name, path, err)
	}
	s.T.Cleanup(func() { _ = resp.Body.Close() })
	s.tr.add(Event{Kind: "stream_status", Status: resp.StatusCode, Text: resp.Header.Get("Content-Type")})
	if resp.StatusCode != http.StatusOK {
		s.T.Fatalf("%s: %s returned %s", s.Target.Name, path, resp.Status)
	}

	scanner := bufio.NewScanner(resp.Body)
	return func() *protocol.Push {
		s.T.Helper()
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" && sse {
				continue
			}
			if sse {
				var ok bool
				line, ok = strings.CutPrefix(line, "data: ")
				if !ok || !scanner.Scan() || scanner.Text() != "" {
					s.T.Fatalf("%s: invalid SSE event framing", s.Target.Name)
				}
			}
			if line == "{}" {
				continue // Unidirectional keepalive, no pong expected.
			}
			push := new(protocol.Push)
			if err := json.Unmarshal([]byte(line), push); err != nil {
				s.T.Fatalf("%s: invalid stream push: %v", s.Target.Name, err)
			}
			s.tr.add(Event{Kind: "stream_push", Body: json.RawMessage(line)})
			return push
		}
		s.T.Fatalf("%s: stream ended before expected push: %v", s.Target.Name, scanner.Err())
		return nil
	}
}
