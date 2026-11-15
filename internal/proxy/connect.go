// Modified by the Cedrazeka authors, 2026. See NOTICE.

package proxy

import (
	"context"

	"github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
)

// ConnectProxy allows to proxy connect requests to application backend to
// authenticate client connection.
type ConnectProxy interface {
	ProxyConnect(context.Context, *proxyproto.ConnectRequest) (*proxyproto.ConnectResponse, error)
	// Name returns proxy name.
	Name() string
	// Protocol for metrics and logging.
	Protocol() string
	// UseBase64 for bytes in requests from Centrifugo to application backend.
	UseBase64() bool
}
