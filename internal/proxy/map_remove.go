// Modified by the Cedrazeka authors, 2026. See NOTICE.

package proxy

import (
	"context"

	"github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
)

// MapRemoveProxy allows to send MapRemove requests.
type MapRemoveProxy interface {
	ProxyMapRemove(context.Context, *proxyproto.MapRemoveRequest) (*proxyproto.MapRemoveResponse, error)
	// Protocol for metrics and logging.
	Protocol() string
	// UseBase64 for bytes in requests from Centrifugo to application backend.
	UseBase64() bool
	// IncludeMeta ...
	IncludeMeta() bool
}
