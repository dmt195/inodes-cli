package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#6366F1") // Indigo
	ColorSuccess   = lipgloss.Color("#10B981") // Green
	ColorError     = lipgloss.Color("#EF4444") // Red
	ColorWarning   = lipgloss.Color("#F59E0B") // Amber
	ColorMuted     = lipgloss.Color("#6B7280") // Gray
	ColorWhite     = lipgloss.Color("#F9FAFB")

	// Styles
	Title = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	Subtitle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	Success = lipgloss.NewStyle().
		Foreground(ColorSuccess)

	Error = lipgloss.NewStyle().
		Foreground(ColorError)

	Warning = lipgloss.NewStyle().
		Foreground(ColorWarning)

	Muted = lipgloss.NewStyle().
		Foreground(ColorMuted)

	Bold = lipgloss.NewStyle().
		Bold(true)

	// Symbols
	SymbolCheck = Success.Render("✓")
	SymbolCross = Error.Render("✗")
	SymbolArrow = Subtitle.Render("▸")
	SymbolDot   = Muted.Render("·")
)
