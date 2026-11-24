package harness

import (
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// JetStream connects to the NATS server on the current target's side.
func (s *Session) JetStream() jetstream.JetStream {
	s.T.Helper()
	nc, err := nats.Connect(natsURL(side(s.Target.Name)), nats.Timeout(Deadline))
	if err != nil {
		s.T.Fatalf("%s: connect to NATS: %v", s.Target.Name, err)
	}
	s.T.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	if err != nil {
		s.T.Fatal(err)
	}
	return js
}
