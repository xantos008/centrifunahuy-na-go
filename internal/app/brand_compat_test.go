package app

import "github.com/xantos008/centrifunahuy-na-go/internal/brand"

// The upstream tests in this package configure the server through CENTRIFUGO_*
// env vars, so they run in Centrifugo compatibility mode.
// Default-mode behavior is covered in internal/brand/e2e (ADR-0022).
func init() { brand.Set(brand.ModeCentrifugo) }
