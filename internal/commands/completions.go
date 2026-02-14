package commands

import (
	"fmt"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/spf13/cobra"
)

// completePipelineIDs returns a ValidArgsFunction that fetches pipeline IDs
// from the API and offers them as tab completions with pipeline names as descriptions.
func completePipelineIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cfg, err := config.Load(
		cmd.Root().PersistentFlags().Lookup("api-key").Value.String(),
		cmd.Root().PersistentFlags().Lookup("base-url").Value.String(),
	)
	if err != nil || cfg.APIKey == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	c := client.New(cfg.BaseURL, cfg.APIKey)
	result, err := c.ListPipelines(0, 50)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, p := range result.Pipelines {
		// Format: "id\tdescription" — Cobra shows the description as a hint
		completions = append(completions, fmt.Sprintf("%s\t%s", p.ID, p.Name))
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeImageFiles returns file completions filtered to image extensions.
func completeImageFiles(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{"png", "jpg", "jpeg", "webp", "gif", "bmp", "tiff"}, cobra.ShellCompDirectiveFilterFileExt
}
