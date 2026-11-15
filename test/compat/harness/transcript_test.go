package harness

import (
	"encoding/json"
	"strings"
	"testing"
)

func ev(body string) Event { return Event{Kind: "ws_recv", Body: json.RawMessage(body)} }

func TestNormalizeVolatileValues(t *testing.T) {
	tr := Transcript{Events: []Event{
		ev(`{"id":1,"connect":{"client":"aaa","version":"6.10.3","ping":25}}`),
		ev(`{"push":{"channel":"c","join":{"info":{"client":"bbb","user":"u"}}}}`),
		ev(`{"push":{"channel":"c","leave":{"info":{"client":"aaa","user":"u"}}}}`),
		ev(`{"id":2,"subscribe":{"epoch":"xyz","offset":3}}`),
	}}
	n, err := Normalize(tr)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(Lines(n), "\n")
	for _, want := range []string{
		`"client":"<client:1>"`, // aaa
		`"client":"<client:2>"`, // bbb
		`"version":"<version:1>"`,
		`"epoch":"<epoch:1>"`,
		`"offset":3`, // not volatile
		`"ping":25`,
		`"user":"u"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in\n%s", want, got)
		}
	}
	// The same client ID maps to the same placeholder (equality is kept).
	if strings.Count(got, `<client:1>`) != 2 {
		t.Errorf("client aaa should appear twice as <client:1>:\n%s", got)
	}
}

func TestNormalizeSameShapeDifferentIDsEqual(t *testing.T) {
	a, _ := Normalize(Transcript{Events: []Event{ev(`{"connect":{"client":"1111"}}`)}})
	b, _ := Normalize(Transcript{Events: []Event{ev(`{"connect":{"client":"2222"}}`)}})
	if d := Diff("a", a, "b", b); d != "" {
		t.Fatalf("volatile-only difference must vanish:\n%s", d)
	}
}

func TestDiffReportsRealDifference(t *testing.T) {
	a, _ := Normalize(Transcript{Events: []Event{ev(`{"error":{"code":103}}`)}})
	b, _ := Normalize(Transcript{Events: []Event{ev(`{"error":{"code":102}}`)}})
	d := Diff("ref", a, "sut", b)
	if !strings.Contains(d, "103") || !strings.Contains(d, "102") {
		t.Fatalf("diff should show both values:\n%s", d)
	}
	// Extra trailing event is a difference too.
	c, _ := Normalize(Transcript{Events: []Event{ev(`{"error":{"code":103}}`), ev(`{}`)}})
	if Diff("ref", a, "sut", c) == "" {
		t.Fatal("length difference not reported")
	}
}

func TestNonJSONBodyIsQuoted(t *testing.T) {
	if got := string(asJSON([]byte("Bad Request\n"))); got != `"Bad Request"` {
		t.Fatalf("got %s", got)
	}
	if asJSON(nil) != nil {
		t.Fatal("empty body must stay empty")
	}
}

func TestGoldenRoundTrip(t *testing.T) {
	dir := t.TempDir()
	prev := GoldenDir
	GoldenDir = dir
	t.Cleanup(func() { GoldenDir = prev })

	tr, _ := Normalize(Transcript{Events: []Event{ev(`{"connect":{"client":"abc"}}`), {Kind: "http", Status: 404, Body: asJSON([]byte("404 page not found"))}}})
	if err := saveGolden("x", tr); err != nil {
		t.Fatal(err)
	}
	got, err := loadGolden("x")
	if err != nil {
		t.Fatal(err)
	}
	if d := Diff("saved", tr, "loaded", got); d != "" {
		t.Fatalf("golden round trip changed the transcript:\n%s", d)
	}
}

func TestPresenceKeysAndShape(t *testing.T) {
	tr, err := Normalize(Transcript{Events: []Event{ev(`{"result":{"presence":{"c-111":{"client":"c-111","user":"u1"},"c-222":{"client":"c-222","user":"u2"}}}}`)}})
	if err != nil {
		t.Fatal(err)
	}
	got := Lines(tr)[0]
	if strings.Contains(got, "c-111") || !strings.Contains(got, `"<client:1>":{"client":"<client:1>","user":"u1"}`) || !strings.Contains(got, `"<client:2>":{"client":"<client:2>","user":"u2"}`) {
		t.Fatalf("presence keys not normalized consistently: %s", got)
	}
	if s := string(shape(json.RawMessage(`{"a":1,"b":["x",true],"c":null}`))); s != `{"a":"<float64>","b":["<string>","<bool>"],"c":"<null>"}` {
		t.Fatalf("shape: %s", s)
	}
}

func TestRunIDReplaced(t *testing.T) {
	tr := withoutRunID(Transcript{Events: []Event{{Kind: "http", Text: "POST /api/x?c=" + runID, Body: json.RawMessage(`{"channel":"a-` + runID + `"}`)}}})
	if strings.Contains(Lines(tr)[0], runID) || !strings.Contains(Lines(tr)[0], "a-RUN") {
		t.Fatalf("run ID not replaced: %s", Lines(tr)[0])
	}
}

func TestOmit(t *testing.T) {
	got := string(omit(json.RawMessage(`{"result":{"nodes":[{"name":"n","metrics":{"items":{"a":1}}}]}}`), []string{"metrics"}))
	if got != `{"result":{"nodes":[{"name":"n"}]}}` {
		t.Fatalf("omit: %s", got)
	}
}
