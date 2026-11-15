// Package e2e checks the brand modes end to end through real upstream code
// (config loading, gRPC registration). It is a separate test-only package
// because the upstream test packages pin compatibility mode (ADR-0022).
package e2e

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/xantos008/centrifunahuy-na-go/internal/apiproto"
	"github.com/xantos008/centrifunahuy-na-go/internal/brand"
	"github.com/xantos008/centrifunahuy-na-go/internal/config"
)

func withMode(t *testing.T, m brand.Mode) {
	t.Helper()
	prev := brand.Current()
	brand.Set(m)
	t.Cleanup(func() { brand.Set(prev) })
}

func TestConfig_DefaultMode(t *testing.T) {
	withMode(t, brand.ModeDefault)
	t.Setenv("CEDRAZEKA_LOG_LEVEL", "debug")
	t.Setenv("CENTRIFUGO_HTTP_SERVER_PORT", "9999")

	conf, meta, err := config.GetConfig(nil, "")
	require.NoError(t, err)
	require.Equal(t, "debug", conf.Log.Level, "CEDRAZEKA_* env is read")
	require.NotEqual(t, 9999, conf.HTTP.Port, "CENTRIFUGO_* env is not read in default mode")
	require.Contains(t, meta.UnknownEnvs, "CENTRIFUGO_HTTP_SERVER_PORT", "and is reported")
	require.Contains(t, meta.KnownEnvVars, "CEDRAZEKA_LOG_LEVEL")
	require.Equal(t, "cedrazeka", conf.Engine.Redis.Prefix)
}

func TestConfig_CompatMode(t *testing.T) {
	withMode(t, brand.ModeCentrifugo)
	t.Setenv("CENTRIFUGO_LOG_LEVEL", "debug")
	t.Setenv("CEDRAZEKA_HTTP_SERVER_PORT", "9999")

	conf, meta, err := config.GetConfig(nil, "")
	require.NoError(t, err)
	require.Equal(t, "debug", conf.Log.Level)
	require.NotEqual(t, 9999, conf.HTTP.Port)
	require.NotContains(t, meta.UnknownEnvs, "CEDRAZEKA_HTTP_SERVER_PORT", "only the active prefix is scanned")
	require.Equal(t, "centrifugo", conf.Engine.Redis.Prefix)
}

func TestConfig_ExplicitValueKept(t *testing.T) {
	withMode(t, brand.ModeDefault)
	t.Setenv("CEDRAZEKA_ENGINE_REDIS_PREFIX", "centrifugo")
	conf, _, err := config.GetConfig(nil, "")
	require.NoError(t, err)
	require.Equal(t, "centrifugo", conf.Engine.Redis.Prefix, "an explicit value is never rewritten")
}

// callPublish calls Publish on the given full service name and returns the gRPC status.
func callPublish(t *testing.T, service string) *status.Status {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	brand.RegisterGRPCService(srv, &apiproto.CentrifugoApi_ServiceDesc, apiproto.UnimplementedCentrifugoApiServer{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	err = conn.Invoke(context.Background(), "/"+service+"/Publish", &apiproto.PublishRequest{}, &apiproto.PublishResponse{})
	st, _ := status.FromError(err)
	return st
}

// Registered services answer "method Publish not implemented"; unknown ones "unknown service".
func served(st *status.Status) bool {
	return st.Code() == codes.Unimplemented && st.Message() == "method Publish not implemented"
}

func TestGRPCAPI_DefaultMode(t *testing.T) {
	withMode(t, brand.ModeDefault)
	require.True(t, served(callPublish(t, brand.APIService)))
	require.False(t, served(callPublish(t, brand.UpstreamAPIService)))
}

func TestGRPCAPI_CompatMode(t *testing.T) {
	withMode(t, brand.ModeCentrifugo)
	require.True(t, served(callPublish(t, brand.APIService)))
	require.True(t, served(callPublish(t, brand.UpstreamAPIService)))
}
