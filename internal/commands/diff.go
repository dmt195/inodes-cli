package commands

import (
	"fmt"
	"os"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/output"
	"github.com/dmt195/inodes-cli/internal/tui"
	"github.com/spf13/cobra"
)

func NewDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "diff <pipeline-id>",
		Short:             "Run diff assessment between API and editor mode",
		Hidden:            true,
		Args:              cobra.ExactArgs(1),
		RunE:              runDiff,
		ValidArgsFunction: completePipelineIDs,
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

func runDiff(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(
		cmd.Root().PersistentFlags().Lookup("api-key").Value.String(),
		cmd.Root().PersistentFlags().Lookup("base-url").Value.String(),
	)
	if err != nil {
		return err
	}
	if err := cfg.RequireAPIKey(); err != nil {
		return err
	}

	asJSON, _ := cmd.Flags().GetBool("json")

	fmt.Fprintf(os.Stderr, "Running diff assessment... ")

	c := client.New(cfg.BaseURL, cfg.APIKey)
	result, err := c.DiffAssessment(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr)
		return err
	}

	fmt.Fprintln(os.Stderr, tui.SymbolCheck)

	if asJSON {
		return output.PrintJSON(result)
	}

	output.PrintDiffResult(result)
	return nil
}
