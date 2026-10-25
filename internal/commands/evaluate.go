package commands

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

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
		Long: `Execute a pipeline defined inline as JSON, without saving it. Use it to try
an edited pipeline before saving it.

Give API parameter values with --param key=value and API images with
--image key=<file|ephemeral-id>; local files are uploaded and deleted
afterwards.

A single output is written to -o (default output.png). A pipeline with
several outputs needs --output-dir; each output is written there as
<name>.png.

Examples:
  inodes evaluate layout.json --image screenshot=shot.png --param caption="Hello"
  inodes evaluate layout.json --output-dir panels/`,
		Args: cobra.ExactArgs(1),
		RunE: runEvaluate,
	}
	cmd.Flags().StringP("output", "o", "output.png", "Output file for a single-output pipeline")
	cmd.Flags().String("output-dir", "", "Directory for every output, written as <name>.png")
	cmd.Flags().StringArray("param", nil, "Value parameter (key=value, repeatable)")
	cmd.Flags().StringArray("image", nil, "Image parameter (key=path or ephemeral ID, repeatable)")
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
	outputDir, _ := cmd.Flags().GetString("output-dir")
	asJSON, _ := cmd.Flags().GetBool("json")
	paramFlags, _ := cmd.Flags().GetStringArray("param")
	imageFlags, _ := cmd.Flags().GetStringArray("image")

	pipeline, err := readPipelineJSON(args[0])
	if err != nil {
		return err
	}

	c := client.New(cfg.BaseURL, cfg.APIKey)

	params := make(map[string]any)
	for _, p := range paramFlags {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			return fmt.Errorf("invalid --param format: %q (expected key=value)", p)
		}
		params[k] = v
	}
	for _, img := range imageFlags {
		k, v, ok := strings.Cut(img, "=")
		if !ok {
			return fmt.Errorf("invalid --image format: %q (expected key=path)", img)
		}
		if idRegex.MatchString(v) {
			params[k] = v
			continue
		}
		fmt.Fprintf(os.Stderr, "Uploading %s... ", filepath.Base(v))
		uploaded, err := c.UploadEphemeral(v)
		if err != nil {
			fmt.Fprintln(os.Stderr, tui.SymbolCross)
			return fmt.Errorf("uploading %s: %w", k, err)
		}
		fmt.Fprintln(os.Stderr, tui.SymbolCheck)
		params[k] = uploaded.ID
		defer func(id string) {
			if err := c.DeleteEphemeral(id); err != nil {
				fmt.Fprintf(os.Stderr, "Could not delete upload %s: %v\n", id, err)
			}
		}(uploaded.ID)
	}
	if len(params) > 0 {
		pipeline["parameters"] = params
	}

	fmt.Fprintf(os.Stderr, "Evaluating pipeline... ")
	result, err := c.EvaluatePipelineJSON(pipeline, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, tui.SymbolCross)
		return err
	}
	fmt.Fprintln(os.Stderr, tui.SymbolCheck)

	if !result.Success {
		return fmt.Errorf("pipeline evaluation failed: %s", result.Error)
	}
	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "Warning [%s] %s: %s\n", w.Code, w.NodeName, w.Message)
	}

	if asJSON {
		return output.PrintJSON(result)
	}

	outputs := result.Outputs
	if len(outputs) == 0 && result.Output != "" {
		// Older servers return only the first output.
		outputs = map[string]string{"output": result.Output}
	}
	if len(outputs) == 0 {
		return fmt.Errorf("no image output in response")
	}

	written, err := writeEvaluateOutputs(outputs, outputPath, outputDir)
	if err != nil {
		return err
	}
	for _, path := range written {
		fmt.Fprintf(os.Stderr, "%s Saved %s\n", tui.SymbolCheck, tui.Bold.Render(path))
	}
	fmt.Fprintf(os.Stderr, "%d credits\n", result.Cost)
	return nil
}

// safeOutputName matches output names that are safe to use as file names.
var safeOutputName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// writeEvaluateOutputs decodes base64 outputs and writes them: to
// outputDir/<name>.png when outputDir is set, otherwise the single output
// to outputPath. It returns the paths written, sorted.
func writeEvaluateOutputs(outputs map[string]string, outputPath, outputDir string) ([]string, error) {
	if outputDir == "" && len(outputs) > 1 {
		names := make([]string, 0, len(outputs))
		for name := range outputs {
			names = append(names, name)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("the pipeline has %d outputs (%s); use --output-dir to write them all", len(outputs), strings.Join(names, ", "))
	}

	paths := make(map[string]string, len(outputs))
	for name := range outputs {
		if outputDir == "" {
			paths[name] = outputPath
			continue
		}
		if !safeOutputName.MatchString(name) {
			return nil, fmt.Errorf("output name %q can't be used as a file name", name)
		}
		paths[name] = filepath.Join(outputDir, name+".png")
	}
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, fmt.Errorf("saving file: %w", err)
		}
	}

	var written []string
	for name, b64 := range outputs {
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("decoding output %q: %w", name, err)
		}
		if err := os.WriteFile(paths[name], data, 0644); err != nil {
			return nil, fmt.Errorf("saving file: %w", err)
		}
		written = append(written, paths[name])
	}
	sort.Strings(written)
	return written, nil
}
