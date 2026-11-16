package harness

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// CheckIsolation is CS-ISO-01 (condition C2 of the phase-2 plan). For a
// profile with backends it verifies, before any scenario runs, that
//   - every server container is attached only to its own side's network;
//   - each backend hostname resolves, inside every server container, to the
//     address of that side's backend container;
//   - the two sides' backends share no volume or bind mount;
//   - a probe written through one side's backend is invisible to the other.
func CheckIsolation() error {
	spec := profiles[Profile()]
	if len(spec.backends) == 0 {
		return nil
	}
	servers := map[string][]string{"ref": {"ref"}, "sut": {"sut", "sut-default"}}
	if spec.nodes == 2 {
		servers["ref"] = append(servers["ref"], "ref-b")
		servers["sut"] = append(servers["sut"], "sut-b")
	}
	for side, svcs := range servers {
		net := "cedrazeka-compat_" + side + "-net"
		for _, svc := range svcs {
			c, err := inspect(svc)
			if err != nil {
				return err
			}
			if got := slices.Sorted(maps.Keys(c.NetworkSettings.Networks)); !slices.Equal(got, []string{net}) {
				return fmt.Errorf("CS-ISO-01: %s is on networks %v, want only %s", svc, got, net)
			}
			for _, b := range spec.backends {
				bc, err := inspect(b + "-" + side)
				if err != nil {
					return err
				}
				want := bc.NetworkSettings.Networks[net].IPAddress
				out, err := compose("exec", "-T", svc, "getent", "hosts", b)
				if err != nil {
					return err
				}
				if f := strings.Fields(out); want == "" || len(f) == 0 || f[0] != want {
					return fmt.Errorf("CS-ISO-01: %q in %s resolves to %q, want %s-%s at %q", b, svc, strings.TrimSpace(out), b, side, want)
				}
			}
		}
	}
	for _, b := range spec.backends {
		r, err := inspect(b + "-ref")
		if err != nil {
			return err
		}
		s, err := inspect(b + "-sut")
		if err != nil {
			return err
		}
		for _, m := range r.Mounts {
			if slices.ContainsFunc(s.Mounts, func(o mount) bool { return o.Source == m.Source }) {
				return fmt.Errorf("CS-ISO-01: %s-ref and %s-sut share mount %s", b, b, m.Source)
			}
		}
	}
	for _, b := range spec.probe {
		if err := probe(b); err != nil {
			return err
		}
	}
	return nil
}

type container struct {
	NetworkSettings struct {
		Networks map[string]struct{ IPAddress string }
	}
	Mounts []mount
}

type mount struct{ Source string }

func inspect(svc string) (container, error) {
	id, err := compose("ps", "-q", svc)
	if err != nil {
		return container{}, err
	}
	if id = strings.TrimSpace(id); id == "" {
		return container{}, fmt.Errorf("CS-ISO-01: service %s is not running", svc)
	}
	out, err := docker("inspect", id)
	if err != nil {
		return container{}, err
	}
	var cs []container
	if err := json.Unmarshal([]byte(out), &cs); err != nil || len(cs) != 1 {
		return container{}, fmt.Errorf("CS-ISO-01: inspect %s: %v", svc, err)
	}
	return cs[0], nil
}

// probe writes a marker through each side's entry point b and expects the
// other side not to see it (Redis: -c follows cluster redirects).
// ponytail: Redis and NATS; the PostgreSQL probe comes with its profiles (step 2.6).
func probe(b string) error {
	if b == "nats" {
		return probeNATS()
	}
	cli := func(side string, args ...string) (string, error) {
		out, err := compose(append([]string{"exec", "-T", b + "-" + side, "redis-cli", "-c", "--no-auth-warning",
			"--user", "compat", "--pass", "compat-test-redis-password"}, args...)...)
		return strings.TrimSpace(out), err
	}
	for _, pair := range [][2]string{{"ref", "sut"}, {"sut", "ref"}} {
		key := "compat-iso-probe-" + pair[0]
		if _, err := cli(pair[0], "SET", key, "1", "EX", "60"); err != nil {
			return err
		}
		own, err := cli(pair[0], "EXISTS", key)
		if err != nil {
			return err
		}
		other, err := cli(pair[1], "EXISTS", key)
		if err != nil {
			return err
		}
		if own != "1" || other != "0" {
			return fmt.Errorf("CS-ISO-01: key written via %s-%s exists %s there, %s via %s-%s (want 1, 0)", b, pair[0], own, other, b, pair[1])
		}
		if _, err := cli(pair[0], "DEL", key); err != nil {
			return err
		}
	}
	return nil
}
