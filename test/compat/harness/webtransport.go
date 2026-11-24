package harness

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/centrifugal/protocol"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/webtransport-go"
)

// WT uses one bidirectional HTTP/3 stream per client protocol connection.
type WT struct {
	s      *Session
	stream *webtransport.Stream
	reader *bufio.Reader
	binary bool
}

// DialWebTransport returns nil when the peer does not negotiate HTTP/3 ALPN.
// The pinned server exhibits that specific TLS alert with a static certificate.
func (s *Session) DialWebTransport(protobuf bool) *WT {
	s.T.Helper()
	dir := filepath.Dir(envOr("COMPAT_COMPOSE_FILE", "../docker-compose.yml"))
	ca, err := fs.ReadFile(os.DirFS(dir), "configs/rsh/tls/ca.crt")
	if err != nil {
		s.T.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		s.T.Fatal("invalid compatibility CA")
	}
	transport := &webtransport.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS13}}
	s.T.Cleanup(func() { _ = transport.Close() })
	addr := envOr("COMPAT_REF_WT_ADDR", "127.0.0.1:18443")
	switch s.Target.Name {
	case "sut":
		addr = envOr("COMPAT_SUT_WT_ADDR", "127.0.0.1:28443")
	case "sut-default":
		addr = envOr("COMPAT_SUT_DEFAULT_WT_ADDR", "127.0.0.1:28444")
	}
	path := "/connection/webtransport"
	if protobuf {
		path += "?cf_protocol=protobuf"
	}
	ctx, cancel := context.WithTimeout(s.T.Context(), Deadline)
	defer cancel()
	response, session, err := transport.Dial(ctx, "https://"+addr+path, nil)
	if response != nil && response.Body != nil {
		s.T.Cleanup(func() { _ = response.Body.Close() })
	}
	if err != nil {
		var transportErr *quic.TransportError
		// QUIC CRYPTO_ERROR (0x100) plus TLS no_application_protocol (120).
		if errors.As(err, &transportErr) && transportErr.ErrorCode == 0x178 && !transportErr.Remote {
			s.tr.add(Event{Kind: "webtransport_handshake", Text: path + " TLS no_application_protocol"})
			return nil
		}
		s.T.Fatalf("%s: WebTransport handshake: %v", s.Target.Name, err)
	}
	s.T.Cleanup(func() { _ = session.CloseWithError(0, "test complete") })
	s.tr.add(Event{Kind: "webtransport_connect", Text: path, Status: response.StatusCode})
	if response.StatusCode != http.StatusOK || response.ProtoMajor != 3 {
		s.T.Fatalf("unexpected handshake: %s %s", response.Proto, response.Status)
	}
	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		s.T.Fatal(err)
	}
	return &WT{s: s, stream: stream, reader: bufio.NewReader(stream), binary: protobuf}
}

func (c *WT) Send(command *protocol.Command) {
	c.s.T.Helper()
	frame, err := protocol.NewJSONCommandEncoder().Encode(command)
	if err != nil {
		c.s.T.Fatal(err)
	}
	c.s.tr.add(Event{Kind: "webtransport_send", Body: json.RawMessage(frame)})
	if c.binary {
		frame, err = protocol.NewProtobufCommandEncoder().Encode(command)
		if err != nil {
			c.s.T.Fatal(err)
		}
	} else {
		frame = append(frame, '\n')
	}
	if err := c.stream.SetWriteDeadline(time.Now().Add(Deadline)); err != nil {
		c.s.T.Fatal(err)
	}
	if _, err := c.stream.Write(frame); err != nil {
		c.s.T.Fatal(err)
	}
}

func (c *WT) Recv() *protocol.Reply {
	c.s.T.Helper()
	if err := c.stream.SetReadDeadline(time.Now().Add(Deadline)); err != nil {
		c.s.T.Fatal(err)
	}
	var reply protocol.Reply
	if c.binary {
		size, err := binary.ReadUvarint(c.reader)
		if err != nil || size > 65536 {
			c.s.T.Fatalf("invalid protobuf reply length %d: %v", size, err)
		}
		frame := make([]byte, size)
		if _, err := io.ReadFull(c.reader, frame); err != nil {
			c.s.T.Fatal(err)
		}
		if err := reply.UnmarshalCF(frame); err != nil {
			c.s.T.Fatal(err)
		}
	} else {
		line, err := c.reader.ReadSlice('\n')
		if err != nil {
			c.s.T.Fatal(err)
		}
		if err := json.Unmarshal(line, &reply); err != nil {
			c.s.T.Fatal(err)
		}
	}
	body, err := protocol.NewJSONReplyEncoder().Encode(&reply)
	if err != nil {
		c.s.T.Fatal(err)
	}
	c.s.tr.add(Event{Kind: "webtransport_recv", Body: json.RawMessage(body)})
	return &reply
}
