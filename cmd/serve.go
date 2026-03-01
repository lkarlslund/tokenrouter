package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lkarlslund/tokenrouter/pkg/config"
	"github.com/lkarlslund/tokenrouter/pkg/proxy"
	"github.com/lkarlslund/tokenrouter/pkg/wizard"
	"github.com/spf13/cobra"
)

var (
	serveConfigPath                 string
	serveListenAddrOverride         string
	serveAllowLocalhostNoAuth       bool
	serveAutoEnablePublicFreeModels bool
)

func init() {
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the proxy server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadServerConfig(serveConfigPath)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("load server config: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "No server config found at %s. Running first-time setup wizard.\n", serveConfigPath)
				cfg = config.NewDefaultServerConfig()
				if err := wizard.RunServerWizard(serveConfigPath, cfg); err != nil {
					return fmt.Errorf("first-time setup failed: %w", err)
				}
				cfg, err = config.LoadServerConfig(serveConfigPath)
				if err != nil {
					return fmt.Errorf("load server config after setup: %w", err)
				}
			}
			if cmd.Flags().Changed("listen-addr") {
				cfg.ListenAddr = serveListenAddrOverride
			}
			if cmd.Flags().Changed("allow-localhost-no-auth") {
				cfg.AllowLocalhostNoAuth = serveAllowLocalhostNoAuth
			}
			if cmd.Flags().Changed("auto-enable-public-free-models") {
				cfg.AutoEnablePublicFreeModels = serveAutoEnablePublicFreeModels
			}

			srv, err := proxy.NewServer(serveConfigPath, cfg)
			if err != nil {
				return fmt.Errorf("create server: %w", err)
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			return srv.Run(ctx)
		},
	}
	serveCmd.Flags().StringVar(&serveConfigPath, "config", config.DefaultServerConfigPath(), "Server config TOML path")
	serveCmd.Flags().StringVar(&serveListenAddrOverride, "listen-addr", "", "Override listen address from config (e.g. 127.0.0.1:7050)")
	serveCmd.Flags().BoolVar(&serveAllowLocalhostNoAuth, "allow-localhost-no-auth", false, "Override allow_localhost_no_auth in config")
	serveCmd.Flags().BoolVar(&serveAutoEnablePublicFreeModels, "auto-enable-public-free-models", false, "Override auto_enable_public_free_models in config")
	rootCmd.AddCommand(serveCmd)
}
