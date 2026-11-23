package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// Engine profiles (docs/plan/phase-2.md). COMPAT_PROFILE selects the one the
// targets were started with (`make compat-up PROFILE=…`); default "m".
var profiles = map[string]profileSpec{
	"cloud":    {nodes: 1, backends: []string{"localstack", "pubsub"}, probe: []string{"localstack", "pubsub"}, noSUTDefault: true},
	"kafka":    {nodes: 1, backends: []string{"kafka"}, probe: []string{"kafka"}, noSUTDefault: true},
	"m":        {nodes: 1},
	"r":        {nodes: 2, backends: []string{"redis"}, probe: []string{"redis"}},
	"r-opt":    {nodes: 1, backends: []string{"redis"}, probe: []string{"redis"}},
	"r-lists":  {nodes: 1, backends: []string{"redis"}, probe: []string{"redis"}},
	"rs":       {nodes: 2, backends: []string{"redis", "replica1", "replica2", "sentinel"}, probe: []string{"sentinel"}},
	"rc":       {nodes: 2, backends: []string{"rc1", "rc2", "rc3"}, probe: []string{"rc1"}},
	"rsh":      {nodes: 2, backends: []string{"redis", "redis2"}, probe: []string{"redis", "redis2"}},
	"n":        {nodes: 2, backends: []string{"nats", "redis"}, probe: []string{"nats", "redis"}},
	"n-raw":    {nodes: 1, backends: []string{"nats", "redis"}, probe: []string{"nats", "redis"}},
	"p":        {nodes: 2, backends: []string{"postgres"}, probe: []string{"postgres"}, noSUTDefault: true, outbox: true},
	"p-notify": {nodes: 1, backends: []string{"postgres"}, probe: []string{"postgres"}, noSUTDefault: true, outbox: true},
}

type profileSpec struct {
	nodes    int      // server nodes per side
	backends []string // backend hostnames, one service per side ("redis" → redis-ref, redis-sut)
	probe    []string // CS-ISO-01 probe entry points: nats, postgres, sentinel (writes via the current master), or writable Redis nodes (cluster: any node)
	// noSUTDefault: skip the branding probe where it would share the SUT's
	// PostgreSQL tables or consume its queue messages.
	noSUTDefault bool
	// outbox: delivery is asynchronous (PG broker outbox worker), so a
	// subscriber that arrives right after a publish may get it as a push.
	outbox bool
}

// Profile is the current engine profile.
func Profile() string { return envOr("COMPAT_PROFILE", "m") }

// CheckProfile reports an unknown COMPAT_PROFILE.
func CheckProfile() error {
	if _, ok := profiles[Profile()]; !ok {
		return fmt.Errorf("unknown COMPAT_PROFILE %q", Profile())
	}
	return nil
}

// Profiles skips the test unless the current profile is one of ps.
func Profiles(t *testing.T, ps ...string) {
	t.Helper()
	if !slices.Contains(ps, Profile()) {
		t.Skipf("profile %q: scenario runs on %v", Profile(), ps)
	}
}

// RefNodes and SUTNodes are the server nodes of each side in the current
// profile. Node A is Ref()/SUT(); node B listens on port +100.
func RefNodes() []Target {
	return nodes(Ref(), Target{"ref", envOr("COMPAT_REF_B_URL", "http://127.0.0.1:18100")})
}

func SUTNodes() []Target {
	return nodes(SUT(), Target{"sut", envOr("COMPAT_SUT_B_URL", "http://127.0.0.1:28100")})
}

func nodes(a, b Target) []Target {
	if profiles[Profile()].nodes == 2 {
		return []Target{a, b}
	}
	return []Target{a}
}

// recordList is the explicit COMPAT_RECORD list: golden files are never
// re-recorded wholesale (condition C3 of the phase-2 plan).
func recordList() []string {
	var out []string
	for _, n := range strings.Split(envOr("COMPAT_RECORD", ""), ",") {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// CheckRecordList fails record mode without an explicit scenario list.
func CheckRecordList() error {
	if Mode() == "record" && len(recordList()) == 0 {
		return fmt.Errorf("record mode needs COMPAT_RECORD=<scenario>[,<scenario>…]; golden files are never re-recorded wholesale")
	}
	return nil
}

// serviceFor maps a service base name to the compose service of one side.
// Nodes are "node" (A) and "node-b"; backends are "<base>-<side>". A session
// can only address its own side (condition C2): every scenario runs the same
// code on ref and on sut, so backend actions are symmetric by construction.
func serviceFor(base, target string) string {
	side := side(target)
	switch base {
	case "node":
		return side
	case "node-b":
		return side + "-b"
	}
	return base + "-" + side
}

// AwaitDelivery lets an asynchronous broker deliver what was just published
// before the scenario subscribes; otherwise a fresh subscriber may receive
// those publications as pushes on ref and sut alike, at random. No-op (and
// no note) on synchronous brokers.
// ponytail: fixed 500 ms = 5x the default outbox poll interval; poll the
// outbox position if it ever flakes.
func (s *Session) AwaitDelivery() {
	if profiles[Profile()].outbox {
		time.Sleep(500 * time.Millisecond)
		s.Note("await outbox delivery")
	}
}

// OnB runs fn with this session pointed at node B of its side: connections
// dialed and API calls made inside fn go to node B (connections keep their
// node afterwards). Recorded as notes "node b" / "node a".
func (s *Session) OnB(fn func()) {
	s.T.Helper()
	ns := RefNodes()
	if side(s.Target.Name) == "sut" {
		ns = SUTNodes()
	}
	if len(ns) < 2 {
		s.T.Fatalf("profile %q has one node", Profile())
	}
	a := s.Target
	s.Target = ns[1]
	s.Note("node b")
	defer func() { s.Target = a; s.Note("node a") }()
	fn()
}

// ReplaceB stops node B of this side and starts a one-off copy of it, on the
// same port, with configs/<config> instead of the profile's config. restore
// removes the copy and starts node B again. Recorded as notes.
func (s *Session) ReplaceB(config string) (restore func()) {
	s.T.Helper()
	s.Compose("stop", "node-b")
	dir, err := filepath.Abs(filepath.Join(filepath.Dir(envOr("COMPAT_COMPOSE_FILE", "../docker-compose.yml")), "configs", config))
	if err != nil {
		s.T.Fatal(err)
	}
	out, err := compose("run", "-d", "--rm", "--no-deps", "--service-ports", "-v", dir+":/etc/compat:ro", serviceFor("node-b", s.Target.Name))
	if err != nil {
		s.T.Fatalf("%s: %v", s.Target.Name, err)
	}
	id := strings.TrimSpace(out[strings.LastIndex(strings.TrimSpace(out), "\n")+1:])
	// On failure the profile must stay usable for later scenarios.
	s.T.Cleanup(func() {
		_, _ = docker("rm", "-f", id)
		_, _ = compose("start", serviceFor("node-b", s.Target.Name))
	})
	s.Note("run node-b with configs/" + config)
	s.OnB(func() {
		if err := WaitReady(s.Target, 30*time.Second); err != nil {
			s.T.Fatal(err)
		}
	})
	return func() {
		s.T.Helper()
		if _, err := docker("rm", "-f", id); err != nil {
			s.T.Fatalf("%s: %v", s.Target.Name, err)
		}
		s.Note("remove node-b copy")
	}
}

// infoNodes is the number of nodes /api/info on the session's current node
// lists.
func (s *Session) infoNodes() int {
	s.T.Helper()
	_, body := s.call(APIRequest{Path: "/api/info", Body: "{}"})
	var r struct {
		Result struct {
			Nodes []json.RawMessage `json:"nodes"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		s.T.Fatalf("%s: info: %v: %s", s.Target.Name, err, body)
	}
	return len(r.Result.Nodes)
}

// AwaitNodes waits until /api/info on the current node lists n nodes and
// records "info: <n> nodes", or the count seen when the timeout passed.
func (s *Session) AwaitNodes(n int, timeout time.Duration) {
	s.T.Helper()
	deadline := time.Now().Add(timeout)
	got := s.infoNodes()
	for got != n && time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		got = s.infoNodes()
	}
	s.Note(fmt.Sprintf("info: %d nodes", got))
}

// side is "ref" or "sut"; sut and sut-default share the sut side.
func side(target string) string {
	if strings.HasPrefix(target, "sut") {
		return "sut"
	}
	return "ref"
}

// Compose runs `docker compose <action> [args…] <service>` on this session's
// side, e.g. s.Compose("restart", "redis") or s.Compose("kill", "node-b").
// The action is recorded as a note.
func (s *Session) Compose(action, base string, args ...string) string {
	s.T.Helper()
	node := Ref()
	if side(s.Target.Name) == "sut" {
		node = SUT()
	}
	if err := checkComposeEndpoint(node, action); err != nil {
		s.T.Fatal(err)
	}
	out, err := compose(append(append([]string{action}, args...), serviceFor(base, s.Target.Name))...)
	if err != nil {
		s.T.Fatalf("%s: %v", s.Target.Name, err)
	}
	s.Note(action + " " + base)
	return out
}

func checkComposeEndpoint(node Target, action string) error {
	binding, err := compose("port", node.Name, "8000")
	if err != nil {
		return fmt.Errorf("checking Compose endpoint for %s: %w", node.Name, err)
	}
	if !matchesEndpoint(node.BaseURL, strings.TrimSpace(binding)) {
		return fmt.Errorf("Compose publishes %s on %s but the test uses %s; check COMPAT_COMPOSE_FILE and COMPOSE_PROJECT_NAME before %s", node.Name, strings.TrimSpace(binding), node.BaseURL, action)
	}
	return nil
}

// Compose profiles publish to loopback. Compare both host and port so tests
// using a different stack cannot stop or restart the local project's services.
func matchesEndpoint(baseURL, binding string) bool {
	u, err := url.Parse(baseURL)
	if err != nil || u.Port() == "" {
		return false
	}
	host, port, err := net.SplitHostPort(binding)
	if err != nil || port != u.Port() {
		return false
	}
	return u.Hostname() == host || (u.Hostname() == "localhost" && net.ParseIP(host).IsLoopback())
}

// Exec runs a command inside this side's service, e.g.
// s.Exec("redis", "redis-cli", "PING"). It is not recorded.
func (s *Session) Exec(base string, cmd ...string) string {
	s.T.Helper()
	out, err := compose(append([]string{"exec", "-T", serviceFor(base, s.Target.Name)}, cmd...)...)
	if err != nil {
		s.T.Fatalf("%s: %v", s.Target.Name, err)
	}
	return out
}

// compose runs `docker compose <args…>` on test/compat/docker-compose.yml
// (addressed from the scenarios package directory, where go test runs) with
// the current profile.
func compose(args ...string) (string, error) {
	return docker(append([]string{"compose", "-f", envOr("COMPAT_COMPOSE_FILE", "../docker-compose.yml"), "--profile", Profile()}, args...)...)
}

func docker(args ...string) (string, error) {
	out, err := exec.Command("docker", args...).CombinedOutput() //nolint:gosec // fixed binary, args from harness and scenario code
	if err != nil {
		return "", fmt.Errorf("docker %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return string(out), nil
}

// StartFails starts a one-off copy of this side's node A with extra env
// settings (KEY=VALUE), expects it to exit, and records the exit status and
// the fatal log line (level, error, message; time dropped).
func (s *Session) StartFails(env ...string) {
	s.T.Helper()
	args := []string{"compose", "-f", envOr("COMPAT_COMPOSE_FILE", "../docker-compose.yml"), "--profile", Profile(), "run", "--rm", "--no-deps"}
	for _, e := range env {
		args = append(args, "-e", e)
	}
	args = append(args, serviceFor("node", s.Target.Name))
	out, err := exec.Command("docker", args...).CombinedOutput() //nolint:gosec // fixed binary, args from scenario code
	var ee *exec.ExitError
	if err != nil && !errors.As(err, &ee) {
		s.T.Fatalf("%s: docker %s: %v", s.Target.Name, strings.Join(args, " "), err)
	}
	var fatal json.RawMessage
	for _, line := range strings.Split(string(out), "\n") {
		var l struct {
			Level   string `json:"level"`
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal([]byte(line), &l) == nil && l.Level == "fatal" {
			fatal, _ = json.Marshal(l)
		}
	}
	code := 0
	if ee != nil {
		code = ee.ExitCode()
	}
	s.tr.add(Event{Kind: "exit", Text: strings.Join(env, " "), Code: code, Body: fatal})
}
