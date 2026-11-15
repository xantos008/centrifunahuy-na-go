// Package harness runs the same compatibility scenario against the reference
// server (Centrifugo OSS v6.10.3) and our server, and compares normalized
// transcripts (ADR-0028, docs/testing/compatibility-suite.md).
package harness

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Test-only credentials, identical in configs/<profile>/config.json. Not secrets.
const (
	HMACSecret = "compat-test-hmac-secret-not-a-real-secret" //nolint:gosec // test-only value, also in configs/
	APIKey     = "compat-test-api-key"                       //nolint:gosec // test-only value, also in configs/
)

// Target is one server under comparison.
type Target struct {
	Name    string // "ref" or "sut"
	BaseURL string // http://host:port
}

// WSURL is the bidirectional WebSocket endpoint.
func (t Target) WSURL() string {
	return "ws" + strings.TrimPrefix(t.BaseURL, "http") + "/connection/websocket"
}

// Ref and SUT read COMPAT_REF_URL / COMPAT_SUT_URL, defaulting to the ports
// published by test/compat/docker-compose.yml.
func Ref() Target { return Target{"ref", envOr("COMPAT_REF_URL", "http://127.0.0.1:18000")} }
func SUT() Target { return Target{"sut", envOr("COMPAT_SUT_URL", "http://127.0.0.1:28000")} }

// SUTDefault is our server in default (Cedrazeka names) mode, for CS-NAME-01.
func SUTDefault() Target {
	return Target{"sut-default", envOr("COMPAT_SUT_DEFAULT_URL", "http://127.0.0.1:28001")}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// WaitReady polls /health until it answers 200 or the timeout passes.
func WaitReady(t Target, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := http.Client{Timeout: time.Second}
	for {
		resp, err := client.Get(t.BaseURL + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s at %s not ready after %s (last error: %v)", t.Name, t.BaseURL, timeout, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
