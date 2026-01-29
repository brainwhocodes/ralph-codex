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

	// Phase-aware task progress
	if len(m.phases) > 0 {
		currentPhaseIdx := m.getCurrentPhaseIndex()
		if currentPhaseIdx >= 0 && currentPhaseIdx < len(m.phases) {
			phase := m.phases[currentPhaseIdx]
			completed := 0
			for _, t := range phase.Tasks {
				if t.Completed {
					completed++
				}
			}
			metaParts = append(metaParts, fmt.Sprintf("P%d:%d/%d", currentPhaseIdx+1, completed, len(phase.Tasks)))
		}
	} else if len(m.tasks) > 0 {
		// Fallback to flat task progress
		completed := 0
		for _, t := range m.tasks {
			if t.Completed {
				completed++
			}
		}
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
// Shows only current phase tasks with phase header
func (m Model) renderTaskPane(width, height int) string {
	var lines []string

	// Check for phases
	if len(m.phases) == 0 {
		// Fallback to flat task list if no phases
		return m.renderFlatTaskPane(width, height)
	}

	// Get current phase (auto-advance if current is complete)
	currentPhaseIdx := m.getCurrentPhaseIndex()
	if currentPhaseIdx < 0 || currentPhaseIdx >= len(m.phases) {
		currentPhaseIdx = 0
	}
	phase := m.phases[currentPhaseIdx]

	// Phase header with progress indicator
	phaseCompleted := 0
	for _, t := range phase.Tasks {
		if t.Completed {
			phaseCompleted++
		}
	}

	// Animated phase indicator when running
	var phaseIcon string
	if phase.Completed {
		phaseIcon = StyleTaskCompleted.Render(IconCheck)
	} else if m.state == StateRunning {
		spinnerFrame := BrailleSpinnerFrames[m.tick%len(BrailleSpinnerFrames)]
		phaseIcon = StyleTaskInProgress.Render(spinnerFrame)
	} else {
		phaseIcon = StyleTaskPending.Render(IconInProgress)
	}

	// Phase header: "● Phase 1: Foundation [2/4]"
	phaseHeader := fmt.Sprintf(" %s %s [%d/%d]", phaseIcon, phase.Name, phaseCompleted, len(phase.Tasks))
	lines = append(lines, StyleTextBase.Render(phaseHeader))
	lines = append(lines, "")

	// Render phase tasks with icons
	for i, task := range phase.Tasks {
		if i >= height-4 { // Leave room for header and summary
			remaining := len(phase.Tasks) - i
			lines = append(lines, StyleTextSubtle.Render(fmt.Sprintf(" ... %d more", remaining)))
			break
		}
		lines = append(lines, m.renderPhaseTaskLine(task, i, currentPhaseIdx, width-2))
	}

	// Phase summary with overall progress
	lines = append(lines, "")
	totalPhases := len(m.phases)
	completedPhases := 0
	for _, p := range m.phases {
		if p.Completed {
			completedPhases++
		}
	}
	summary := fmt.Sprintf(" Phase %d/%d • %d/%d tasks", currentPhaseIdx+1, totalPhases, phaseCompleted, len(phase.Tasks))
	lines = append(lines, StyleTextMuted.Render(summary))

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

// renderFlatTaskPane renders tasks without phase grouping (fallback)
func (m Model) renderFlatTaskPane(width, height int) string {
	var lines []string

	completed := 0
	for _, t := range m.tasks {
		if t.Completed {
			completed++
		}
	}

	if len(m.tasks) == 0 {
		lines = append(lines, StyleTextMuted.Render(" No tasks loaded"))
	} else {
		for i, task := range m.tasks {
			if i >= height-2 {
				remaining := len(m.tasks) - i
				lines = append(lines, StyleTextSubtle.Render(fmt.Sprintf(" ... %d more", remaining)))
				break
			}
			lines = append(lines, m.renderTaskLine(task, i, width-2))
		}
		lines = append(lines, "")
		summary := fmt.Sprintf(" %d of %d complete", completed, len(m.tasks))
		lines = append(lines, StyleTextMuted.Render(summary))
	}

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

// getCurrentPhaseIndex returns the index of the first incomplete phase
func (m Model) getCurrentPhaseIndex() int {
	for i, phase := range m.phases {
		if !phase.Completed {
			return i
		}
	}
	// All complete, return last
	if len(m.phases) > 0 {
		return len(m.phases) - 1
	}
	return 0
}

// renderPhaseTaskLine renders a task line within a phase context
func (m Model) renderPhaseTaskLine(task Task, taskIdx, phaseIdx int, maxWidth int) string {
	// Calculate global task index for active tracking
	globalIdx := 0
	for i := 0; i < phaseIdx; i++ {
		globalIdx += len(m.phases[i].Tasks)
	}
	globalIdx += taskIdx

	isActive := globalIdx == m.activeTaskIdx && m.state == StateRunning
	text := task.Text

	// Truncate if needed
	if maxWidth > 10 && len(text) > maxWidth-6 {
		text = text[:maxWidth-9] + "..."
	}

	var icon string
	var textStyle lipgloss.Style

	if task.Completed {
		icon = StyleTaskCompleted.Render(IconCheck)
		textStyle = StyleTaskTextCompleted
	} else if isActive {
		spinnerFrame := BrailleSpinnerFrames[m.tick%len(BrailleSpinnerFrames)]
		icon = StyleTaskInProgress.Render(spinnerFrame)
		textStyle = StyleTaskTextActive
	} else {
		icon = StyleTaskPending.Render(IconPending)
		textStyle = StyleTaskTextPending
	}

	return " " + icon + " " + textStyle.Render(text)
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
// Shows all phases with current phase expanded
func (m Model) renderTasksFullView() string {
	width := m.width
	if width < 60 {
		width = 60
	}

	header := m.renderHeader(width)

	var lines []string

	// If we have phases, show phase-organized view
	if len(m.phases) > 0 {
		currentPhaseIdx := m.getCurrentPhaseIndex()

		// Overall progress
		totalTasks := 0
		completedTasks := 0
		for _, phase := range m.phases {
			for _, t := range phase.Tasks {
				totalTasks++
				if t.Completed {
					completedTasks++
				}
			}
		}

		progressPct := 0
		if totalTasks > 0 {
			progressPct = (completedTasks * 100) / totalTasks
		}
		lines = append(lines, StyleTextBase.Render(fmt.Sprintf(" Overall: %d/%d (%d%%)", completedTasks, totalTasks, progressPct)))

		// Progress bar
		barWidth := 40
		filledCount := 0
		if totalTasks > 0 {
			filledCount = (completedTasks * barWidth) / totalTasks
		}
		emptyCount := barWidth - filledCount
		taskProgressBar := " " + StyleProgressFilled.Render(strings.Repeat("█", filledCount)) +
			StyleProgressEmpty.Render(strings.Repeat("░", emptyCount))
		lines = append(lines, taskProgressBar)
		lines = append(lines, "")

		// Render each phase
		globalTaskIdx := 0
		for phaseIdx, phase := range m.phases {
			// Phase header with icon
			var phaseIcon string
			if phase.Completed {
				phaseIcon = StyleTaskCompleted.Render(IconCheck)
			} else if phaseIdx == currentPhaseIdx && m.state == StateRunning {
				spinnerFrame := BrailleSpinnerFrames[m.tick%len(BrailleSpinnerFrames)]
				phaseIcon = StyleTaskInProgress.Render(spinnerFrame)
			} else if phaseIdx == currentPhaseIdx {
				phaseIcon = StyleTaskPending.Render(IconInProgress)
			} else {
				phaseIcon = StyleTextMuted.Render(IconPending)
			}

			// Count phase progress
			phaseCompleted := 0
			for _, t := range phase.Tasks {
				if t.Completed {
					phaseCompleted++
				}
			}

			phaseHeader := fmt.Sprintf(" %s %s [%d/%d]", phaseIcon, phase.Name, phaseCompleted, len(phase.Tasks))
			if phaseIdx == currentPhaseIdx {
				lines = append(lines, StyleTextBase.Render(phaseHeader))
			} else {
				lines = append(lines, StyleTextMuted.Render(phaseHeader))
			}

			// Show tasks for current phase, collapse others
			if phaseIdx == currentPhaseIdx {
				for taskIdx, task := range phase.Tasks {
					lines = append(lines, m.renderPhaseTaskLine(task, taskIdx, phaseIdx, width-4))
					globalTaskIdx++
				}
			} else {
				globalTaskIdx += len(phase.Tasks)
			}
			lines = append(lines, "")
		}
	} else {
		// Fallback: flat task list
		completed := 0
		for _, t := range m.tasks {
			if t.Completed {
				completed++
			}
		}

		progressPct := 0
		if len(m.tasks) > 0 {
			progressPct = (completed * 100) / len(m.tasks)
		}
		lines = append(lines, StyleTextBase.Render(fmt.Sprintf(" Progress: %d/%d (%d%%)", completed, len(m.tasks), progressPct)))

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

		for i, task := range m.tasks {
			lines = append(lines, m.renderTaskLine(task, i, width-4))
		}
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

// renderPreflightSummary renders a preflight summary panel
// Shows mode, plan file, remaining tasks, circuit state, and rate limit status
func (m Model) renderPreflightSummary(width int) string {
	var lines []string

	// Title
	lines = append(lines, StyleTextBase.Render(" Preflight Check"))
	lines = append(lines, StyleTextMuted.Render(" "+strings.Repeat("─", width-4)))

	// Skip reason (if any)
	if m.preflightShouldSkip {
		lines = append(lines, "")
		lines = append(lines, StyleErrorMsg.Render(fmt.Sprintf(" ⚠ Skipped: %s", m.preflightSkipReason)))
	}

	// Mode and plan file
	lines = append(lines, "")
	if m.preflightMode != "" {
		lines = append(lines, fmt.Sprintf(" %s %s",
			StyleTextMuted.Render("Mode:"),
			StyleTextBase.Render(m.preflightMode)))
	}
	if m.preflightPlanFile != "" {
		lines = append(lines, fmt.Sprintf(" %s %s",
			StyleTextMuted.Render("Plan:"),
			StyleTextBase.Render(m.preflightPlanFile)))
	}

	// Task summary
	if m.preflightTotalTasks > 0 {
		lines = append(lines, "")
		progress := m.preflightTotalTasks - m.preflightRemainingCount
		pct := 0
		if m.preflightTotalTasks > 0 {
			pct = (progress * 100) / m.preflightTotalTasks
		}

		// Progress bar
		barWidth := 20
		filled := 0
		if m.preflightTotalTasks > 0 {
			filled = (progress * barWidth) / m.preflightTotalTasks
		}
		empty := barWidth - filled
		bar := StyleProgressFilled.Render(strings.Repeat("█", filled)) +
			StyleProgressEmpty.Render(strings.Repeat("░", empty))

		lines = append(lines, fmt.Sprintf(" %s %d/%d (%d%%) %s",
			StyleTextMuted.Render("Progress:"),
			progress,
			m.preflightTotalTasks,
			pct,
			bar))
	}

	// Circuit state
	circuitState := m.preflightCircuitState
	if circuitState == "" {
		circuitState = m.circuitState
	}
	if circuitState == "" {
		circuitState = "CLOSED"
	}
	circuitState = strings.ToLower(circuitState)

	var circuitStyle lipgloss.Style
	switch circuitState {
	case "closed":
		circuitStyle = StyleCircuitClosed
	case "half_open", "half-open":
		circuitStyle = StyleCircuitHalfOpen
		circuitState = "half-open"
	case "open":
		circuitStyle = StyleCircuitOpen
	default:
		circuitStyle = StyleTextMuted
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf(" %s %s",
		StyleTextMuted.Render("Circuit:"),
		circuitStyle.Render(circuitState)))

	// Rate limit
	if m.preflightCallsRemaining > 0 || m.preflightRateLimitOK {
		var rateStyle lipgloss.Style
		if m.preflightCallsRemaining < 3 {
			rateStyle = StyleWarningMsg
		} else {
			rateStyle = StyleSuccessMsg
		}
		lines = append(lines, fmt.Sprintf(" %s %s",
			StyleTextMuted.Render("Calls:"),
			rateStyle.Render(fmt.Sprintf("%d remaining", m.preflightCallsRemaining))))
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Render(content)
}

// renderNextTasksPanel renders a panel showing the next N remaining tasks
// This is displayed at the start of each loop iteration
func (m Model) renderNextTasksPanel(width int, maxTasks int) string {
	var lines []string

	// Title
	if m.preflightRemainingCount == 0 {
		lines = append(lines, StyleSuccessMsg.Render(" ✓ All tasks complete!"))
	} else {
		lines = append(lines, StyleTextBase.Render(fmt.Sprintf(" Next Tasks (%d remaining)", m.preflightRemainingCount)))
	}
	lines = append(lines, StyleTextMuted.Render(" "+strings.Repeat("─", width-4)))

	// Tasks to show
	tasksToShow := m.preflightRemainingTasks
	if len(tasksToShow) == 0 && len(m.tasks) > 0 {
		// Fallback: use tasks from model if preflight hasn't populated yet
		count := 0
		for _, t := range m.tasks {
			if !t.Completed && count < maxTasks {
				tasksToShow = append(tasksToShow, t.Text)
				count++
			}
		}
	}

	if len(tasksToShow) == 0 && m.preflightRemainingCount > 0 {
		lines = append(lines, "")
		lines = append(lines, StyleTextMuted.Render(" Loading tasks..."))
	} else if len(tasksToShow) > 0 {
		lines = append(lines, "")
		for i, task := range tasksToShow {
			if i >= maxTasks {
				break
			}
			// Truncate task text if too long
			taskText := task
			if len(taskText) > width-8 {
				taskText = taskText[:width-11] + "..."
			}
			// Strip [ ] prefix if present for cleaner display
			taskText = strings.TrimPrefix(taskText, "[ ] ")
			lines = append(lines, fmt.Sprintf(" %s %d. %s",
				StyleTaskPending.Render(IconPending),
				i+1,
				StyleTaskTextPending.Render(taskText)))
		}

		// Show "+X more" if there are more tasks
		remaining := m.preflightRemainingCount - len(tasksToShow)
		if remaining > 0 {
			lines = append(lines, "")
			lines = append(lines, StyleTextSubtle.Render(fmt.Sprintf(" ... and %d more", remaining)))
		}
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Render(content)
}
