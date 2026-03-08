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

func NewEstimateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimate <pipeline.json>",
		Short: "Estimate the cost of executing a pipeline",
		Args:  cobra.ExactArgs(1),
		RunE:  runEstimate,
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

func runEstimate(cmd *cobra.Command, args []string) error {
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

	pipeline, err := readPipelineJSON(args[0])
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Estimating cost... ")

	c := client.New(cfg.BaseURL, cfg.APIKey)
	result, err := c.EstimatePipelineCost(pipeline)
	if err != nil {
		fmt.Fprintln(os.Stderr)
		return err
	}

	fmt.Fprintln(os.Stderr, tui.SymbolCheck)

	if asJSON {
		return output.PrintJSON(result)
	}

	output.PrintEstimateResult(result)
	return nil
}
