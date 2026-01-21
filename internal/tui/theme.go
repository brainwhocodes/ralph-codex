package tui

import "github.com/charmbracelet/lipgloss"

// Charmtone color palette from charmbracelet/x/exp/charmtone
var (
	// Primary brand colors
	Charple = lipgloss.Color("#6B50FF") // Purple - primary brand color
	Dolly   = lipgloss.Color("#FF60FF") // Bright pink/magenta - secondary
	Julep   = lipgloss.Color("#00FFB2") // Bright mint/green - success accent
	Zest    = lipgloss.Color("#E8FE96") // Lime yellow - warning/accent
	Butter  = lipgloss.Color("#FFFAF1") // Warm white

	// Background colors (dark theme)
	Pepper   = lipgloss.Color("#201F26") // Deep dark - main background
	BBQ      = lipgloss.Color("#2D2C35") // Slightly lighter - panels
	Charcoal = lipgloss.Color("#3A3943") // Borders, subtle backgrounds
	Iron     = lipgloss.Color("#4D4C57") // Overlays, highlights

	// Foreground/text colors
	Salt   = lipgloss.Color("#F1EFEF") // Bright white - selected text
	Ash    = lipgloss.Color("#DFDBDD") // Primary text
	Smoke  = lipgloss.Color("#BFBCC8") // Secondary text
	Squid  = lipgloss.Color("#858392") // Muted text
	Oyster = lipgloss.Color("#605F6B") // Very muted/subtle

	// Status colors
	Guac     = lipgloss.Color("#12C78F") // Success - green
	Sriracha = lipgloss.Color("#EB4268") // Error - red/pink
	Malibu   = lipgloss.Color("#00A4FF") // Info - blue
	Coral    = lipgloss.Color("#FF577D") // Error alt - coral red
)

// Icons - Crush-style
const (
	IconCheck       = "✓"
	IconError       = "×"
	IconWarning     = "⚠"
	IconInfo        = "ⓘ"
	IconPending     = "•"
	IconInProgress  = "●"
	IconArrowRight  = "→"
	IconBorderThin  = "│"
	IconBorderThick = "▌"
	IconDiagonal    = "╱"
)

// SAX animation frames - musical notes cycle when running
// All frames have notes - no empty frame to avoid "reverting" effect
var SaxNotes = []string{
	"SAX ♩",
	"SAX ♪",
	"SAX ♫",
	"SAX ♬",
	"SAX ♫",
	"SAX ♪",
}

// ThinkingWave - animated thinking indicator
var ThinkingWave = []string{
	"∿",
	"∿∿",
	"∿∿∿",
	"∿∿",
	"∿",
}

// Theme holds all the semantic color mappings
type Theme struct {
	Name string

	// Brand
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color

	// Backgrounds
	BgBase   lipgloss.Color
	BgPanel  lipgloss.Color
	BgBorder lipgloss.Color
	BgOverlay lipgloss.Color

	// Foregrounds
	FgBase     lipgloss.Color
	FgSelected lipgloss.Color
	FgMuted    lipgloss.Color
	FgSubtle   lipgloss.Color

	// Status
	Success lipgloss.Color
	Error   lipgloss.Color
	Warning lipgloss.Color
	Info    lipgloss.Color
}

// DefaultTheme returns the Charmtone-based dark theme
func DefaultTheme() *Theme {
	return &Theme{
		Name: "charmtone",

		// Brand colors
		Primary:   Charple,
		Secondary: Dolly,
		Accent:    Zest,

		// Backgrounds
		BgBase:    Pepper,
		BgPanel:   BBQ,
		BgBorder:  Charcoal,
		BgOverlay: Iron,

		// Foregrounds
		FgBase:     Ash,
		FgSelected: Salt,
		FgMuted:    Squid,
		FgSubtle:   Oyster,

		// Status
		Success: Guac,
		Error:   Sriracha,
		Warning: Zest,
		Info:    Malibu,
	}
}

// Gradient renders text with a two-color gradient effect
// This is a simplified version - actual gradient would use ANSI
func GradientText(text string, from, to lipgloss.Color) string {
	// For now, use the primary color with bold
	// A true gradient would interpolate between colors character by character
	return lipgloss.NewStyle().
		Foreground(from).
		Bold(true).
		Render(text)
}

// DiagonalSeparator creates a string of diagonal lines
func DiagonalSeparator(width int) string {
	if width <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < width; i++ {
		result += IconDiagonal
	}
	return result
}
