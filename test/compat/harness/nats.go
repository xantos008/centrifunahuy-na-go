package harness

import (
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

// AwaitNATSSubscriptions waits for the broker to process the asynchronous
// subscription requests before another node publishes to their subjects.
func (s *Session) AwaitNATSSubscriptions(subjects ...string) {
	s.T.Helper()
	deadline := time.Now().Add(Deadline)
	for {
		list := s.Exec("nats", "wget", "-qO-", "http://127.0.0.1:8222/subsz?subs=1")
		missing := ""
		for _, subject := range subjects {
			if !strings.Contains(list, `"`+subject+`"`) {
				missing = subject
				break
			}
		}
		if missing == "" {
			return
		}
		if time.Now().After(deadline) {
			s.T.Fatalf("%s: NATS did not register %s", s.Target.Name, missing)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// natsURL is the loopback client port of one side's NATS (compose: ref
// 14222, sut 24222), with the test-only user.
func natsURL(side string) string {
	port := map[string]string{"ref": "14222", "sut": "24222"}[side]
	return envOr("COMPAT_"+map[string]string{"ref": "REF", "sut": "SUT"}[side]+"_NATS_URL",
		"nats://compat:compat-test-nats-password@127.0.0.1:"+port)
}

// NATSPublish publishes data to a raw subject on this session's side's NATS
// (condition C2: the side follows the target). Recorded as a note.
func (s *Session) NATSPublish(subject string, data []byte) {
	s.T.Helper()
	nc, err := nats.Connect(natsURL(side(s.Target.Name)), nats.Timeout(Deadline))
	if err != nil {
		s.T.Fatalf("%s: nats: %v", s.Target.Name, err)
	}
	defer nc.Close()
	if err := nc.Publish(subject, data); err != nil {
		s.T.Fatalf("%s: nats publish: %v", s.Target.Name, err)
	}
	if err := nc.Flush(); err != nil {
		s.T.Fatalf("%s: nats flush: %v", s.Target.Name, err)
	}
	s.Note("nats publish " + subject)
}

// NATSSubscribe subscribes to a raw subject on this session's side's NATS
// and returns next, which waits for one message and records it as a note
// ("nats recv <subject> <data>"). The connection closes with the test.
func (s *Session) NATSSubscribe(subject string) (next func()) {
	s.T.Helper()
	nc, err := nats.Connect(natsURL(side(s.Target.Name)), nats.Timeout(Deadline))
	if err != nil {
		s.T.Fatalf("%s: nats: %v", s.Target.Name, err)
	}
	s.T.Cleanup(nc.Close)
	sub, err := nc.SubscribeSync(subject)
	if err == nil {
		err = nc.Flush()
	}
	if err != nil {
		s.T.Fatalf("%s: nats subscribe: %v", s.Target.Name, err)
	}
	return func() {
		s.T.Helper()
		m, err := sub.NextMsg(Deadline)
		if err != nil {
			s.T.Fatalf("%s: nats recv %s: %v", s.Target.Name, subject, err)
		}
		s.Note("nats recv " + m.Subject + " " + string(m.Data))
	}
}

// probeNATS subscribes on both sides, publishes through one, and expects
// only that side to receive it (CS-ISO-01).
func probeNATS() error {
	conns := map[string]*nats.Conn{}
	subs := map[string]*nats.Subscription{}
	for _, sd := range []string{"ref", "sut"} {
		nc, err := nats.Connect(natsURL(sd), nats.Timeout(Deadline))
		if err != nil {
			return fmt.Errorf("CS-ISO-01: nats-%s: %w", sd, err)
		}
		defer nc.Close()
		sub, err := nc.SubscribeSync("compat-iso-probe")
		if err != nil {
			return err
		}
		if err := nc.Flush(); err != nil {
			return err
		}
		conns[sd], subs[sd] = nc, sub
	}
	for _, pair := range [][2]string{{"ref", "sut"}, {"sut", "ref"}} {
		if err := conns[pair[0]].Publish("compat-iso-probe", []byte(pair[0])); err != nil {
			return err
		}
		if _, err := subs[pair[0]].NextMsg(Deadline); err != nil {
			return fmt.Errorf("CS-ISO-01: probe via nats-%s not received there: %w", pair[0], err)
		}
		if m, err := subs[pair[1]].NextMsg(500 * time.Millisecond); err == nil {
			return fmt.Errorf("CS-ISO-01: probe via nats-%s reached nats-%s (%q)", pair[0], pair[1], m.Data)
		}
	}
	return nil
}
