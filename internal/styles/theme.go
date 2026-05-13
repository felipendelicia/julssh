package styles

import "github.com/charmbracelet/lipgloss"

var (
	Primary   = lipgloss.Color("#7C3AED")
	Secondary = lipgloss.Color("#A78BFA")
	Muted     = lipgloss.Color("#6B7280")
	ErrColor  = lipgloss.Color("#EF4444")
	Selected  = lipgloss.Color("#374151")
	Border    = lipgloss.Color("#4B5563")

	Title = lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true).
		MarginBottom(1)

	SelectedRow = lipgloss.NewStyle().
		Background(Selected).
		Foreground(Secondary).
		Bold(true)

	NormalRow = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D1D5DB"))

	MutedText = lipgloss.NewStyle().
		Foreground(Muted)

	ErrText = lipgloss.NewStyle().
		Foreground(ErrColor)

	Tag = lipgloss.NewStyle().
		Foreground(Secondary).
		Background(lipgloss.Color("#2D1B69")).
		Padding(0, 1)

	Footer = lipgloss.NewStyle().
		Foreground(Muted).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(Border).
		MarginTop(1).
		PaddingTop(1)

	FieldLabel = lipgloss.NewStyle().
		Foreground(Secondary).
		Width(15)

	ActiveInput = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F9FAFB")).
		Background(Selected).
		Padding(0, 1)

	InactiveInput = lipgloss.NewStyle().
		Foreground(Muted).
		Padding(0, 1)

	InlineErr = lipgloss.NewStyle().
		Foreground(ErrColor).
		MarginLeft(15)
)
