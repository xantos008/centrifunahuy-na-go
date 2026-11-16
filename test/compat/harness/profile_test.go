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
