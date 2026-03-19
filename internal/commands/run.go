package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
ID (ULID) from a previous 'inodes upload'.`,
		Args:              cobra.ExactArgs(1),
		RunE:              runPipeline,
		ValidArgsFunction: completePipelineIDs,
	}
	cmd.Flags().StringArray("param", nil, "Value parameter (key=value, repeatable)")
	cmd.Flags().StringArray("image", nil, "Image parameter (key=path, repeatable)")
	cmd.Flags().StringP("output", "o", "output.png", "Output file path")
	cmd.Flags().Bool("url-only", false, "Print image URL instead of downloading")
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
	outputPath, _ := cmd.Flags().GetString("output")
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

	if urlOnly {
		if report.ImageDetails.ImageUrl != "" {
			fmt.Println(c.ResolveURL(report.ImageDetails.ImageUrl))
		} else {
			return fmt.Errorf("no image URL in response")
		}
		return nil
	}

	// Download the result image
	if report.ImageDetails.ImageUrl == "" {
		return fmt.Errorf("no image URL in response — pipeline may not have produced an output")
	}

	fmt.Fprintf(os.Stderr, "Downloading result... ")
	imageData, _, err := c.DownloadFile(report.ImageDetails.ImageUrl)
	if err != nil {
		fmt.Fprintln(os.Stderr, tui.SymbolCross)
		return fmt.Errorf("downloading result: %w", err)
	}
	fmt.Fprintln(os.Stderr, tui.SymbolCheck)

	if err := os.WriteFile(outputPath, imageData, 0644); err != nil {
		return fmt.Errorf("saving file: %w", err)
	}

	output.PrintRunResult(report, outputPath)
	return nil
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
