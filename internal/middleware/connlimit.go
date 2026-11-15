// Modified by the Cedrazeka authors, 2026. See NOTICE.

package middleware

import (
	"net/http"
	"time"

	"github.com/xantos008/centrifunahuy-na-go/internal/config"
	"github.com/xantos008/centrifunahuy-na-go/internal/metrics"
	"github.com/xantos008/centrifunahuy-na-go/internal/tools"

	"github.com/centrifugal/centrifuge"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

var connLimitReachedLogLimiter = tools.NewIntervalLimiter(3 * time.Second)

type ConnLimit struct {
	node         *centrifuge.Node
	cfgContainer *config.Container
	rl           *rate.Limiter
}

func NewConnLimit(node *centrifuge.Node, cfgContainer *config.Container) *ConnLimit {
	rl := connectionRateLimiter(cfgContainer.Config().Client.ConnectionRateLimit)
	return &ConnLimit{node: node, cfgContainer: cfgContainer, rl: rl}
}

func (l *ConnLimit) Middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l.rl != nil && !l.rl.Allow() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		connLimit := l.cfgContainer.ConfigRef().Client.ConnectionLimit
		if connLimit > 0 && l.node.Hub().NumClients() >= connLimit {
			metrics.ConnLimitReached.Inc()
			if connLimitReachedLogLimiter.Allow() {
				log.Warn().Int("limit", connLimit).Msg("node connection limit reached")
			}
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func connectionRateLimiter(connRateLimit int) *rate.Limiter {
	if connRateLimit > 0 {
		return rate.NewLimiter(rate.Limit(connRateLimit), connRateLimit)
	}
	return nil
}
