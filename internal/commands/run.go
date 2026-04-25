package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/output"
	"github.com/dmt195/inodes-cli/internal/tui"
	"github.com/spf13/cobra"
)

// idRegex matches asset IDs (26-char ULIDs) to distinguish them from file paths.
var idRegex = regexp.MustCompile(`^[0-9A-HJ-NP-TV-Za-hj-np-tv-z]{26}$`)

func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <pipeline-id>",
		Short: "Execute a pipeline",
		Long: `Execute a pipeline with parameters.

The pipeline-id is a 26-character ULID (e.g., 01KM2XGX2RPYRQ9F7V2ZP3F5TQ).
Use 'inodes list' to find pipeline IDs.

Interactive mode (default): prompts for missing parameters.
CI/CD mode (--no-prompt): uses flags and defaults only.

Image parameters accept either a local file path or a 26-character asset
ID (ULID) from a previous 'inodes upload'.

Outputs:
  Pipelines may produce one or more named outputs (e.g. "thumbnail",
  "banner"). Use 'inodes describe <id>' to list them.

  Single-output pipelines: -o file.png writes the one image. If -o is
  omitted, the file is named after the output (e.g. output.png).

  Multi-output pipelines, choose one of:
    --output-dir ./dist                     write each as <name>.<format>
    --output thumbnail=t.jpg --output banner=b.png   per-output paths
    --url-only                              print "name=url" lines
    --json                                  raw server response`,
		Args:              cobra.ExactArgs(1),
		RunE:              runPipeline,
		ValidArgsFunction: completePipelineIDs,
	}
	cmd.Flags().StringArray("param", nil, "Value parameter (key=value, repeatable)")
	cmd.Flags().StringArray("image", nil, "Image parameter (key=path, repeatable)")
	cmd.Flags().StringArrayP("output", "o", nil, "Output path: single file (single-output pipelines) or name=path (multi-output, repeatable)")
	cmd.Flags().String("output-dir", "", "Write each output to <dir>/<name>.<format> (multi-output pipelines)")
	cmd.Flags().Bool("url-only", false, "Print image URL(s) instead of downloading. Multi-output prints 'name=url' lines.")
	cmd.Flags().Bool("json", false, "Output full report as JSON")
	cmd.Flags().Bool("no-prompt", false, "Disable interactive prompts")
	return cmd
}

func runPipeline(cmd *cobra.Command, args []string) error {
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

	pipelineID := args[0]
	paramFlags, _ := cmd.Flags().GetStringArray("param")
	imageFlags, _ := cmd.Flags().GetStringArray("image")
	outputFlags, _ := cmd.Flags().GetStringArray("output")
	outputDir, _ := cmd.Flags().GetString("output-dir")
	urlOnly, _ := cmd.Flags().GetBool("url-only")
	asJSON, _ := cmd.Flags().GetBool("json")
	noPrompt, _ := cmd.Flags().GetBool("no-prompt")

	c := client.New(cfg.BaseURL, cfg.APIKey)

	// 1. Describe pipeline to get parameter metadata
	fmt.Fprintf(os.Stderr, "Fetching pipeline info... ")
	desc, err := c.DescribePipeline(pipelineID)
	if err != nil {
		fmt.Fprintln(os.Stderr, tui.SymbolCross)
		return err
	}
	fmt.Fprintln(os.Stderr, tui.SymbolCheck)

	// 2. Parse --param flags into params map
	params := make(map[string]any)
	for _, p := range paramFlags {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			return fmt.Errorf("invalid --param format: %q (expected key=value)", p)
		}
		params[k] = v
	}

	// 3. Parse --image flags: upload files, resolve to asset IDs
	imageParams := make(map[string]string)
	for _, img := range imageFlags {
		k, v, ok := strings.Cut(img, "=")
		if !ok {
			return fmt.Errorf("invalid --image format: %q (expected key=path)", img)
		}
		imageParams[k] = v
	}

	// Upload image files that are local paths
	for key, val := range imageParams {
		if idRegex.MatchString(val) {
			// Already an asset ID
			params[key] = val
			continue
		}
		// It's a file path — upload it
		fmt.Fprintf(os.Stderr, "Uploading %s... ", filepath.Base(val))
		result, err := c.UploadEphemeral(val)
		if err != nil {
			fmt.Fprintln(os.Stderr, tui.SymbolCross)
			return fmt.Errorf("uploading %s: %w", key, err)
		}
		fmt.Fprintln(os.Stderr, tui.SymbolCheck)
		params[key] = result.ID
	}

	// 4. Fill in missing value params from defaults
	for _, apiNode := range desc.ApiNodes {
		if _, exists := params[apiNode.Key]; !exists && apiNode.DefaultValue != nil {
			params[apiNode.Key] = apiNode.DefaultValue
		}
	}

	// 5. Interactive prompt for missing params
	interactive := output.IsInteractive() && !noPrompt
	if interactive {
		if err := promptForMissingParams(desc, params, c); err != nil {
			return err
		}
	}

	// Validate all required params are present
	for _, apiNode := range desc.ApiNodes {
		if _, exists := params[apiNode.Key]; !exists {
			return fmt.Errorf("missing required parameter: %s (use --param %s=value)", apiNode.Key, apiNode.Key)
		}
	}

	// 6. Execute pipeline
	fmt.Fprintf(os.Stderr, "Executing pipeline... ")
	report, err := c.EvaluatePipeline(pipelineID, params, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, tui.SymbolCross)
		return err
	}
	fmt.Fprintln(os.Stderr, tui.SymbolCheck)

	// 7. Output results
	if asJSON {
		return output.PrintJSON(report)
	}

	if len(report.Outputs) == 0 {
		return fmt.Errorf("no outputs in response — pipeline may not have produced any")
	}

	if urlOnly {
		return printOutputURLs(c, report)
	}

	plan, err := planOutputWrites(report, outputFlags, outputDir)
	if err != nil {
		return err
	}

	writes, err := downloadAndWriteOutputs(c, report, plan)
	if err != nil {
		return err
	}

	output.PrintRunResult(report, writes)
	return nil
}

// outputNamesSorted returns the output names from a report in alphabetical order.
func outputNamesSorted(report *client.PipelineReport) []string {
	names := make([]string, 0, len(report.Outputs))
	for name := range report.Outputs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// printOutputURLs prints output URLs to stdout. Single-output pipelines print
// just the URL (back-compat); multi-output prints "name=url" lines.
func printOutputURLs(c *client.Client, report *client.PipelineReport) error {
	names := outputNamesSorted(report)
	if len(names) == 1 {
		out := report.Outputs[names[0]]
		if out.ImageUrl == "" {
			return fmt.Errorf("no image URL for output %q", names[0])
		}
		fmt.Println(c.ResolveURL(out.ImageUrl))
		return nil
	}
	for _, name := range names {
		out := report.Outputs[name]
		if out.ImageUrl == "" {
			fmt.Fprintf(os.Stderr, "%s no image URL for output %q\n", tui.SymbolCross, name)
			continue
		}
		fmt.Printf("%s=%s\n", name, c.ResolveURL(out.ImageUrl))
	}
	return nil
}

// planOutputWrites resolves --output / --output-dir flags into a map of
// output-name → file-path. Returns an error if the flags are inconsistent
// with the pipeline's output count or names.
func planOutputWrites(report *client.PipelineReport, outputFlags []string, outputDir string) (map[string]string, error) {
	names := outputNamesSorted(report)

	// --output-dir takes precedence: write every output as <dir>/<name>.<format>.
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return nil, fmt.Errorf("creating output dir: %w", err)
		}
		plan := make(map[string]string, len(names))
		for _, name := range names {
			out := report.Outputs[name]
			ext := string(out.Format)
			if ext == "" {
				ext = "png"
			}
			plan[name] = filepath.Join(outputDir, name+"."+ext)
		}
		return plan, nil
	}

	// Parse --output flags. Each entry is either a single path (single-output
	// pipelines) or a "name=path" mapping (multi-output).
	hasMapping := false
	mappings := make(map[string]string)
	var bareSinglePath string
	for _, f := range outputFlags {
		if name, path, ok := strings.Cut(f, "="); ok && name != "" && path != "" && !strings.ContainsAny(name, `/\`) {
			hasMapping = true
			mappings[name] = path
			continue
		}
		bareSinglePath = f
	}

	switch {
	case hasMapping && bareSinglePath != "":
		return nil, fmt.Errorf("--output values must all be 'name=path' mappings or a single path, not both")
	case hasMapping:
		for name := range mappings {
			if _, ok := report.Outputs[name]; !ok {
				return nil, fmt.Errorf("--output %s: pipeline has no output named %q (available: %s)",
					name, name, strings.Join(names, ", "))
			}
		}
		// Warn for outputs we will skip.
		for _, name := range names {
			if _, ok := mappings[name]; !ok {
				fmt.Fprintf(os.Stderr, "%s skipping output %q (no --output mapping)\n", tui.SymbolArrow, name)
			}
		}
		return mappings, nil
	case bareSinglePath != "":
		if len(names) > 1 {
			return nil, fmt.Errorf(
				"pipeline produces %d outputs: %s. Use --output-dir <dir> or --output <name>=<path> (repeatable)",
				len(names), strings.Join(names, ", "),
			)
		}
		return map[string]string{names[0]: bareSinglePath}, nil
	default:
		// No flags: default-name file per output.
		if len(names) > 1 {
			return nil, fmt.Errorf(
				"pipeline produces %d outputs: %s. Use --output-dir <dir> or --output <name>=<path> (repeatable)",
				len(names), strings.Join(names, ", "),
			)
		}
		out := report.Outputs[names[0]]
		ext := string(out.Format)
		if ext == "" {
			ext = "png"
		}
		return map[string]string{names[0]: names[0] + "." + ext}, nil
	}
}

// downloadAndWriteOutputs downloads each output's image_url and writes it to
// the path resolved by planOutputWrites. Returns the list of written entries
// in alphabetical order for stable summary output.
func downloadAndWriteOutputs(c *client.Client, report *client.PipelineReport, plan map[string]string) ([]output.WrittenOutput, error) {
	names := make([]string, 0, len(plan))
	for n := range plan {
		names = append(names, n)
	}
	sort.Strings(names)

	writes := make([]output.WrittenOutput, 0, len(names))
	for _, name := range names {
		path := plan[name]
		out := report.Outputs[name]
		if out.ImageUrl == "" {
			return writes, fmt.Errorf("output %q has no image URL", name)
		}
		fmt.Fprintf(os.Stderr, "Downloading %s... ", name)
		data, _, err := c.DownloadFile(out.ImageUrl)
		if err != nil {
			fmt.Fprintln(os.Stderr, tui.SymbolCross)
			return writes, fmt.Errorf("downloading %s: %w", name, err)
		}
		fmt.Fprintln(os.Stderr, tui.SymbolCheck)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return writes, fmt.Errorf("saving %s: %w", name, err)
		}
		writes = append(writes, output.WrittenOutput{Name: name, Path: path})
	}
	return writes, nil
}

func promptForMissingParams(desc *client.PipelineDescription, params map[string]any, c *client.Client) error {
	var fields []huh.Field

	// Value parameters
	for _, apiNode := range desc.ApiNodes {
		if _, exists := params[apiNode.Key]; exists {
			continue
		}
		key := apiNode.Key
		defaultVal := ""
		if apiNode.DefaultValue != nil {
			defaultVal = fmt.Sprintf("%v", apiNode.DefaultValue)
		}

		val := defaultVal
		field := huh.NewInput().
			Title(key).
			Description(fmt.Sprintf("Type: %s", apiNode.DataType)).
			Value(&val)

		// Capture for closure
		fields = append(fields, field)
		// We need to set the value after the form runs, so use a deferred approach
		defer func(k string, v *string) {
			if *v != "" {
				params[k] = *v
			}
		}(key, &val)
	}

	// Image parameters (only missing ones)
	for _, apiImage := range desc.ApiImageNodes {
		if _, exists := params[apiImage.Key]; exists {
			continue
		}
		key := apiImage.Key
		req := "optional"
		if apiImage.Required {
			req = "required"
		}

		val := ""
		field := huh.NewInput().
			Title(fmt.Sprintf("%s (image, %s)", key, req)).
			Description("Enter file path or asset ID").
			Value(&val)

		fields = append(fields, field)
		defer func(k string, v *string, cl *client.Client) {
			if *v == "" {
				return
			}
			// Upload if it's a file path
			if !idRegex.MatchString(*v) {
				fmt.Fprintf(os.Stderr, "Uploading %s... ", filepath.Base(*v))
				result, err := cl.UploadEphemeral(*v)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s upload failed: %v\n", tui.SymbolCross, err)
					return
				}
				fmt.Fprintln(os.Stderr, tui.SymbolCheck)
				params[k] = result.ID
			} else {
				params[k] = *v
			}
		}(key, &val, c)
	}

	if len(fields) == 0 {
		return nil
	}

	form := huh.NewForm(huh.NewGroup(fields...))
	return form.Run()
}
