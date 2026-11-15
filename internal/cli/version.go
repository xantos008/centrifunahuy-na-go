// Modified by the Cedrazeka authors, 2026. See NOTICE.

package cli

import (
	"fmt"
	"runtime"

	"github.com/xantos008/centrifunahuy-na-go/internal/brand"
	"github.com/xantos008/centrifunahuy-na-go/internal/build"

	"github.com/spf13/cobra"
)

func Version() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: brand.Product + " version information",
		Long:  "Print the version information of " + brand.Product,
		Run: func(cmd *cobra.Command, args []string) {
			version()
		},
	}
}

func version() {
	fmt.Printf("%s v%s (Go version: %s)\n", brand.Product, build.Version, runtime.Version())
}
