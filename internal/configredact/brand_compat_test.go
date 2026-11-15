package configredact

import "github.com/xantos008/centrifunahuy-na-go/internal/brand"

// The upstream tests in this package assert upstream names (CENTRIFUGO_* env,
// "centrifugo" defaults), so they run in Centrifugo compatibility mode.
// Default-mode behavior is covered in internal/brand/e2e (ADR-0022).
func init() { brand.Set(brand.ModeCentrifugo) }
