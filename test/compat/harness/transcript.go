package harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Event is one observable wire event. Body is JSON (a frame, an HTTP body, or
// a JSON string for non-JSON bodies).
type Event struct {
	Kind   string          `json:"kind"`             // ws_send, ws_recv, ws_close, http
	Text   string          `json:"text,omitempty"`   // close reason, or "POST /api/publish"
	Code   int             `json:"code,omitempty"`   // close code
	Status int             `json:"status,omitempty"` // HTTP status
	Body   json.RawMessage `json:"body,omitempty"`
}

// Transcript is an ordered list of events from one scenario run.
type Transcript struct {
	Events []Event `json:"events"`
}

func (tr *Transcript) add(e Event) { tr.Events = append(tr.Events, e) }

// volatileKeys holds JSON object keys whose values differ between processes
// or runs. Their values are replaced by numbered placeholders; equality inside
// one transcript is kept. Changing this list requires a known-differences
// entry or an update of docs/testing/compatibility-suite.md §3.
// Scenarios must not use these keys inside their own payloads.
var volatileKeys = map[string]bool{
	"client":  true, // client ID (UUID)
	"session": true, // session ID
	"epoch":   true, // stream epoch
	"node":    true, // node ID
	"uid":     true, // node ID in info
	"version": true, // server version
	"uptime":  true,
	"time":    true, // wall-clock timestamps
}

// Normalize returns a copy of tr with volatile values replaced.
func Normalize(tr Transcript) (Transcript, error) {
	seen := map[string]map[string]int{} // key -> value -> placeholder number
	out := Transcript{Events: make([]Event, len(tr.Events))}
	for i, e := range tr.Events {
		out.Events[i] = e
		if len(e.Body) == 0 {
			continue
		}
		dec := json.NewDecoder(bytes.NewReader(e.Body))
		dec.UseNumber()
		var v any // generic JSON tree: the wire shape is what is compared
		if err := dec.Decode(&v); err != nil {
			return Transcript{}, fmt.Errorf("event %d: %w", i, err)
		}
		v = normalizeValue(v, seen)
		b, err := marshal(v) // object keys come out sorted
		if err != nil {
			return Transcript{}, err
		}
		out.Events[i].Body = b
	}
	return out, nil
}

func normalizeValue(v any, seen map[string]map[string]int) any {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			if volatileKeys[k] {
				if _, isObj := val.(map[string]any); !isObj {
					x[k] = placeholder(k, fmt.Sprint(val), seen)
					continue
				}
			}
			x[k] = normalizeValue(val, seen)
		}
	case []any:
		for i := range x {
			x[i] = normalizeValue(x[i], seen)
		}
	}
	return v
}

func placeholder(key, val string, seen map[string]map[string]int) string {
	if seen[key] == nil {
		seen[key] = map[string]int{}
	}
	n, ok := seen[key][val]
	if !ok {
		n = len(seen[key]) + 1
		seen[key][val] = n
	}
	return fmt.Sprintf("<%s:%d>", key, n)
}

// Lines renders a transcript as one canonical JSON line per event.
func Lines(tr Transcript) []string {
	lines := make([]string, len(tr.Events))
	for i, e := range tr.Events {
		b, _ := marshal(e)
		lines[i] = string(b)
	}
	return lines
}

// marshal is json.Marshal without HTML escaping, so placeholders read as <client:1>.
func marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// Diff returns "" when both transcripts are equal, else a readable report.
func Diff(aName string, a Transcript, bName string, b Transcript) string {
	la, lb := Lines(a), Lines(b)
	if strings.Join(la, "\n") == strings.Join(lb, "\n") {
		return ""
	}
	var sb strings.Builder
	n := max(len(la), len(lb))
	for i := 0; i < n; i++ {
		var x, y string
		if i < len(la) {
			x = la[i]
		}
		if i < len(lb) {
			y = lb[i]
		}
		if x == y {
			fmt.Fprintf(&sb, "  %3d   %s\n", i, x)
			continue
		}
		fmt.Fprintf(&sb, "- %3d %s: %s\n+ %3d %s: %s\n", i, aName, x, i, bName, y)
	}
	return sb.String()
}
