package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/config"
	"github.com/dmt195/inodes-cli/internal/tui"
	"github.com/spf13/cobra"
)

// NewAssetCmd returns the `asset` command group for the asset library.
func NewAssetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asset",
		Short: "Work with images in your asset library",
	}
	cmd.AddCommand(newAssetGetCmd())
	return cmd
}

func newAssetGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <asset-id>",
		Short: "Download an asset library image",
		Long: `Download the original file of an image in your asset library, for example
to measure a device frame. Downloads are free.

Without -o, the file is saved in the current directory under its original
name (or the asset ID).`,
		Args: cobra.ExactArgs(1),
		RunE: runAssetGet,
	}
	cmd.Flags().StringP("output", "o", "", "File to write")
	return cmd
}

func runAssetGet(cmd *cobra.Command, args []string) error {
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

	data, contentType, filename, err := client.New(cfg.BaseURL, cfg.APIKey).DownloadAsset(args[0])
	if err != nil {
		return err
	}

	outputPath, _ := cmd.Flags().GetString("output")
	if outputPath == "" {
		outputPath = defaultAssetFilename(args[0], filename, contentType)
	}
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("saving file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "%s Saved %s (%d bytes)\n", tui.SymbolCheck, tui.Bold.Render(outputPath), len(data))
	return nil
}

// defaultAssetFilename picks a file name in the current directory: the
// server's filename without any directory part, or the asset ID plus an
// extension for the content type.
func defaultAssetFilename(assetID, serverName, contentType string) string {
	if name := filepath.Base(serverName); serverName != "" && name != "." && name != "/" && name != ".." {
		return name
	}
	ext := ".bin"
	if t, ok := strings.CutPrefix(contentType, "image/"); ok && t != "" {
		ext = "." + strings.TrimSuffix(t, "+xml")
	}
	return assetID + ext
}
