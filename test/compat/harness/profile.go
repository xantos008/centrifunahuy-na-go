package harness

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// Engine profiles (docs/plan/phase-2.md). COMPAT_PROFILE selects the one the
// targets were started with (`make compat-up PROFILE=…`); default "m".
var profiles = map[string]profileSpec{
	"m": {nodes: 1},
	"r": {nodes: 2, backends: []string{"redis"}},
}

type profileSpec struct {
	nodes    int      // server nodes per side
	backends []string // backend hostnames, one service per side ("redis" → redis-ref, redis-sut)
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
	side := "ref"
	if strings.HasPrefix(target, "sut") { // sut and sut-default share the sut side
		side = "sut"
	}
	switch base {
	case "node":
		return side
	case "node-b":
		return side + "-b"
	}
	return base + "-" + side
}

// Compose runs `docker compose <action> [args…] <service>` on this session's
// side, e.g. s.Compose("restart", "redis") or s.Compose("kill", "node-b").
// The action is recorded as a note.
func (s *Session) Compose(action, base string, args ...string) string {
	s.T.Helper()
	out, err := compose(append(append([]string{action}, args...), serviceFor(base, s.Target.Name))...)
	if err != nil {
		s.T.Fatalf("%s: %v", s.Target.Name, err)
	}
	s.Note(action + " " + base)
	return out
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
