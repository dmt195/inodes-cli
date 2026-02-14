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
	return &cobra.Command{
		Use:   "configure",
		Short: "Set API key and base URL",
		RunE:  runConfigure,
	}
}

func runConfigure(cmd *cobra.Command, args []string) error {
	cfg, _ := config.Load("", "")

	apiKey := cfg.APIKey
	baseURL := cfg.BaseURL

	if output.IsInteractive() {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("API Key").
					Description("Your Image Nodes API key").
					Value(&apiKey).
					EchoMode(huh.EchoModePassword),
				huh.NewInput().
					Title("Base URL").
					Description("API server URL").
					Value(&baseURL),
			),
		)

		if err := form.Run(); err != nil {
			return err
		}
	} else {
		// Non-interactive: require env vars or flags
		if apiKey == "" {
			return fmt.Errorf("API key required. Set INODES_API_KEY or run interactively")
		}
	}

	cfg.APIKey = apiKey
	cfg.BaseURL = baseURL

	// Test connectivity
	fmt.Fprintf(os.Stderr, "Testing connection... ")
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
