package harness

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/xantos008/centrifunahuy-na-go/internal/unigrpc/unistream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// OpenUniGRPC consumes the public unistream service. A clean gRPC completion
// yields nil. Failures, including a missed disconnect followed by timeout, fail the test.
func (s *Session) OpenUniGRPC(req *unistream.ConnectRequest) func() *unistream.Push {
	s.T.Helper()
	return s.OpenUniGRPCWithMetadata(req, nil)
}

// OpenUniGRPCWithMetadata attaches transport metadata to the unistream call.
func (s *Session) OpenUniGRPCWithMetadata(req *unistream.ConnectRequest, md metadata.MD) func() *unistream.Push {
	s.T.Helper()
	addr := envOr("COMPAT_REF_UNI_GRPC_ADDR", "127.0.0.1:18011")
	if s.Target.Name == "sut" {
		addr = envOr("COMPAT_SUT_UNI_GRPC_ADDR", "127.0.0.1:28011")
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		s.T.Fatal(err)
	}
	s.T.Cleanup(func() { _ = conn.Close() })
	ctx, cancel := context.WithTimeout(s.T.Context(), Deadline)
	s.T.Cleanup(cancel)
	request, err := json.Marshal(req)
	if err != nil {
		s.T.Fatal(err)
	}
	s.tr.add(Event{Kind: "grpc_send", Text: unistream.CentrifugoUniStream_Consume_FullMethodName, Body: request})
	stream, err := unistream.NewCentrifugoUniStreamClient(conn).Consume(metadata.NewOutgoingContext(ctx, md), req)
	if err != nil {
		s.T.Fatalf("%s: consume: %v", s.Target.Name, err)
	}
	return func() *unistream.Push {
		s.T.Helper()
		push, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			s.tr.add(Event{Kind: "grpc_eof"})
			return nil
		}
		if err != nil {
			s.T.Fatalf("%s: receive unistream push: %v", s.Target.Name, err)
		}
		body, err := json.Marshal(push)
		if err != nil {
			s.T.Fatal(err)
		}
		s.tr.add(Event{Kind: "grpc_push", Body: body})
		return push
	}
}
