// Modified by the Cedrazeka authors, 2026. See NOTICE.

package app

import (
	"net"
	"strconv"
	"strings"

	"github.com/xantos008/centrifunahuy-na-go/internal/config"
	"github.com/xantos008/centrifunahuy-na-go/internal/metrics/graphite"

	"github.com/centrifugal/centrifuge"
	"github.com/prometheus/client_golang/prometheus"
)

func graphiteExporter(cfg config.Config, nodeCfg centrifuge.Config, gatherer prometheus.Gatherer) *graphite.Exporter {
	return graphite.New(graphite.Config{
		Address:  net.JoinHostPort(cfg.Graphite.Host, strconv.Itoa(cfg.Graphite.Port)),
		Gatherer: gatherer,
		Prefix:   strings.TrimSuffix(cfg.Graphite.Prefix, ".") + "." + graphite.PreparePathComponent(nodeCfg.Name),
		Interval: cfg.Graphite.Interval.ToDuration(),
		Tags:     cfg.Graphite.Tags,
	})
}
