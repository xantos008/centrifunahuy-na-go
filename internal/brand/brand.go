// Package brand holds Cedrazeka's user-visible names and the optional
// Centrifugo compatibility mode (ADR-0022).
//
// The mode is process-wide. It is resolved once in main, before any
// configuration is read, and is read-only afterwards. A process-wide value is
// used instead of passing names through every upstream call site, which would
// make each upstream upgrade a large merge.
package brand

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"google.golang.org/grpc"
)

// Product is the product name. It is the same in every mode.
const Product = "Cedrazeka"

// Binary is the executable name. It is the same in every mode.
const Binary = "cedrazeka"

// CompatFlag and CompatEnv select the mode. Their value is CompatCentrifugo or empty.
const (
	CompatFlag       = "compat"
	CompatEnv        = "CEDRAZEKA_COMPAT"
	CompatCentrifugo = "centrifugo"
)

// Mode selects which names the process uses for operator- and wire-facing identifiers.
type Mode uint8

const (
	// ModeDefault uses Cedrazeka names.
	ModeDefault Mode = iota
	// ModeCentrifugo uses upstream Centrifugo names for env, metrics, key
	// prefixes, headers and gRPC services, to interoperate with existing
	// deployments, clients and backends.
	ModeCentrifugo
)

func (m Mode) String() string {
	if m == ModeCentrifugo {
		return CompatCentrifugo
	}
	return "default"
}

// upstreamIdent is the identifier upstream uses in defaults, env and metrics.
const upstreamIdent = "centrifugo"

// Upstream gRPC service full names and our replacements (default mode).
const (
	UpstreamAPIService       = "centrifugal.centrifugo.api.CentrifugoApi"
	UpstreamProxyService     = "centrifugal.centrifugo.proxy.CentrifugoProxy"
	UpstreamUniStreamService = "centrifugal.centrifugo.unistream.CentrifugoUniStream"

	APIService       = "cedrazeka.api.v1.Api"
	ProxyService     = "cedrazeka.proxy.v1.Proxy"
	UniStreamService = "cedrazeka.unistream.v1.UniStream"
)

var ourService = map[string]string{
	UpstreamAPIService:       APIService,
	UpstreamProxyService:     ProxyService,
	UpstreamUniStreamService: UniStreamService,
}

var current atomic.Uint32 // holds a Mode

// Current returns the process mode.
func Current() Mode { return Mode(current.Load()) }

// Set sets the process mode. Call it once, before configuration is loaded.
func Set(m Mode) { current.Store(uint32(m)) }

// ParseMode parses a --compat / CEDRAZEKA_COMPAT value.
func ParseMode(v string) (Mode, error) {
	switch v {
	case "":
		return ModeDefault, nil
	case CompatCentrifugo:
		return ModeCentrifugo, nil
	}
	return ModeDefault, fmt.Errorf("unknown %s value %q: want %q or empty", CompatFlag, v, CompatCentrifugo)
}

// Resolve finds the mode from command-line args (without the program name)
// and the environment. The --compat flag wins over CEDRAZEKA_COMPAT.
func Resolve(args []string, getenv func(string) string) (Mode, error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			break
		}
		for _, p := range []string{"--" + CompatFlag, "-" + CompatFlag} {
			if a == p {
				if i+1 >= len(args) {
					return ModeDefault, fmt.Errorf("flag --%s needs a value", CompatFlag)
				}
				return ParseMode(args[i+1])
			}
			if v, ok := strings.CutPrefix(a, p+"="); ok {
				return ParseMode(v)
			}
		}
	}
	return ParseMode(getenv(CompatEnv))
}

// Ident is the lower-case identifier used for metric namespaces, key
// prefixes, client IDs, telemetry service names and span attribute prefixes.
func Ident() string {
	if Current() == ModeCentrifugo {
		return upstreamIdent
	}
	return Binary
}

// EnvPrefix is the environment variable prefix without the trailing underscore.
func EnvPrefix() string { return strings.ToUpper(Ident()) }

// EnvVar returns the full env var name for an upper-case suffix, e.g. "VAR_X".
func EnvVar(suffix string) string { return EnvPrefix() + "_" + suffix }

// MapDefault converts an upstream default value that carries the upstream
// identifier ("centrifugo", "centrifugo-method", …) to the current mode. Other
// values are returned unchanged.
func MapDefault(def string) string {
	if Current() == ModeCentrifugo {
		return def
	}
	if def == upstreamIdent {
		return Binary
	}
	if rest, ok := strings.CutPrefix(def, upstreamIdent+"-"); ok {
		return Binary + "-" + rest
	}
	return def
}

// Header returns an HTTP header name with the current identifier, e.g.
// Header("Error-Mode") is "X-Cedrazeka-Error-Mode".
func Header(name string) string {
	id := Ident()
	return "X-" + strings.ToUpper(id[:1]) + id[1:] + "-" + name
}

// GRPCServiceDescs returns the service descriptors to register for an
// upstream-generated desc: ours in every mode, plus the original one in
// compatibility mode.
func GRPCServiceDescs(upstream *grpc.ServiceDesc) []*grpc.ServiceDesc {
	name, ok := ourService[upstream.ServiceName]
	if !ok {
		return []*grpc.ServiceDesc{upstream}
	}
	ours := *upstream
	ours.ServiceName = name
	if Current() == ModeCentrifugo {
		return []*grpc.ServiceDesc{&ours, upstream}
	}
	return []*grpc.ServiceDesc{&ours}
}

// RegisterGRPCService registers impl under the names GRPCServiceDescs returns.
func RegisterGRPCService(s grpc.ServiceRegistrar, upstream *grpc.ServiceDesc, impl any) {
	for _, d := range GRPCServiceDescs(upstream) {
		s.RegisterService(d, impl)
	}
}

// GRPCClientConn wraps a connection used by an upstream-generated client so
// that calls go to our service name in default mode. In compatibility mode it
// returns cc unchanged.
func GRPCClientConn(cc grpc.ClientConnInterface) grpc.ClientConnInterface {
	if Current() == ModeCentrifugo {
		return cc
	}
	return renamingConn{cc}
}

type renamingConn struct{ grpc.ClientConnInterface }

func (c renamingConn) Invoke(ctx context.Context, method string, args, reply any, opts ...grpc.CallOption) error {
	return c.ClientConnInterface.Invoke(ctx, renameMethod(method), args, reply, opts...)
}

func (c renamingConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return c.ClientConnInterface.NewStream(ctx, desc, renameMethod(method), opts...)
}

// renameMethod maps "/<upstream service>/<Method>" to "/<our service>/<Method>".
func renameMethod(method string) string {
	svc, m, ok := strings.Cut(strings.TrimPrefix(method, "/"), "/")
	if !ok {
		return method
	}
	if ours, ok := ourService[svc]; ok {
		return "/" + ours + "/" + m
	}
	return method
}
