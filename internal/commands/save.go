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

func NewSaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save <pipeline.json>",
		Short: "Save a pipeline definition to your account",
		Long:  "Save a pipeline JSON definition to your Image Nodes account. The pipeline is validated before saving. Requires a paid subscription.",
		Args:  cobra.ExactArgs(1),
		RunE:  runSave,
	}
	cmd.Flags().String("name", "", "Pipeline name (required)")
	cmd.Flags().String("description", "", "Pipeline description")
	cmd.Flags().Bool("json", false, "Output as JSON")
	cmd.MarkFlagRequired("name")
	return cmd
}

func runSave(cmd *cobra.Command, args []string) error {
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

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	asJSON, _ := cmd.Flags().GetBool("json")

	pipeline, err := readPipelineJSON(args[0])
	if err != nil {
		return err
	}

	// Extract the inner pipeline object if wrapped
	inner, ok := pipeline["pipeline"].(map[string]any)
	if !ok {
		return fmt.Errorf("pipeline JSON must contain a 'pipeline' object with nodes, values, and connection maps")
	}

	fmt.Fprintf(os.Stderr, "Saving pipeline... ")

	c := client.New(cfg.BaseURL, cfg.APIKey)
	result, err := c.SavePipeline(name, description, inner)
	if err != nil {
		fmt.Fprintln(os.Stderr)
		return err
	}

	fmt.Fprintln(os.Stderr, tui.SymbolCheck)

	if asJSON {
		return output.PrintJSON(result)
	}

	output.PrintSaveResult(result)
	return nil
}
