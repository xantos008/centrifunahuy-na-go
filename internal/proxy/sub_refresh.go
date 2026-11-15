// Modified by the Cedrazeka authors, 2026. See NOTICE.

package proxy

import (
	"context"

	"github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
)

// SubRefreshProxy allows to send sub refresh requests.
type SubRefreshProxy interface {
	ProxySubRefresh(context.Context, *proxyproto.SubRefreshRequest) (*proxyproto.SubRefreshResponse, error)
	// Protocol for metrics and logging.
	Protocol() string
	// UseBase64 for bytes in requests from Centrifugo to application backend.
	UseBase64() bool
	// IncludeMeta ...
	IncludeMeta() bool
}
