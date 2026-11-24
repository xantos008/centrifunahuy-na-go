package scenarios

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/xantos008/centrifunahuy-na-go/internal/unigrpc/unistream"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
	"google.golang.org/grpc/metadata"
)

func TestCS_PROXY_14(t *testing.T) {
	h.Profiles(t, "p", "r")
	h.Run(t, "CS-PROXY-14", func(s *h.Session) {
		// A signed-token login exercises the same client and listener before
		// requiring connect callback data from the tokenless connection.
		control := s.OpenUniGRPC(&unistream.ConnectRequest{
			Token: h.HS256(h.Claims{Sub: "metadata-control-user"}),
		})
		if push := control(); push.GetConnect().GetClient() == "" {
			s.T.Fatalf("expected a successful uni gRPC control connection, got %v", push)
		}

		const key = "x-compat-connect-marker"
		markers := []string{"transport-first", "transport-second"}
		read := s.OpenUniGRPCWithMetadata(&unistream.ConnectRequest{
			Name:    "proxy-metadata",
			Headers: map[string]string{key: "frame"},
		}, metadata.MD{
			key:                         markers,
			"x-compat-connect-unlisted": {"transport-unlisted"},
		})
		push := read()
		connected := push.GetConnect()
		if connected.GetClient() == "" || connected.GetSession() == "" || connected.GetPong() {
			s.T.Fatalf("expected uni gRPC connect push from callback, got %v", push)
		}
		var observed struct {
			Transport  string
			Markers    []string
			Unlisted   bool
			Credential bool
			Gzip       bool
		}
		wantsGRPC := h.Profile() == "r"
		transport := "http"
		if wantsGRPC {
			transport = "grpc"
		}
		if err := json.Unmarshal(connected.GetData(), &observed); err != nil {
			s.T.Fatalf("invalid callback observation: %v", err)
		}
		if observed.Transport != transport || !slices.Equal(observed.Markers, markers) || observed.Unlisted ||
			observed.Credential != wantsGRPC || observed.Gzip != wantsGRPC {
			s.T.Fatalf("unexpected %s callback observation: %+v", transport, observed)
		}
		disconnectTransport(s, "metadata-proxy-user")
		if d := read().GetDisconnect(); d.GetCode() != 4001 || d.GetReason() != "test complete" {
			s.T.Fatalf("expected callback user disconnect, got %v", d)
		}
		if got := read(); got != nil {
			s.T.Fatalf("expected end of callback user stream, got %v", got)
		}
	})
}
