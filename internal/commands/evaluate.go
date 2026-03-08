package commands

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/output"
	"github.com/dmt195/inodes-cli/internal/tui"
	"github.com/spf13/cobra"
)

func NewEvaluateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evaluate <pipeline.json>",
		Short: "Execute a pipeline defined as a JSON file",
		Args:  cobra.ExactArgs(1),
		RunE:  runEvaluate,
	}
	cmd.Flags().StringP("output", "o", "output.png", "Output file path")
	cmd.Flags().Bool("json", false, "Output full report as JSON")
	return cmd
}

func runEvaluate(cmd *cobra.Command, args []string) error {
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

	outputPath, _ := cmd.Flags().GetString("output")
	asJSON, _ := cmd.Flags().GetBool("json")

	pipeline, err := readPipelineJSON(args[0])
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Evaluating pipeline... ")

	c := client.New(cfg.BaseURL, cfg.APIKey)
	result, err := c.EvaluatePipelineJSON(pipeline, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, tui.SymbolCross)
		return err
	}
	fmt.Fprintln(os.Stderr, tui.SymbolCheck)

	if !result.Success {
		return fmt.Errorf("pipeline evaluation failed: %s", result.Error)
	}

	if asJSON {
		return output.PrintJSON(result)
	}

	if result.Output == "" {
		return fmt.Errorf("no image output in response")
	}

	// Decode base64 image and save
	imageData, err := base64.StdEncoding.DecodeString(result.Output)
	if err != nil {
		return fmt.Errorf("decoding image: %w", err)
	}

	if err := os.WriteFile(outputPath, imageData, 0644); err != nil {
		return fmt.Errorf("saving file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%s Saved to %s (%d credits)\n",
		tui.SymbolCheck,
		tui.Bold.Render(outputPath),
		result.Cost,
	)
	return nil
}
