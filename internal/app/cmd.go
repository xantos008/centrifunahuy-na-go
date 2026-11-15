// Modified by the Cedrazeka authors, 2026. See NOTICE.

package app

import (
	"github.com/spf13/cobra"
	"github.com/xantos008/centrifunahuy-na-go/internal/brand"
	"github.com/xantos008/centrifunahuy-na-go/internal/config"
)

// Centrifugo returns the root command running the server with opts.
func Centrifugo(opts Options) *cobra.Command {
	var configFile string
	cmd := &cobra.Command{
		Use:   "",
		Short: brand.Product,
		Long:  brand.Product + " – scalable real-time messaging server in language-agnostic way",
		Run: func(cmd *cobra.Command, args []string) {
			Run(cmd, configFile, opts)
		},
	}
	cmd.Flags().StringVarP(&configFile, "config", "c", "config.json", "path to config file")
	cmd.Flags().Bool("watch-config-file", false, "restart server when config file changes (for development)")
	config.DefineFlags(cmd)
	// Read by brand.Resolve in main before cobra runs; declared here so cobra accepts it.
	cmd.PersistentFlags().String(brand.CompatFlag, "", `compatibility mode: "centrifugo" uses upstream env, metric, key prefix, header and gRPC names (ADR-0022)`)
	return cmd
}
