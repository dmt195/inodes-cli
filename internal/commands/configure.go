package commands

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/output"
	"github.com/dmt195/inodes-cli/internal/tui"
	"github.com/spf13/cobra"
)

func NewConfigureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Set API key",
		RunE:  runConfigure,
	}
	cmd.Flags().String("base-url", "", "Override API base URL (for dev/local, e.g. http://localhost:8081)")
	return cmd
}

func runConfigure(cmd *cobra.Command, args []string) error {
	cfg, _ := config.Load("", "")

	apiKey := cfg.APIKey
	baseURLFlag, _ := cmd.Flags().GetString("base-url")

	if output.IsInteractive() {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("API Key").
					Description("Your Image Nodes API key").
					Value(&apiKey).
					EchoMode(huh.EchoModePassword),
			),
		)

		if err := form.Run(); err != nil {
			return err
		}
	} else {
		if apiKey == "" {
			return fmt.Errorf("API key required. Set INODES_API_KEY or run interactively")
		}
	}

	cfg.APIKey = apiKey
	if baseURLFlag != "" {
		cfg.BaseURL = baseURLFlag
	}

	// Test connectivity
	fmt.Fprintf(os.Stderr, "Testing connection to %s... ", cfg.BaseURL)
	c := client.New(cfg.BaseURL, cfg.APIKey)
	if err := c.TestAuth(); err != nil {
		fmt.Fprintln(os.Stderr, tui.SymbolCross)
		return fmt.Errorf("authentication failed: %w", err)
	}
	fmt.Fprintln(os.Stderr, tui.SymbolCheck)

	// Save config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%s Configuration saved\n", tui.SymbolCheck)
	return nil
}
