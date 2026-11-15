// Modified by the Cedrazeka authors, 2026. See NOTICE.

package telemetry

import (
	"github.com/xantos008/centrifunahuy-na-go/internal/configtypes"

	"go.opentelemetry.io/otel/sdk/resource"
)

// resourceDetectors returns extra resource detectors to attach to the
// telemetry resource based on the OpenTelemetry configuration.
func resourceDetectors(cfg configtypes.OpenTelemetry) ([]resource.Detector, error) {
	return nil, nil
}
