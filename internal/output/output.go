package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dmt195/inodes-cli/internal/client"
	"github.com/dmt195/inodes-cli/internal/tui"
)

// IsInteractive returns true if stdout is a terminal
func IsInteractive() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// PrintJSON outputs data as formatted JSON to stdout
func PrintJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintPipelineList prints pipelines as a table
func PrintPipelineList(result *client.PipelineListResponse) {
	if len(result.Pipelines) == 0 {
		fmt.Println(tui.Muted.Render("No pipelines found."))
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, tui.Bold.Render("ID")+"\t"+tui.Bold.Render("NAME")+"\t"+tui.Bold.Render("UPDATED"))

	for _, p := range result.Pipelines {
		fav := ""
		if p.IsFavourite {
			fav = " ★"
		}
		locked := ""
		if p.IsLocked {
			locked = " 🔒"
		}

		updated := p.UpdatedAt
		if t, err := time.Parse(time.RFC3339, p.UpdatedAt); err == nil {
			updated = t.Format("2006-01-02 15:04")
		}

		fmt.Fprintf(w, "%s\t%s%s%s\t%s\n", p.ID, p.Name, fav, locked, updated)
	}
	w.Flush()

	fmt.Printf("\n%s\n", tui.Muted.Render(fmt.Sprintf(
		"Showing %d of %d pipelines (page %d/%d)",
		len(result.Pipelines), result.Meta.Count,
		result.Meta.CurrentPage, result.Meta.TotalPages,
	)))
}

// PrintPipelineDescription prints pipeline parameter info
func PrintPipelineDescription(desc *client.PipelineDescription) {
	fmt.Println(tui.Title.Render(desc.Name))
	if desc.Description != "" {
		fmt.Println(tui.Muted.Render(desc.Description))
	}
	fmt.Println()

	if len(desc.ApiNodes) > 0 {
		fmt.Println(tui.Subtitle.Render("── Value Parameters ──"))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, tui.Bold.Render("KEY")+"\t"+tui.Bold.Render("TYPE")+"\t"+tui.Bold.Render("DEFAULT"))
		for _, n := range desc.ApiNodes {
			def := fmt.Sprintf("%v", n.DefaultValue)
			if def == "<nil>" {
				def = ""
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", n.Key, n.DataType, def)
		}
		w.Flush()
		fmt.Println()
	}

	if len(desc.ApiImageNodes) > 0 {
		fmt.Println(tui.Subtitle.Render("── Image Parameters ──"))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, tui.Bold.Render("KEY")+"\t"+tui.Bold.Render("REQUIRED"))
		for _, n := range desc.ApiImageNodes {
			req := "optional"
			if n.Required {
				req = "required"
			}
			fmt.Fprintf(w, "%s\t%s\n", n.Key, req)
		}
		w.Flush()
		fmt.Println()
	}

	if len(desc.ApiNodes) == 0 && len(desc.ApiImageNodes) == 0 {
		fmt.Println(tui.Muted.Render("This pipeline has no API parameters."))
	}

	fmt.Printf("%s %s\n", tui.Muted.Render("Pipeline ID:"), desc.ID)
}

// PrintRunResult prints the result summary of a pipeline execution
func PrintRunResult(report *client.PipelineReport, outputPath string) {
	duration := report.TotalProcessingTime / time.Millisecond
	dims := fmt.Sprintf("%dx%d", report.ImageDetails.Width, report.ImageDetails.Height)
	format := report.ImageDetails.Format
	if format == "" {
		format = "png"
	}

	parts := []string{
		fmt.Sprintf("Saved to %s", tui.Bold.Render(outputPath)),
		fmt.Sprintf("(%s %s", dims, format),
	}

	parts = append(parts, fmt.Sprintf("%dms", duration))
	parts = append(parts, fmt.Sprintf("%d credits)", report.TotalUnitsBillable))

	fmt.Fprintf(os.Stderr, "%s %s\n", tui.SymbolCheck, strings.Join(parts, ", "))
}
