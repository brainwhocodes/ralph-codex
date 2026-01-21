package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderSplitView renders the main split pane layout (tasks top, output bottom)
func (m Model) renderSplitView() string {
	width := m.width
	height := m.height
	if width < 80 {
		width = 80
	}
	if height < 24 {
		height = 24
	}

	// Header height + status bar + footer
	headerHeight := 1
	statusHeight := 1
	footerHeight := 1
	contentHeight := height - headerHeight - statusHeight - footerHeight - 2

	// Split content vertically: 40% tasks, 60% output
	topHeight := (contentHeight * 40) / 100
	bottomHeight := contentHeight - topHeight - 1 // 1 for divider

	// Render header
	header := m.renderHeader(width)

	// Render status bar
	statusBar := m.renderStatusBar(width)

	// Render top pane (tasks)
	topPane := m.renderTaskPane(width, topHeight)

	// Render horizontal divider
	divider := m.renderHorizontalDivider(width)

	// Render bottom pane (output)
	bottomPane := m.renderOutputPane(width, bottomHeight)

	// Render footer
	footer := m.renderFooter(width)

	// Join everything vertically
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		statusBar,
		topPane,
		divider,
		bottomPane,
		footer,
	)
}

// renderHeader renders the Crush-style header with gradient text and diagonal separators
// Format: Charm LISA SAX ♫ ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱ mode • loop 2 • 3/10
func (m Model) renderHeader(width int) string {
	// Brand prefix and name
	brandPrefix := StyleBrandPrefix.Render("Charm")
	brandName := GradientText("LISA", Dolly, Charple)

	// Animated SAX with musical notes when running
	var saxAnim string
	if m.state == StateRunning {
		frame := m.tick % len(SaxNotes)
		saxAnim = " " + StyleBrandPrefix.Render(SaxNotes[frame])
	}

	// Build metadata on right side
	var metaParts []string

	// Mode
	modeName := string(m.projectMode)
	if modeName == "" {
		modeName = "ready"
	}
	metaParts = append(metaParts, modeName)

	// Loop number
	metaParts = append(metaParts, fmt.Sprintf("loop %d", m.loopNumber))

	// Task progress
	completed := 0
	for _, t := range m.tasks {
		if t.Completed {
			completed++
		}
	}
	if len(m.tasks) > 0 {
		metaParts = append(metaParts, fmt.Sprintf("%d/%d", completed, len(m.tasks)))
	}

	metadata := StyleHeaderMeta.Render(strings.Join(metaParts, MetaDotSeparator))

	// Calculate space for diagonal separators (include SAX animation width)
	brandWidth := lipgloss.Width(brandPrefix) + 1 + lipgloss.Width(brandName) + lipgloss.Width(saxAnim) + 1
	metaWidth := lipgloss.Width(metadata) + 1
	diagWidth := width - brandWidth - metaWidth - 2

	if diagWidth < 3 {
		diagWidth = 3
	}

	diagonals := StyleDiagonal.Render(DiagonalSeparator(diagWidth))

	// Assemble header
	leftPart := fmt.Sprintf("%s %s%s ", brandPrefix, brandName, saxAnim)
	headerContent := leftPart + diagonals + " " + metadata

	return StyleHeader.Width(width).Render(headerContent)
}

// renderStatusBar renders the status bar with state indicator, spinner, and circuit state
// Format: ● running  STATUS: WORKING  tasks: 2  files: 3     circuit closed
func (m Model) renderStatusBar(width int) string {
	// State indicator with icon
	var stateIcon, stateText string
	var stateStyle lipgloss.Style

	switch m.state {
	case StateInitializing:
		stateIcon = IconInProgress
		stateText = "initializing"
		stateStyle = StyleInfoMsg
	case StateRunning:
		// Animated spinner for running
		stateIcon = BrailleSpinnerFrames[m.tick%len(BrailleSpinnerFrames)]
		stateText = "running"
		stateStyle = StyleSuccessMsg
	case StatePaused:
		stateIcon = IconPending
		stateText = "paused"
		stateStyle = StyleWarningMsg
	case StateComplete:
		stateIcon = IconCheck
		stateText = "complete"
		stateStyle = StyleSuccessMsg
	case StateError:
		stateIcon = IconError
		stateText = "error"
		stateStyle = StyleErrorMsg
	}

	leftStatus := stateStyle.Render(stateIcon + " " + stateText)

	// Add analysis status in the middle if available
	var midStatus string
	if m.analysisStatus != "" {
		var statusStyle lipgloss.Style
		switch m.analysisStatus {
		case "COMPLETE":
			statusStyle = StyleSuccessMsg
		case "BLOCKED":
			statusStyle = StyleErrorMsg
		default:
			statusStyle = StyleInfoMsg
		}
		midStatus = StyleTextMuted.Render(" │ ") + statusStyle.Render(m.analysisStatus)

		if m.tasksCompleted > 0 {
			midStatus += StyleTextMuted.Render(fmt.Sprintf(" tasks:%d", m.tasksCompleted))
		}
		if m.filesModified > 0 {
			midStatus += StyleTextMuted.Render(fmt.Sprintf(" files:%d", m.filesModified))
		}
		if m.exitSignal {
			midStatus += StyleSuccessMsg.Render(" EXIT")
		}
	}

	// Context usage indicator (before circuit)
	var contextIndicator string
	if m.contextLimit > 0 {
		usagePct := int(m.contextUsagePercent * 100)
		var ctxStyle lipgloss.Style
		switch {
		case m.contextThreshold:
			ctxStyle = StyleErrorMsg // Red when threshold reached
		case m.contextUsagePercent >= 0.6:
			ctxStyle = StyleWarningMsg // Yellow when > 60%
		default:
			ctxStyle = StyleTextMuted // Normal
		}
		// Mini progress bar for context
		barWidth := 10
		filled := int(m.contextUsagePercent * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}
		empty := barWidth - filled
		contextIndicator = StyleTextMuted.Render("ctx ") +
			ctxStyle.Render(fmt.Sprintf("%d%%", usagePct)) +
			StyleTextMuted.Render(" [") +
			ctxStyle.Render(strings.Repeat("█", filled)) +
			StyleProgressEmpty.Render(strings.Repeat("░", empty)) +
			StyleTextMuted.Render("]")
		if m.contextWasCompacted {
			contextIndicator += StyleWarningMsg.Render(" ⟳")
		}
	}

	// Circuit state on right
	circuitState := m.circuitState
	if circuitState == "" {
		circuitState = "closed"
	}
	circuitState = strings.ToLower(circuitState)

	var circuitStyle lipgloss.Style
	switch circuitState {
	case "closed":
		circuitStyle = StyleCircuitClosed
	case "half_open":
		circuitStyle = StyleCircuitHalfOpen
		circuitState = "half-open"
	case "open":
		circuitStyle = StyleCircuitOpen
	default:
		circuitStyle = StyleTextMuted
	}

	rightStatus := ""
	if contextIndicator != "" {
		rightStatus = contextIndicator + StyleTextMuted.Render("  ")
	}
	rightStatus += StyleTextMuted.Render("circuit ") + circuitStyle.Render(circuitState)

	// Calculate padding between left and right
	leftWidth := lipgloss.Width(leftStatus) + lipgloss.Width(midStatus)
	rightWidth := lipgloss.Width(rightStatus)
	paddingWidth := width - leftWidth - rightWidth - 4
	if paddingWidth < 1 {
		paddingWidth = 1
	}

	statusContent := " " + leftStatus + midStatus + strings.Repeat(" ", paddingWidth) + rightStatus

	return StyleStatus.Width(width).Render(statusContent)
}

// renderHorizontalDivider renders a subtle horizontal divider
func (m Model) renderHorizontalDivider(width int) string {
	return StyleDivider.Render(strings.Repeat(DividerChar, width))
}

// renderTaskPane renders the tasks pane with Crush-style icons
func (m Model) renderTaskPane(width, height int) string {
	var lines []string

	// Count completed tasks
	completed := 0
	for _, t := range m.tasks {
		if t.Completed {
			completed++
		}
	}

	if len(m.tasks) == 0 {
		lines = append(lines, StyleTextMuted.Render(" No tasks loaded"))
	} else {
		// Render tasks with icons
		for i, task := range m.tasks {
			if i >= height-2 { // Leave room for summary
				remaining := len(m.tasks) - i
				lines = append(lines, StyleTextSubtle.Render(fmt.Sprintf(" ... %d more", remaining)))
				break
			}
			lines = append(lines, m.renderTaskLine(task, i, width-2))
		}

		// Task summary
		lines = append(lines, "")
		summary := fmt.Sprintf(" %d of %d complete", completed, len(m.tasks))
		lines = append(lines, StyleTextMuted.Render(summary))
	}

	// Pad to fill height
	for len(lines) < height {
		lines = append(lines, "")
	}

	content := strings.Join(lines[:height], "\n")
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(content)
}

// renderTaskLine renders a single task with Crush-style icons
func (m Model) renderTaskLine(task Task, index int, maxWidth int) string {
	isActive := index == m.activeTaskIdx && m.state == StateRunning
	text := task.Text

	// Truncate if needed
	if maxWidth > 10 && len(text) > maxWidth-6 {
		text = text[:maxWidth-9] + "..."
	}

	// Get icon based on state
	var icon string
	var textStyle lipgloss.Style

	if task.Completed {
		icon = StyleTaskCompleted.Render(IconCheck)
		textStyle = StyleTaskTextCompleted
	} else if isActive {
		// Use animated spinner for active task
		spinnerFrame := BrailleSpinnerFrames[m.tick%len(BrailleSpinnerFrames)]
		icon = StyleTaskInProgress.Render(spinnerFrame)
		textStyle = StyleTaskTextActive
	} else {
		icon = StyleTaskPending.Render(IconPending)
		textStyle = StyleTaskTextPending
	}

	return " " + icon + " " + textStyle.Render(text)
}

// backendDisplayName returns a display-friendly name for the backend
func (m Model) backendDisplayName() string {
	switch m.backend {
	case "opencode":
		return "OpenCode"
	case "cli":
		return "Codex CLI"
	default:
		if m.backend != "" {
			return m.backend
		}
		return "agent"
	}
}

// renderOutputPane renders the output pane with live agent output
func (m Model) renderOutputPane(width, height int) string {
	var lines []string

	// Calculate space for reasoning (if any)
	reasoningHeight := 0
	if len(m.reasoningLines) > 0 {
		reasoningHeight = 3 // divider + thinking line + padding
	}

	// Show reasoning at the top if we have it
	if len(m.reasoningLines) > 0 {
		// Get the latest reasoning (last line only to avoid clutter)
		latestReasoning := m.reasoningLines[len(m.reasoningLines)-1]
		// Truncate if too long
		if len(latestReasoning) > width-20 {
			latestReasoning = latestReasoning[:width-23] + "..."
		}
		// Animated thinking indicator
		thinkAnim := ThinkingWave[m.tick%len(ThinkingWave)]
		lines = append(lines, StyleReasoning.Render(" ["+thinkAnim+"] "+latestReasoning))
		lines = append(lines, "")
	}

	if len(m.outputLines) == 0 && len(m.reasoningLines) == 0 {
		lines = append(lines, StyleTextMuted.Render(fmt.Sprintf(" Waiting for %s output...", m.backendDisplayName())))
	} else if len(m.outputLines) > 0 {
		// Show most recent output lines that fit
		maxLines := height - reasoningHeight - 2
		start := 0
		if len(m.outputLines) > maxLines {
			start = len(m.outputLines) - maxLines
		}

		for i := start; i < len(m.outputLines); i++ {
			line := m.outputLines[i]
			// Truncate long lines
			if len(line) > width-4 {
				line = line[:width-7] + "..."
			}
			lines = append(lines, " "+line)
		}
	}

	// Pad to fill height
	for len(lines) < height {
		lines = append(lines, "")
	}

	content := strings.Join(lines[:height], "\n")
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(content)
}

// renderFooter renders the Crush-style footer with keybindings
// Format: r run • p pause • l logs • c circuit • ? help • q quit
func (m Model) renderFooter(width int) string {
	bindings := []struct {
		key  string
		desc string
	}{
		{"r", "run"},
		{"p", "pause"},
		{"l", "logs"},
		{"c", "circuit"},
		{"t", "tasks"},
		{"o", "output"},
		{"?", "help"},
		{"q", "quit"},
	}

	var parts []string
	for _, b := range bindings {
		parts = append(parts, StyleHelpKey.Render(b.key)+" "+StyleHelpDesc.Render(b.desc))
	}

	footerContent := " " + strings.Join(parts, StyleTextSubtle.Render(MetaDotSeparator))

	return StyleFooter.Width(width).Render(footerContent)
}

// renderTasksFullView renders tasks in full screen mode
func (m Model) renderTasksFullView() string {
	width := m.width
	if width < 60 {
		width = 60
	}

	header := m.renderHeader(width)

	// Task progress header
	completed := 0
	for _, t := range m.tasks {
		if t.Completed {
			completed++
		}
	}

	var lines []string

	progressPct := 0
	if len(m.tasks) > 0 {
		progressPct = (completed * 100) / len(m.tasks)
	}
	lines = append(lines, StyleTextBase.Render(fmt.Sprintf(" Progress: %d/%d (%d%%)", completed, len(m.tasks), progressPct)))

	// Progress bar
	barWidth := 40
	filledCount := 0
	if len(m.tasks) > 0 {
		filledCount = (completed * barWidth) / len(m.tasks)
	}
	emptyCount := barWidth - filledCount
	taskProgressBar := " " + StyleProgressFilled.Render(strings.Repeat("█", filledCount)) +
		StyleProgressEmpty.Render(strings.Repeat("░", emptyCount))
	lines = append(lines, taskProgressBar)
	lines = append(lines, "")
	lines = append(lines, StyleDivider.Render(strings.Repeat(DividerChar, width-4)))
	lines = append(lines, "")

	// All tasks
	for i, task := range m.tasks {
		lines = append(lines, m.renderTaskLine(task, i, width-4))
	}

	content := strings.Join(lines, "\n")

	footer := StyleFooter.Width(width).Render(
		fmt.Sprintf(" %s return%s%s quit",
			StyleHelpKey.Render("t"),
			StyleTextSubtle.Render(MetaDotSeparator),
			StyleHelpKey.Render("q")),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		content,
		"",
		footer,
	)
}

// renderOutputFullView renders output in full screen mode
func (m Model) renderOutputFullView() string {
	width := m.width
	height := m.height
	if width < 60 {
		width = 60
	}
	if height < 20 {
		height = 20
	}

	header := m.renderHeader(width)

	var lines []string

	// Show reasoning at the top if we have it
	if len(m.reasoningLines) > 0 {
		latestReasoning := m.reasoningLines[len(m.reasoningLines)-1]
		if len(latestReasoning) > width-20 {
			latestReasoning = latestReasoning[:width-23] + "..."
		}
		// Animated thinking indicator
		thinkAnim := ThinkingWave[m.tick%len(ThinkingWave)]
		lines = append(lines, StyleReasoning.Render(" ["+thinkAnim+"] "+latestReasoning))
		lines = append(lines, "")
	}

	if len(m.outputLines) == 0 && len(m.reasoningLines) == 0 {
		lines = append(lines, StyleTextMuted.Render(fmt.Sprintf(" Waiting for %s output...", m.backendDisplayName())))
	} else if len(m.outputLines) > 0 {
		// Show most recent output lines
		maxLines := height - 8
		start := 0
		if len(m.outputLines) > maxLines {
			start = len(m.outputLines) - maxLines
		}

		for i := start; i < len(m.outputLines); i++ {
			line := m.outputLines[i]
			if len(line) > width-4 {
				line = line[:width-7] + "..."
			}
			lines = append(lines, " "+line)
		}
	}

	content := strings.Join(lines, "\n")

	footer := StyleFooter.Width(width).Render(
		fmt.Sprintf(" %s return%s%s quit",
			StyleHelpKey.Render("o"),
			StyleTextSubtle.Render(MetaDotSeparator),
			StyleHelpKey.Render("q")),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		content,
		"",
		footer,
	)
}

// renderLogsFullView renders logs in full screen mode
func (m Model) renderLogsFullView() string {
	width := m.width
	height := m.height
	if width < 60 {
		width = 60
	}
	if height < 20 {
		height = 20
	}

	header := m.renderHeader(width)

	var lines []string

	if len(m.logs) == 0 {
		lines = append(lines, StyleTextMuted.Render(" No log entries yet..."))
	} else {
		// Show all available logs
		maxLines := height - 6
		start := 0
		if len(m.logs) > maxLines {
			start = len(m.logs) - maxLines
		}

		for i := start; i < len(m.logs); i++ {
			lines = append(lines, " "+m.logs[i])
		}
	}

	content := strings.Join(lines, "\n")

	footer := StyleFooter.Width(width).Render(
		fmt.Sprintf(" %s return%s%d entries",
			StyleHelpKey.Render("l"),
			StyleTextSubtle.Render(MetaDotSeparator),
			len(m.logs)),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		content,
		"",
		footer,
	)
}
