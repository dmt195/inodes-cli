package commands

import (
	"fmt"

	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewConfigCmd returns the `config` command group.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect CLI configuration",
	}
	cmd.AddCommand(newConfigShowCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the config file path, base URL and API key, and where each comes from",
		Long: `Show the configuration the CLI will use and where each value comes from.

Values are resolved in this order: --api-key/--base-url flags, then the
INODES_API_KEY/INODES_BASE_URL environment variables, then the config file.
The API key is masked.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(
				cmd.Root().PersistentFlags().Lookup("api-key").Value.String(),
				cmd.Root().PersistentFlags().Lookup("base-url").Value.String(),
			)
			if err != nil {
				return err
			}
			path, err := config.FilePath()
			if err != nil {
				path = fmt.Sprintf("(unavailable: %v)", err)
			}

			asJSON, _ := cmd.Flags().GetBool("json")
			if asJSON {
				return output.PrintJSON(map[string]string{
					"config_file":     path,
					"base_url":        cfg.BaseURL,
					"base_url_source": cfg.BaseURLSource,
					"api_key":         config.MaskKey(cfg.APIKey),
					"api_key_source":  cfg.APIKeySource,
				})
			}
			fmt.Printf("Config file:  %s\n", path)
			fmt.Printf("Base URL:     %s  (from %s)\n", cfg.BaseURL, cfg.BaseURLSource)
			fmt.Printf("API key:      %s  (from %s)\n", config.MaskKey(cfg.APIKey), cfg.APIKeySource)
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}
