// Modified by the Cedrazeka authors, 2026. See NOTICE.

package proxy

import (
	"context"

	"github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
)

// RPCProxy allows to proxy RPC requests to application backend.
type RPCProxy interface {
	ProxyRPC(context.Context, *proxyproto.RPCRequest) (*proxyproto.RPCResponse, error)
	// Protocol for metrics and logging.
	Protocol() string
	// UseBase64 for bytes in requests from Centrifugo to application backend.
	UseBase64() bool
	// IncludeMeta ...
	IncludeMeta() bool
}
