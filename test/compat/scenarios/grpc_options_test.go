package scenarios

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	api "github.com/xantos008/centrifunahuy-na-go/internal/apiproto"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	reflection "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestCS_API_05(t *testing.T) {
	h.Profiles(t, "m", "rsh")
	h.Run(t, "CS-API-05", func(s *h.Session) {
		ctx, cancel := context.WithTimeout(s.T.Context(), h.Deadline)
		defer cancel()
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "apikey "+h.APIKey)
		conn := s.GRPCConn()
		client := api.NewCentrifugoApiClient(conn)
		tlsEnabled := h.Profile() == "rsh"

		var remote peer.Peer
		reply, err := client.Publish(ctx, &api.PublishRequest{Channel: s.Ch("pub:grpc"), Data: []byte("{}")}, grpc.Peer(&remote))
		if err != nil || reply.GetResult() == nil || reply.GetError() != nil {
			s.T.Fatalf("publish failed: reply %v, status %v", reply, err)
		}
		if tlsEnabled {
			info, ok := remote.AuthInfo.(credentials.TLSInfo)
			if !ok || len(info.State.VerifiedChains) == 0 || info.State.NegotiatedProtocol != "h2" {
				s.T.Fatal("expected an HTTP/2 connection with a verified TLS certificate chain")
			}
			s.Note("Publish OK over verified TLS with ALPN " + info.State.NegotiatedProtocol)
		} else {
			s.Note("Publish OK over plaintext")
		}
		checkPublishReflection(s, ctx, conn, tlsEnabled)

		// Same command under the profile default, then with a per-request override.
		for _, override := range []bool{false, true} {
			callCtx := ctx
			if override {
				callCtx = metadata.AppendToOutgoingContext(ctx, "x-centrifugo-error-mode", "transport")
			}
			reply, err := client.Publish(callCtx, &api.PublishRequest{})
			var detail *api.Error
			if tlsEnabled || override {
				st := status.Convert(err)
				if reply != nil || st.Code() != codes.InvalidArgument || st.Message() != "bad request" || len(st.Details()) != 1 {
					s.T.Fatalf("unexpected Publish transport error: reply %v, status %v", reply, err)
				}
				detail, _ = st.Details()[0].(*api.Error)
			} else {
				if err != nil || reply.GetResult() != nil {
					s.T.Fatalf("unexpected Publish envelope: reply %v, status %v", reply, err)
				}
				detail = reply.GetError()
			}
			if detail.GetCode() != 107 || detail.GetMessage() != "bad request" || detail.GetTemporary() {
				s.T.Fatalf("unexpected application error in Publish: %v", detail)
			}
			s.Note(fmt.Sprintf("Publish transport metadata=%t status=%s error=%d %s temporary=%t",
				override, status.Code(err), detail.GetCode(), detail.GetMessage(), detail.GetTemporary()))
		}
	})
}

func checkPublishReflection(s *h.Session, ctx context.Context, conn *grpc.ClientConn, enabled bool) {
	s.T.Helper()
	stream, err := reflection.NewServerReflectionClient(conn).ServerReflectionInfo(ctx)
	if err != nil {
		s.T.Fatal(err)
	}
	service := "centrifugal.centrifugo.api.CentrifugoApi"
	req := &reflection.ServerReflectionRequest{
		MessageRequest: &reflection.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: service + ".Publish"},
	}
	if err := stream.Send(req); err != nil && !errors.Is(err, io.EOF) {
		s.T.Fatal(err)
	}
	resp, err := stream.Recv()
	if !enabled {
		if status.Code(err) != codes.Unimplemented {
			s.T.Fatalf("disabled reflection: got %v, want Unimplemented", err)
		}
		s.Note("reflection status " + status.Code(err).String())
		return
	}
	if err != nil {
		s.T.Fatal(err)
	}
	for _, data := range resp.GetFileDescriptorResponse().GetFileDescriptorProto() {
		var file descriptorpb.FileDescriptorProto
		if err := proto.Unmarshal(data, &file); err != nil {
			s.T.Fatal(err)
		}
		for _, svc := range file.GetService() {
			if file.GetPackage()+"."+svc.GetName() != service {
				continue
			}
			for _, method := range svc.GetMethod() {
				if method.GetName() == "Publish" && method.GetInputType() == ".centrifugal.centrifugo.api.PublishRequest" &&
					method.GetOutputType() == ".centrifugal.centrifugo.api.PublishResponse" &&
					!method.GetClientStreaming() && !method.GetServerStreaming() {
					s.Note(fmt.Sprintf("reflection unary %s.%s(%s) returns %s",
						service, method.GetName(), method.GetInputType(), method.GetOutputType()))
					return
				}
			}
		}
	}
	s.T.Fatal("reflection did not return the expected unary Publish descriptor")
}
