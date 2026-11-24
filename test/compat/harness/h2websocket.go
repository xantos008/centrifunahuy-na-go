package harness

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"net"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

// H2WebSocketConnect checks the negotiated RFC 8441 setting, then exchanges
// the first client-protocol message on the extended CONNECT stream. The Go
// HTTP client rejects :protocol when it validates application header names.
func (s *Session) H2WebSocketConnect(path, payload string) json.RawMessage {
	s.T.Helper()
	addr := envOr("COMPAT_REF_H2_ADDR", "127.0.0.1:18082")
	if s.Target.Name == "sut" {
		addr = envOr("COMPAT_SUT_H2_ADDR", "127.0.0.1:28082")
	}
	conn, err := net.DialTimeout("tcp", addr, Deadline)
	if err != nil {
		s.T.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if err = conn.SetDeadline(time.Now().Add(Deadline)); err != nil {
		s.T.Fatal(err)
	}
	fr := http2.NewFramer(conn, conn)
	fr.ReadMetaHeaders = hpack.NewDecoder(4096, nil)
	if _, err = conn.Write([]byte(http2.ClientPreface)); err != nil {
		s.T.Fatal(err)
	}
	if err = fr.WriteSettings(); err != nil {
		s.T.Fatal(err)
	}
	var enabled bool
	read := func() http2.Frame {
		s.T.Helper()
		f, err := fr.ReadFrame()
		if err != nil {
			s.T.Fatal(err)
		}
		switch x := f.(type) {
		case *http2.SettingsFrame:
			if !x.IsAck() {
				if value, ok := x.Value(http2.SettingEnableConnectProtocol); ok {
					enabled = value == 1
				}
				if err := fr.WriteSettingsAck(); err != nil {
					s.T.Fatal(err)
				}
			}
		case *http2.GoAwayFrame:
			s.T.Fatalf("GOAWAY %v: %s", x.ErrCode, x.DebugData())
		case *http2.RSTStreamFrame:
			s.T.Fatalf("stream reset: %v", x.ErrCode)
		case *http2.PingFrame:
			if !x.IsAck() {
				if err := fr.WritePing(true, x.Data); err != nil {
					s.T.Fatal(err)
				}
			}
		}
		return f
	}
	for !enabled {
		read()
	}
	var block bytes.Buffer
	enc := hpack.NewEncoder(&block)
	for _, field := range []hpack.HeaderField{
		{Name: ":method", Value: "CONNECT"}, {Name: ":scheme", Value: "http"}, {Name: ":authority", Value: addr},
		{Name: ":path", Value: path}, {Name: ":protocol", Value: "websocket"}, {Name: "sec-websocket-version", Value: "13"},
	} {
		if err := enc.WriteField(field); err != nil {
			s.T.Fatal(err)
		}
	}
	if err = fr.WriteHeaders(http2.HeadersFrameParam{StreamID: 1, BlockFragment: block.Bytes(), EndHeaders: true}); err != nil {
		s.T.Fatal(err)
	}
	for {
		header, ok := read().(*http2.MetaHeadersFrame)
		if !ok {
			continue
		}
		if header.StreamID != 1 || header.PseudoValue("status") != "200" || header.StreamEnded() {
			s.T.Fatalf("extended CONNECT %s: %v", path, header.Fields)
		}
		break
	}
	s.tr.add(Event{Kind: "h2_websocket_handshake", Text: path, Status: 200})
	data := []byte(payload)
	var frame []byte
	switch {
	case len(data) < 126:
		frame = []byte{0x81, 0x80 | byte(len(data))}
	case len(data) <= 65535:
		frame = binary.BigEndian.AppendUint16([]byte{0x81, 0xfe}, uint16(len(data))) //nolint:gosec // bounded above
	default:
		s.T.Fatal("connect request exceeds test frame limit")
	}
	var mask [4]byte
	if _, err = rand.Read(mask[:]); err != nil {
		s.T.Fatal(err)
	}
	frame = append(frame, mask[:]...)
	for i, b := range data {
		frame = append(frame, b^mask[i%4])
	}
	if err = fr.WriteData(1, false, frame); err != nil {
		s.T.Fatal(err)
	}
	s.tr.add(Event{Kind: "h2_websocket_send", Body: json.RawMessage(payload)})
	var received []byte
	for {
		f, ok := read().(*http2.DataFrame)
		if !ok {
			continue
		}
		received = append(received, f.Data()...)
		if f.StreamEnded() && len(received) == 0 {
			s.T.Fatal("HTTP/2 stream ended before WebSocket connect reply")
		}
		if len(received) < 2 {
			continue
		}
		size, start := int(received[1]), 2
		if received[0] != 0x81 || size >= 127 {
			s.T.Fatalf("unexpected WebSocket frame %x", received[:2])
		}
		if size == 126 {
			if len(received) < 4 {
				continue
			}
			size, start = int(binary.BigEndian.Uint16(received[2:4])), 4
		}
		if len(received) < start+size {
			continue
		}
		reply := json.RawMessage(received[start : start+size])
		if !json.Valid(reply) {
			s.T.Fatalf("invalid WebSocket JSON: %q", reply)
		}
		s.tr.add(Event{Kind: "h2_websocket_recv", Body: reply})
		return reply
	}
}
