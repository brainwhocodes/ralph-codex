package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Keybinding represents a keybinding help entry
type Keybinding struct {
	Key         string
	Description string
}

// KeybindingSection represents a section of keybindings
type KeybindingSection struct {
	Title string
	Keys  []Keybinding
}

// GetKeybindingHelp returns formatted help text for all keybindings
func GetKeybindingHelp() string {
	sections := []KeybindingSection{
		{
			Title: "Navigation",
			Keys: []Keybinding{
				{"q / Ctrl+C", "Quit Ralph Codex"},
				{"?", "Toggle help screen"},
			},
		},
		{
			Title: "Loop Control",
			Keys: []Keybinding{
				{"r", "Run / Restart loop"},
				{"p", "Pause / Resume loop"},
			},
		},
		{
			Title: "Views",
			Keys: []Keybinding{
				{"c", "Show circuit breaker status"},
				{"R", "Reset circuit breaker"},
			},
		},
		{
			Title: "CLI Options",
			Keys: []Keybinding{
				{"--monitor", "Enable integrated TUI monitoring"},
				{"--verbose", "Verbose output"},
				{"--backend cli", "Use CLI backend"},
				{"--backend sdk", "Use SDK backend"},
			},
		},
		{
			Title: "Project Options",
			Keys: []Keybinding{
				{"--project <path>", "Set project directory"},
				{"--prompt <file>", "Set prompt file"},
			},
		},
		{
			Title: "Rate Limiting",
			Keys: []Keybinding{
				{"--calls <num>", "Max API calls per hour"},
				{"--timeout <sec>", "Codex execution timeout"},
			},
		},
		{
			Title: "Project Commands",
			Keys: []Keybinding{
				{"setup --name <proj>", "Create new project"},
				{"import --source <file>", "Import PRD/document"},
				{"status", "Show project status"},
				{"reset-circuit", "Reset circuit breaker"},
			},
		},
		{
			Title: "Troubleshooting",
			Keys: []Keybinding{
				{"Press 'r' after error", "Retry failed operation"},
				{"Press 'R' after loop", "Reset circuit if stuck"},
				{"Check logs with 'l'", "View detailed execution logs"},
			},
		},
	}

	var builder strings.Builder

	for _, section := range sections {
		builder.WriteString(StyleHeader.Render(section.Title))
		builder.WriteString("\n\n")

		for _, keybinding := range section.Keys {
			builder.WriteString(fmt.Sprintf("  %s %s\n",
				StyleHelpKey.Render(keybinding.Key),
				StyleHelpDesc.Render(keybinding.Description)))
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

// RenderHelpScreen returns the full help screen
func (m Model) renderHelpView() string {
	width := m.width
	if width < 60 {
		width = 60
	}
	height := m.height
	if height < 20 {
		height = 20
	}

	const headerHeight = 3
	const footerHeight = 1

	header := StyleHeader.Copy().Width(width).Render("Ralph Codex - Help")

	version := StyleHelpDesc.Render("Version 1.0.0")

	divider := StyleDivider.Copy().Width(width).Render(strings.Repeat("─", width))

	helpContent := GetKeybindingHelp()

	middleContent := version + "\n" + divider + "\n\n" + helpContent
	middleHeight := height - headerHeight - footerHeight - 2
	if middleHeight < 10 {
		middleHeight = 10
	}

	middleContainer := lipgloss.NewStyle().
		Width(width).
		Height(middleHeight).
		Render(middleContent)

	// Footer
	footer := StyleStatus.Copy().Width(width).Render(
		fmt.Sprintf(" %s Return to status  %s",
			StyleHelpKey.Render("?"),
			StyleInfoMsg.Render("Tip: Use --monitor flag for TUI mode")),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		middleContainer,
		footer,
	)
}
