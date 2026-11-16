package scenarios

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// CS-NAME-01 (sut only, ADR-0022): default mode exposes Cedrazeka names,
// compatibility mode exposes the upstream ones.
func TestCS_NAME_01(t *testing.T) {
	h.Profiles(t, "m")
	if h.Mode() == "record" {
		t.Skip("sut-only scenario")
	}
	def, compat := h.SUTDefault(), h.SUT()
	if err := h.WaitReady(def, h.Deadline); err != nil {
		t.Fatal(err)
	}

	metrics := func(tg h.Target) string {
		resp, err := http.Get(tg.BaseURL + "/metrics")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}
	if m := metrics(def); !strings.Contains(m, "\ncedrazeka_") || strings.Contains(m, "\ncentrifugo_") {
		t.Error("default mode: metrics must use the cedrazeka_ prefix only")
	}
	if m := metrics(compat); !strings.Contains(m, "\ncentrifugo_") || strings.Contains(m, "\ncedrazeka_") {
		t.Error("compat mode: metrics must use the centrifugo_ prefix only")
	}

	// Transport error mode turns API errors into HTTP status codes; it is
	// selected by a branded header.
	status := func(tg h.Target, header string) int {
		req, _ := http.NewRequest(http.MethodPost, tg.BaseURL+"/api/publish", strings.NewReader(`{"channel":"","data":{}}`))
		req.Header.Set("X-API-Key", h.APIKey)
		req.Header.Set(header, "transport")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	if status(def, "X-Cedrazeka-Error-Mode") == http.StatusOK {
		t.Error("default mode: X-Cedrazeka-Error-Mode must select transport error mode")
	}
	if status(def, "X-Centrifugo-Error-Mode") != http.StatusOK {
		t.Error("default mode: X-Centrifugo-Error-Mode must be ignored")
	}
	if status(compat, "X-Centrifugo-Error-Mode") == http.StatusOK {
		t.Error("compat mode: X-Centrifugo-Error-Mode must select transport error mode")
	}
}

// CS-NAME-02 (sut only, ADR-0022): default mode prefixes Redis keys and
// channels and NATS subjects with "cedrazeka", compatibility mode with
// "centrifugo". sut and sut-default share the sut side's backends, so each
// mode must leave only its own prefix on the client channel it serves.
// (PostgreSQL: table_prefix "cf" in both modes, not a brand name.)
func TestCS_NAME_02(t *testing.T) {
	h.Profiles(t, "r", "n")
	if h.Mode() == "record" {
		t.Skip("sut-only scenario")
	}
	if err := h.WaitReady(h.SUTDefault(), h.Deadline); err != nil {
		t.Fatal(err)
	}
	modes := []struct {
		tg            h.Target
		prefix, other string
	}{{h.SUTDefault(), "cedrazeka", "centrifugo"}, {h.SUT(), "centrifugo", "cedrazeka"}}
	for _, m := range modes {
		s := &h.Session{T: t, Target: m.tg}
		ch := s.Ch("names-" + m.prefix) // channel without namespace: presence on
		ws := connect(s, h.HS256(h.Claims{Sub: "u1"}))
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, ch))
		ws.Recv(1)
		redis := func(args ...string) string {
			return s.Exec("redis", append([]string{"redis-cli", "--no-auth-warning", "--user", "compat", "--pass", "compat-test-redis-password"}, args...)...)
		}
		// Presence lives in Redis on both profiles.
		for p, want := range map[string]string{m.prefix: "1", m.other: "0"} {
			if got := strings.TrimSpace(redis("EXISTS", p+".presence.data."+ch)); got != want {
				t.Errorf("%s: EXISTS %s.presence.data.<ch> = %s, want %s", m.tg.Name, p, got, want)
			}
		}
		// The broker's subscription for the client channel.
		var subs func(p string) bool
		if h.Profile() == "r" {
			subs = func(p string) bool {
				return strings.Contains(redis("PUBSUB", "CHANNELS", p+".client.*"), ch)
			}
		} else {
			sz := s.Exec("nats", "wget", "-qO-", "http://127.0.0.1:8222/subsz?subs=1")
			subs = func(p string) bool { return strings.Contains(sz, `"`+p+".client."+ch+`"`) }
		}
		if !subs(m.prefix) || subs(m.other) {
			t.Errorf("%s: broker subscription must use the %s prefix only", m.tg.Name, m.prefix)
		}
		ws.Close()
	}
}
