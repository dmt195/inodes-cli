package main

import (
	"fmt"
	"os"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/commands"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:           "inodes",
		Short:         "Image Nodes CLI — build and run image processing pipelines",
		Long:          "A command-line tool for the Image Nodes API. Discover pipelines, upload images, execute pipelines, and download results.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	root.PersistentFlags().String("api-key", "", "API key (overrides config and INODES_API_KEY)")
	root.PersistentFlags().String("base-url", "", "API base URL (overrides config and INODES_BASE_URL)")
	root.PersistentFlags().Int("retries", client.DefaultRetries, "Times to retry rate limits, busy responses and network errors (0 to disable)")
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		retries, err := cmd.Flags().GetInt("retries")
		if err != nil {
			return err
		}
		if retries < 0 {
			return fmt.Errorf("--retries must be 0 or more")
		}
		client.DefaultRetries = retries
		return nil
	}

	// Commands
	root.AddCommand(commands.NewConfigureCmd())
	root.AddCommand(commands.NewConfigCmd())
	root.AddCommand(commands.NewListCmd())
	root.AddCommand(commands.NewDescribeCmd())
	root.AddCommand(commands.NewRunCmd())
	root.AddCommand(commands.NewUploadCmd())
	root.AddCommand(commands.NewUploadDeleteCmd())
	root.AddCommand(commands.NewSchemaCmd())
	root.AddCommand(commands.NewFontsCmd())
	root.AddCommand(commands.NewValidateCmd())
	root.AddCommand(commands.NewEstimateCmd())
	root.AddCommand(commands.NewEvaluateCmd())
	root.AddCommand(commands.NewSaveCmd())
	root.AddCommand(commands.NewDiffCmd())
	root.AddCommand(commands.NewExportCmd())
	root.AddCommand(commands.NewDeleteCmd())
	root.AddCommand(commands.NewSetCmd())
	root.AddCommand(commands.NewVersionsCmd())
	root.AddCommand(commands.NewRestoreCmd())
	root.AddCommand(commands.NewAssetCmd())
	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("inodes", version)
		},
	})

	if err := root.Execute(); err != nil {
		// Map errors to exit codes
		code := 1
		msg := err.Error()
		switch {
		case contains(msg, "401", "authentication failed", "API key"):
			code = 2
		case contains(msg, "API error"):
			code = 3
		case contains(msg, "connection refused", "no such host", "timeout"):
			code = 4
		case contains(msg, "file not found", "saving file", "permission denied"):
			code = 5
		}

		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(code)
	}
}

func contains(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
