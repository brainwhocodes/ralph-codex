package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Log level constants
const (
	LogLevelInfo    = "INFO"
	LogLevelWarn    = "WARN"
	LogLevelError   = "ERROR"
	LogLevelSuccess = "SUCCESS"
)

// Spinner frame constants
var (
	// Simple arrow spinner for task progress
	SpinnerFrames = []string{">", ">>", ">>>", ">>", ">"}

	// Braille spinner for status bar
	BrailleSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

// Border styles
var (
	// Border styles for boxes
	BorderNormal  = lipgloss.NormalBorder()
	BorderRounded = lipgloss.RoundedBorder()
	BorderDouble  = lipgloss.DoubleBorder()
	BorderThick   = lipgloss.ThickBorder()

	// Box styles
	StyleBox = lipgloss.NewStyle().
			Border(BorderNormal, true, true, true, true).
			BorderForeground(ColorSecondary).
			Padding(1)

	StyleBoxRounded = lipgloss.NewStyle().
			Border(BorderRounded, true, true, true, true).
			BorderForeground(ColorSecondary).
			Padding(1)

	StyleBoxError = lipgloss.NewStyle().
			Border(BorderNormal, true, true, true, true).
			BorderForeground(ColorError).
			Padding(1)
)

// Divider styles
var (
	StyleDivider = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Width(60)

	DividerChar = "─"

	StyleDividerThick = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Width(60)

	DividerCharThick = "═"
)

// Progress bar styles
var (
	StyleProgressEmpty = lipgloss.NewStyle().
				Foreground(TextMuted)

	StyleProgressFilled = lipgloss.NewStyle().
				Foreground(ColorAccent)

	StyleProgressFull = lipgloss.NewStyle().
				Foreground(ColorAccent)

	StyleProgressBar = lipgloss.NewStyle().
				Foreground(ColorAccent)
)

// Circuit breaker styles
var (
	StyleCircuitClosed = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	StyleCircuitHalfOpen = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)

	StyleCircuitOpen = lipgloss.NewStyle().
				Foreground(ColorError).
				Bold(true)
)

// Error panel styles
var (
	StyleErrorPanel = lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(ColorError).
			Padding(1, 2).
			Width(60)

	StyleErrorTitle = lipgloss.NewStyle().
			Foreground(TextPrimary).
			Bold(true).
			Underline(true)

	StyleErrorStack = lipgloss.NewStyle().
			Foreground(TextSecondary).
			Italic(true)

	StyleRetryButton = lipgloss.NewStyle().
				Foreground(TextPrimary).
				Background(ColorAccent).
				Bold(true).
				Padding(0, 2)
)

// Collapsible section styles
var (
	StyleCollapsedHeader = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true)

	StyleExpandedHeader = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	StyleSectionContent = lipgloss.NewStyle().
				Foreground(TextSecondary).
				Padding(0, 2)
)

// Spinner styles
var (
	StyleSpinner = lipgloss.NewStyle().
		Foreground(ColorAccent)
)

// Color scheme for TUI
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#7D56F4") // Purple
	ColorSecondary = lipgloss.Color("#3B82F6") // Blue
	ColorAccent    = lipgloss.Color("#10B981") // Green
	ColorWarning   = lipgloss.Color("#F59E0B") // Amber
	ColorError     = lipgloss.Color("#EF4444") // Red

	// Background colors
	BgPrimary   = lipgloss.Color("#1F2937") // Dark gray
	BgSecondary = lipgloss.Color("#374151") // Lighter gray
	BgHighlight = lipgloss.Color("#4B5563") // Highlight gray

	// Text colors
	TextPrimary   = lipgloss.Color("#F9FAFB") // White
	TextSecondary = lipgloss.Color("#D1D5DB") // Light gray
	TextMuted     = lipgloss.Color("#9CA3AF") // Muted gray
)

// Styles
var (
	// Header style
	StyleHeader = lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(ColorPrimary).
			Bold(true).
			Padding(1, 2).
			Width(60)

	// Status bar
	StyleStatus = lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(BgSecondary).
			Padding(0, 2)

	// Status badges
	StyleStatusInitializing = lipgloss.NewStyle().
				Foreground(TextPrimary).
				Background(ColorSecondary).
				Padding(0, 1).
				Bold(true)

	StyleStatusRunning = lipgloss.NewStyle().
				Foreground(TextPrimary).
				Background(ColorAccent).
				Padding(0, 1).
				Bold(true)

	StyleStatusPaused = lipgloss.NewStyle().
				Foreground(TextPrimary).
				Background(ColorWarning).
				Padding(0, 1).
				Bold(true)

	StyleStatusError = lipgloss.NewStyle().
				Foreground(TextPrimary).
				Background(ColorError).
				Padding(0, 1).
				Bold(true)

	StyleStatusComplete = lipgloss.NewStyle().
				Foreground(TextPrimary).
				Background(ColorPrimary).
				Padding(0, 1).
				Bold(true)

	// Log viewer
	StyleLog = lipgloss.NewStyle().
			Foreground(TextSecondary).
			Background(BgSecondary).
			Padding(1, 2).
			Width(60).
			Height(15)

	// Help text
	StyleHelpKey = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	StyleHelpDesc = lipgloss.NewStyle().
			Foreground(TextMuted)

	// Error message style
	StyleErrorMsg = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	// Info message style
	StyleInfoMsg = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)
)

// StyledLogEntry returns a styled log entry with emoji prefix
func StyledLogEntry(level, message string) string {
	switch level {
	case LogLevelInfo:
		return StyleHelpDesc.Render("ℹ️  " + message)
	case LogLevelWarn:
		return StyleCircuitHalfOpen.Render("⚠️  " + message)
	case LogLevelError:
		return StyleErrorMsg.Render("❌ " + message)
	case LogLevelSuccess:
		return StyleCircuitClosed.Render("✅ " + message)
	default:
		return StyleHelpDesc.Render("   " + message)
	}
}
