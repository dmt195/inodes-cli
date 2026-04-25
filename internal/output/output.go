package output

import (
	"encoding/json"
	"fmt"
	"os"
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

	if len(desc.Outputs) > 0 {
		fmt.Println(tui.Subtitle.Render("── Outputs ──"))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, tui.Bold.Render("KEY")+"\t"+tui.Bold.Render("FORMAT")+"\t"+tui.Bold.Render("QUALITY"))
		for _, o := range desc.Outputs {
			quality := "-"
			if o.Quality > 0 {
				quality = fmt.Sprintf("%d", o.Quality)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", o.Key, o.Format, quality)
		}
		w.Flush()
		fmt.Println()
	}

	if len(desc.ApiNodes) == 0 && len(desc.ApiImageNodes) == 0 {
		fmt.Println(tui.Muted.Render("This pipeline has no API parameters."))
	}

	fmt.Printf("%s %s\n", tui.Muted.Render("Pipeline ID:"), desc.ID)
}

// PrintDiffResult prints the result of a diff assessment
func PrintDiffResult(r *client.DiffAssessmentResult) {
	fmt.Println(tui.Subtitle.Render("── Diff Assessment ──"))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\n", tui.Bold.Render("Avg Diff"), fmt.Sprintf("%.4f", r.AvgDiff))
	fmt.Fprintf(w, "%s\t%s\n", tui.Bold.Render("Max Diff"), fmt.Sprintf("%d", r.MaxDiff))
	fmt.Fprintf(w, "%s\t%s\n", tui.Bold.Render("API Resolution"), fmt.Sprintf("%dx%d", r.ApiWidth, r.ApiHeight))
	fmt.Fprintf(w, "%s\t%s\n", tui.Bold.Render("Editor Resolution"), fmt.Sprintf("%dx%d", r.EditorWidth, r.EditorHeight))
	fmt.Fprintf(w, "%s\t%s\n", tui.Bold.Render("Scale Factor"), fmt.Sprintf("%.4f", r.ScaleFactor))
	fmt.Fprintf(w, "%s\t%s\n", tui.Bold.Render("Pixels Compared"), fmt.Sprintf("%d", r.PixelsCompared))
	w.Flush()
}

// PrintNodeSchemas prints the available node types
func PrintNodeSchemas(schema *client.SchemaNodesResponse) {
	if len(schema.Nodes) == 0 {
		fmt.Println(tui.Muted.Render("No node types found."))
		return
	}

	fmt.Println(tui.Title.Render(fmt.Sprintf("%d Node Types", len(schema.Nodes))))
	fmt.Println()

	// Group by category
	categories := make(map[string][]client.NodeSchema)
	var order []string
	for _, n := range schema.Nodes {
		cat := n.Category
		if cat == "" {
			cat = "Other"
		}
		if _, exists := categories[cat]; !exists {
			order = append(order, cat)
		}
		categories[cat] = append(categories[cat], n)
	}

	for _, cat := range order {
		nodes := categories[cat]
		fmt.Println(tui.Subtitle.Render(fmt.Sprintf("── %s ──", cat)))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, n := range nodes {
			desc := n.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			fmt.Fprintf(w, "  %s\t%s\n", tui.Bold.Render(n.Type), desc)
		}
		w.Flush()
		fmt.Println()
	}
}

// PrintValidateResult prints pipeline validation results
func PrintValidateResult(r *client.ValidateResponse) {
	if r.Valid {
		fmt.Printf("%s Pipeline is valid\n", tui.SymbolCheck)
	} else {
		fmt.Printf("%s Pipeline is invalid\n", tui.SymbolCross)
		for _, e := range r.Errors {
			fmt.Printf("  %s %s\n", tui.SymbolArrow, e.Message)
		}
	}
}

// PrintEstimateResult prints pipeline cost estimation
func PrintEstimateResult(r *client.EstimateCostResponse) {
	fmt.Println(tui.Subtitle.Render("── Cost Estimate ──"))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%s\t%d credits\n", tui.Bold.Render("Estimated cost"), r.EstimatedCost)
	fmt.Fprintf(w, "%s\t%d\n", tui.Bold.Render("Node count"), r.NodeCount)
	w.Flush()
	if len(r.Breakdown) > 0 {
		fmt.Println()
		fmt.Println(tui.Subtitle.Render("── Breakdown ──"))
		w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for nodeType, cost := range r.Breakdown {
			fmt.Fprintf(w, "  %s\t%d\n", nodeType, cost)
		}
		w.Flush()
	}
}

// PrintSaveResult prints the result of saving a pipeline
func PrintSaveResult(r *client.SavePipelineResponse) {
	fmt.Printf("%s Pipeline saved: %s\n", tui.SymbolCheck, tui.Bold.Render(r.Name))
	fmt.Printf("  %s ID:       %s\n", tui.SymbolArrow, r.ID)
	fmt.Printf("  %s Evaluate: %s\n", tui.SymbolArrow, r.EvaluateURL)
	fmt.Printf("  %s Describe: %s\n", tui.SymbolArrow, r.DescribeURL)
}

// PrintPipelineExport prints a human-readable summary of an exported pipeline
func PrintPipelineExport(p *client.PipelineFull) {
	fmt.Println(tui.Title.Render(p.Name))
	fmt.Printf("%s %s\n", tui.Muted.Render("Pipeline ID:"), p.ID)

	if pd, ok := p.PipelineData["nodes"].(map[string]any); ok {
		fmt.Printf("%s %d\n", tui.Muted.Render("Nodes:"), len(pd))
	}
	if conns, ok := p.PipelineData["connectionMapFwd"].(map[string]any); ok {
		fmt.Printf("%s %d\n", tui.Muted.Render("Connections:"), len(conns))
	}

	fmt.Println()
	fmt.Println(tui.Muted.Render("Use --json or -o <file> to export the full graph data."))
	fmt.Println(tui.Muted.Render("  inodes export " + p.ID + " --json > pipeline.json"))
	fmt.Println(tui.Muted.Render("  inodes export " + p.ID + " -o pipeline.json"))
}

// PrintRunResult prints a per-output summary of a pipeline execution.
// Each entry of writes is { name, path }, in the order they were written.
func PrintRunResult(report *client.PipelineReport, writes []WrittenOutput) {
	duration := report.TotalProcessingTime / time.Millisecond
	for _, wr := range writes {
		out, ok := report.Outputs[wr.Name]
		if !ok {
			continue
		}
		dims := fmt.Sprintf("%dx%d", out.Width, out.Height)
		format := out.Format
		if format == "" {
			format = "png"
		}
		fmt.Fprintf(os.Stderr, "%s %s → %s (%s %s)\n",
			tui.SymbolCheck, tui.Bold.Render(wr.Name), tui.Bold.Render(wr.Path), dims, format)
	}
	fmt.Fprintf(os.Stderr, "%s %dms, %d credits\n",
		tui.SymbolArrow, duration, report.TotalUnitsBillable)
}

// WrittenOutput records that a named output was saved to a path.
type WrittenOutput struct {
	Name string
	Path string
}
