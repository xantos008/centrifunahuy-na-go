package harness

import (
	"slices"
	"testing"
)

func TestServiceForStaysOnOwnSide(t *testing.T) {
	cases := []struct{ base, target, want string }{
		{"node", "ref", "ref"},
		{"node", "sut", "sut"},
		{"node-b", "ref", "ref-b"},
		{"node-b", "sut", "sut-b"},
		{"redis", "ref", "redis-ref"},
		{"redis", "sut", "redis-sut"},
		{"redis", "sut-default", "redis-sut"},
	}
	for _, c := range cases {
		if got := serviceFor(c.base, c.target); got != c.want {
			t.Errorf("serviceFor(%q, %q) = %q, want %q", c.base, c.target, got, c.want)
		}
	}
}

func TestMatchesEndpoint(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		binding string
		want    bool
	}{
		{"matching address", "http://127.0.0.1:18000", "127.0.0.1:18000", true},
		{"matching hostname", "http://localhost:18000", "localhost:18000", true},
		{"localhost IPv4 alias", "http://localhost:18000", "127.0.0.1:18000", true},
		{"localhost IPv6 alias", "http://localhost:18000", "[::1]:18000", true},
		{"localhost wrong port", "http://localhost:18500", "127.0.0.1:18000", false},
		{"loopback wrong port", "http://127.0.0.1:18500", "127.0.0.1:18000", false},
		{"other host", "http://192.0.2.1:18000", "127.0.0.1:18000", false},
		{"localhost with non-loopback binding", "http://localhost:18000", "192.0.2.1:18000", false},
		{"malformed binding", "http://127.0.0.1:18000", "127.0.0.1", false},
		{"URL missing port", "http://127.0.0.1", "127.0.0.1:18000", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := matchesEndpoint(c.baseURL, c.binding); got != c.want {
				t.Errorf("matchesEndpoint(%q, %q) = %t, want %t", c.baseURL, c.binding, got, c.want)
			}
		})
	}
}

func TestRecordListIsExplicit(t *testing.T) {
	t.Setenv("COMPAT_MODE", "record")
	t.Setenv("COMPAT_RECORD", "")
	if CheckRecordList() == nil {
		t.Error("record mode without COMPAT_RECORD must fail")
	}
	t.Setenv("COMPAT_RECORD", " CS-HIST-01, ,CS-REC-01 ")
	if err := CheckRecordList(); err != nil {
		t.Fatal(err)
	}
	if got := recordList(); !slices.Equal(got, []string{"CS-HIST-01", "CS-REC-01"}) {
		t.Errorf("recordList() = %v", got)
	}
}

func TestProfileNodes(t *testing.T) {
	t.Setenv("COMPAT_PROFILE", "nope")
	if CheckProfile() == nil {
		t.Error("unknown profile must fail")
	}
	t.Setenv("COMPAT_PROFILE", "m")
	if err := CheckProfile(); err != nil {
		t.Fatal(err)
	}
	if n := len(SUTNodes()); n != 1 {
		t.Errorf("m: %d sut nodes, want 1", n)
	}
	profiles["two"] = profileSpec{nodes: 2}
	defer delete(profiles, "two")
	t.Setenv("COMPAT_PROFILE", "two")
	if n := len(RefNodes()); n != 2 {
		t.Errorf("two-node profile: %d ref nodes, want 2", n)
	}
}

func TestProfileProbes(t *testing.T) {
	for name, p := range profiles {
		if len(p.backends) > 0 && len(p.probe) == 0 {
			t.Errorf("%s: backends without a CS-ISO-01 probe", name)
		}
		for _, b := range p.probe {
			if !slices.Contains(p.backends, b) {
				t.Errorf("%s: probe %q is not a backend", name, b)
			}
		}
	}
}
