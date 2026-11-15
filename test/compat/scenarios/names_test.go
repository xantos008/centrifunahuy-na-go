package scenarios

import (
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
