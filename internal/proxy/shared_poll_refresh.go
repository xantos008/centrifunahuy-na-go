// Modified by the Cedrazeka authors, 2026. See NOTICE.

package proxy

import (
	"context"

	"github.com/xantos008/centrifunahuy-na-go/internal/proxyproto"
)

// SharedPollRefreshProxy allows to send SharedPollRefresh requests.
type SharedPollRefreshProxy interface {
	ProxySharedPollRefresh(context.Context, *proxyproto.SharedPollRefreshRequest) (*proxyproto.SharedPollRefreshResponse, error)
	// Protocol for metrics and logging.
	Protocol() string
}
