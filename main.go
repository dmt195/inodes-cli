package main

import (
	"fmt"
	"os"

	"github.com/dmt195/inodes-cli/internal/commands"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:           "inodes",
		Short:         "Image Nodes CLI — build and run image processing pipelines",
		Long:          "A command-line tool for the Image Nodes API. Discover pipelines, upload images, execute pipelines, and download results.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	root.PersistentFlags().String("api-key", "", "API key (overrides config and INODES_API_KEY)")
	root.PersistentFlags().String("base-url", "", "API base URL (overrides config and INODES_BASE_URL)")

	// Commands
	root.AddCommand(commands.NewConfigureCmd())
	root.AddCommand(commands.NewListCmd())
	root.AddCommand(commands.NewDescribeCmd())
	root.AddCommand(commands.NewRunCmd())
	root.AddCommand(commands.NewUploadCmd())
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
