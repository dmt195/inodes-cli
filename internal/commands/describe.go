package commands

import (
	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <pipeline-id>",
		Short: "Show pipeline parameters",
		Long: `Show the API parameters (images and values) for a pipeline.

The pipeline-id is a 26-character ULID (e.g., 01KM2XGX2RPYRQ9F7V2ZP3F5TQ).
Use 'inodes list' to find pipeline IDs.`,
		Args:              cobra.ExactArgs(1),
		RunE:              runDescribe,
		ValidArgsFunction: completePipelineIDs,
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

func runDescribe(cmd *cobra.Command, args []string) error {
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

	c := client.New(cfg.BaseURL, cfg.APIKey)
	desc, err := c.DescribePipeline(args[0])
	if err != nil {
		return err
	}

	if asJSON {
		return output.PrintJSON(desc)
	}

	output.PrintPipelineDescription(desc)
	return nil
}
