package harness

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/xantos008/centrifunahuy-na-go/internal/apiproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// GRPCConn opens the side's server API using the listener settings for the profile.
func (s *Session) GRPCConn() *grpc.ClientConn {
	s.T.Helper()
	addr := envOr("COMPAT_REF_GRPC_ADDR", "127.0.0.1:18010")
	if s.Target.Name == "sut" {
		addr = envOr("COMPAT_SUT_GRPC_ADDR", "127.0.0.1:28010")
	}
	creds := insecure.NewCredentials()
	if Profile() == "rsh" {
		dir := filepath.Dir(envOr("COMPAT_COMPOSE_FILE", "../docker-compose.yml"))
		tlsCreds, err := credentials.NewClientTLSFromFile(filepath.Join(dir, "configs", "rsh", "tls", "ca.crt"), "")
		if err != nil {
			s.T.Fatal(err)
		}
		creds = tlsCreds
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		s.T.Fatal(err)
	}
	s.T.Cleanup(func() { _ = conn.Close() })
	return conn
}

// GRPC opens the side's server API. The concrete generated messages encode
// the HTTP fixtures' application JSON fields as raw protobuf bytes.
func (s *Session) GRPC() func(method, body string, authenticated bool, want codes.Code) {
	s.T.Helper()
	conn := s.GRPCConn()
	service := apiproto.File_api_proto.Services().ByName("CentrifugoApi")
	methods := make(map[string]protoreflect.MethodDescriptor)
	for i := 0; i < service.Methods().Len(); i++ {
		m := service.Methods().Get(i)
		methods[strings.ToLower(string(m.Name()))] = m
	}
	newMessage := func(desc protoreflect.MessageDescriptor) protoreflect.ProtoMessage {
		t, err := protoregistry.GlobalTypes.FindMessageByName(desc.FullName())
		if err != nil {
			s.T.Fatal(err)
		}
		return t.New().Interface()
	}
	return func(method, body string, authenticated bool, want codes.Code) {
		s.T.Helper()
		m := methods[strings.ReplaceAll(method, "_", "")]
		if m == nil {
			s.T.Fatalf("no server API method for %q", method)
		}
		in, out := newMessage(m.Input()), newMessage(m.Output())
		if err := json.Unmarshal([]byte(body), in); err != nil {
			s.T.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(s.T.Context(), Deadline)
		defer cancel()
		if authenticated {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "apikey "+APIKey)
		}
		path := "/" + string(service.FullName()) + "/" + string(m.Name())
		s.tr.add(Event{Kind: "grpc_send", Text: path, Body: json.RawMessage(body)})
		err := conn.Invoke(ctx, path, in, out)
		code := status.Code(err)
		if code != want {
			s.T.Fatalf("%s %s: got %v, want %v", s.Target.Name, path, err, want)
		}
		if err != nil {
			s.tr.add(Event{Kind: "grpc_status", Code: int(code), Text: status.Convert(err).Message()})
			return
		}
		result, err := json.Marshal(out)
		if err != nil {
			s.T.Fatal(err)
		}
		if method == "info" {
			result = shape(omit(result, []string{"metrics", "uptime"}))
		}
		s.tr.add(Event{Kind: "grpc_reply", Body: result})
	}
}
