package scenarios

import (
	"testing"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

// CS-ENG-CFG: startup with invalid engine settings (env over the profile
// config): exit status and the fatal log line. Needs no backend.
func TestCS_ENG_CFG(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-ENG-CFG", func(s *h.Session) {
		s.StartFails("CENTRIFUGO_ENGINE_TYPE=foo")
		s.StartFails("CENTRIFUGO_BROKER_ENABLED=true", "CENTRIFUGO_BROKER_TYPE=redisnats") // unreleased feature flag off
		s.StartFails("CENTRIFUGO_ENGINE_TYPE=redis", "CENTRIFUGO_ENGINE_REDIS_SENTINEL_ADDRESS=127.0.0.1:26379")
	})
}
