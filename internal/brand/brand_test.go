package brand

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func withMode(t *testing.T, m Mode) {
	t.Helper()
	prev := Current()
	Set(m)
	t.Cleanup(func() { Set(prev) })
}

func TestResolve(t *testing.T) {
	env := func(v string) func(string) string {
		return func(k string) string {
			if k == CompatEnv {
				return v
			}
			return ""
		}
	}
	cases := []struct {
		name    string
		args    []string
		env     string
		want    Mode
		wantErr bool
	}{
		{"none", nil, "", ModeDefault, false},
		{"env", nil, "centrifugo", ModeCentrifugo, false},
		{"flag eq", []string{"--compat=centrifugo"}, "", ModeCentrifugo, false},
		{"flag space", []string{"serve", "--compat", "centrifugo"}, "", ModeCentrifugo, false},
		{"single dash", []string{"-compat=centrifugo"}, "", ModeCentrifugo, false},
		{"flag empty wins over env", []string{"--compat="}, "centrifugo", ModeDefault, false},
		{"after --", []string{"--", "--compat=centrifugo"}, "", ModeDefault, false},
		{"bad flag", []string{"--compat=foo"}, "", ModeDefault, true},
		{"bad env", nil, "foo", ModeDefault, true},
		{"missing value", []string{"--compat"}, "", ModeDefault, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Resolve(c.args, env(c.env))
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.want, got)
		})
	}
}

func TestNames(t *testing.T) {
	withMode(t, ModeDefault)
	require.Equal(t, "cedrazeka", Ident())
	require.Equal(t, "CEDRAZEKA", EnvPrefix())
	require.Equal(t, "CEDRAZEKA_VAR_X", EnvVar("VAR_X"))
	require.Equal(t, "X-Cedrazeka-Error-Mode", Header("Error-Mode"))
	require.Equal(t, "cedrazeka", MapDefault("centrifugo"))
	require.Equal(t, "cedrazeka-tag-", MapDefault("centrifugo-tag-"))
	require.Equal(t, "cf_", MapDefault("cf_"))
	require.Equal(t, "centrifugoish", MapDefault("centrifugoish"))

	Set(ModeCentrifugo)
	require.Equal(t, "centrifugo", Ident())
	require.Equal(t, "CENTRIFUGO_VAR_X", EnvVar("VAR_X"))
	require.Equal(t, "X-Centrifugo-Error-Mode", Header("Error-Mode"))
	require.Equal(t, "centrifugo-tag-", MapDefault("centrifugo-tag-"))
}

func TestGRPCServiceDescs(t *testing.T) {
	up := &grpc.ServiceDesc{ServiceName: UpstreamAPIService}

	withMode(t, ModeDefault)
	ds := GRPCServiceDescs(up)
	require.Len(t, ds, 1)
	require.Equal(t, APIService, ds[0].ServiceName)
	require.Equal(t, UpstreamAPIService, up.ServiceName, "upstream desc must not be modified")

	Set(ModeCentrifugo)
	ds = GRPCServiceDescs(up)
	require.Len(t, ds, 2)
	require.Equal(t, APIService, ds[0].ServiceName)
	require.Equal(t, UpstreamAPIService, ds[1].ServiceName)

	other := &grpc.ServiceDesc{ServiceName: "x.Y"}
	require.Equal(t, []*grpc.ServiceDesc{other}, GRPCServiceDescs(other))
}

type recordConn struct{ method string }

func (r *recordConn) Invoke(_ context.Context, method string, _, _ any, _ ...grpc.CallOption) error {
	r.method = method
	return nil
}

func (r *recordConn) NewStream(_ context.Context, _ *grpc.StreamDesc, method string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
	r.method = method
	return nil, nil
}

func TestGRPCClientConn(t *testing.T) {
	ctx := context.Background()
	rc := &recordConn{}

	withMode(t, ModeDefault)
	cc := GRPCClientConn(rc)
	require.NoError(t, cc.Invoke(ctx, "/"+UpstreamProxyService+"/Connect", nil, nil))
	require.Equal(t, "/"+ProxyService+"/Connect", rc.method)
	_, _ = cc.NewStream(ctx, &grpc.StreamDesc{}, "/"+UpstreamProxyService+"/SubscribeUnidirectional")
	require.Equal(t, "/"+ProxyService+"/SubscribeUnidirectional", rc.method)
	require.NoError(t, cc.Invoke(ctx, "/other.Svc/M", nil, nil))
	require.Equal(t, "/other.Svc/M", rc.method)

	Set(ModeCentrifugo)
	require.Same(t, rc, GRPCClientConn(rc))
}
