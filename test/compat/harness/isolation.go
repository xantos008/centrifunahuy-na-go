package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	pb "cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/xantos008/centrifunahuy-na-go/test/compat/internal/pubsubfixture"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	for _, node := range []Target{Ref(), SUT()} {
		if err := checkComposeEndpoint(node, "isolation probes"); err != nil {
			return err
		}
	}
	servers := map[string][]string{"ref": {"ref"}, "sut": {"sut", "sut-default"}}
	if spec.noSUTDefault {
		servers["sut"] = []string{"sut"}
	}
	if spec.nodes == 2 {
		servers["ref"] = append(servers["ref"], "ref-b")
		servers["sut"] = append(servers["sut"], "sut-b")
	}
	for side, svcs := range servers {
		net := envOr("COMPOSE_PROJECT_NAME", "cedrazeka-compat") + "_" + side + "-net"
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
func probe(b string) error {
	switch b {
	case "pubsub":
		return probePubSub()
	case "localstack":
		return probeSQS()
	case "kafka":
		return probeKafka()
	case "nats":
		return probeNATS()
	case "postgres":
		return probePostgres()
	}
	cli := func(side string, args ...string) (string, error) {
		pre := []string{"exec", "-T", b + "-" + side, "redis-cli"}
		if b == "sentinel" {
			// The current master, which changes with a failover (CS-ENG-07).
			out, err := compose("exec", "-T", b+"-"+side, "redis-cli", "-p", "26379", "sentinel", "get-master-addr-by-name", "mymaster")
			addr := strings.Fields(out)
			if err != nil || len(addr) != 2 {
				return "", fmt.Errorf("CS-ISO-01: sentinel-%s master: %v %q", side, err, out)
			}
			pre = append(pre, "-h", addr[0], "-p", addr[1])
		}
		out, err := compose(append(append(pre, "-c", "--no-auth-warning",
			"--user", "compat", "--pass", "compat-test-redis-password"), args...)...)
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

// Create a queue on each side in turn and compare name lookups across emulators.
func probeSQS() error {
	query := func(side string, args ...string) (string, error) {
		return compose(append([]string{"exec", "-T", "localstack-" + side, "awslocal", "sqs"}, args...)...)
	}
	for _, pair := range [][2]string{{"ref", "sut"}, {"sut", "ref"}} {
		name := "compat-iso-" + pair[0] + "-" + runID
		url, err := query(pair[0], "create-queue", "--queue-name", name, "--query", "QueueUrl", "--output", "text")
		if err != nil {
			return err
		}
		for i, side := range pair {
			got, err := query(side, "list-queues", "--queue-name-prefix", name, "--query", "QueueUrls", "--output", "text")
			if err != nil {
				return err
			}
			if strings.Contains(got, "/"+name) != (i == 0) {
				return fmt.Errorf("CS-ISO-01: queue %s created on %s, lookup on %s returned %q", name, pair[0], side, got)
			}
		}
		if _, err := query(pair[0], "delete-queue", "--queue-url", strings.TrimSpace(url)); err != nil {
			return err
		}
	}
	return nil
}

func probeKafka() error {
	topics := func(side string, args ...string) (string, error) {
		return compose(append([]string{"exec", "-T", "kafka-" + side, "kafka-topics", "--bootstrap-server", "kafka:29092"}, args...)...)
	}
	for _, pair := range [][2]string{{"ref", "sut"}, {"sut", "ref"}} {
		name := "compat-iso-" + pair[0] + "-" + runID
		if _, err := topics(pair[0], "--create", "--topic", name, "--partitions", "1", "--replication-factor", "1"); err != nil {
			return err
		}
		for i, side := range pair {
			list, err := topics(side, "--list")
			if err != nil {
				return err
			}
			if slices.Contains(strings.Fields(list), name) != (i == 0) {
				return fmt.Errorf("CS-ISO-01: topic %s created on %s, lookup on %s returned %q", name, pair[0], side, list)
			}
		}
		if _, err := topics(pair[0], "--delete", "--topic", name); err != nil {
			return err
		}
	}
	return nil
}

// Create a topic via each emulator and require NotFound from the opposite side.
func probePubSub() error {
	clients := make(map[string]pb.PublisherClient, 2)
	for _, side := range []string{"ref", "sut"} {
		conn, err := pubsubfixture.Dial(side)
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close() }()
		clients[side] = pb.NewPublisherClient(conn)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, pair := range [][2]string{{"ref", "sut"}, {"sut", "ref"}} {
		name := "projects/compat-project/topics/compat-iso-" + pair[0] + "-" + runID
		if _, err := clients[pair[0]].CreateTopic(ctx, &pb.Topic{Name: name}); err != nil {
			return fmt.Errorf("CS-ISO-01: create topic on %s: %w", pair[0], err)
		}
		_, lookupErr := clients[pair[1]].GetTopic(ctx, &pb.GetTopicRequest{Topic: name})
		_, deleteErr := clients[pair[0]].DeleteTopic(ctx, &pb.DeleteTopicRequest{Topic: name})
		if status.Code(lookupErr) != codes.NotFound {
			return fmt.Errorf("CS-ISO-01: lookup of %s on %s returned %v, want NotFound", name, pair[1], lookupErr)
		}
		if deleteErr != nil {
			return deleteErr
		}
	}
	return nil
}
