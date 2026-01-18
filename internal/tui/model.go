package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/brainwhocodes/ralph-codex/internal/loop"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// State represents TUI state
type State int

const (
	StateInitializing State = iota
	StateRunning
	StatePaused
	StateComplete
	StateError
)

func (s State) String() string {
	switch s {
	case StateInitializing:
		return "Initializing"
	case StateRunning:
		return "Running"
	case StatePaused:
		return "Paused"
	case StateComplete:
		return "Complete"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// LoopUpdateMsg is sent when loop controller updates
type LoopUpdateMsg struct {
	LoopNumber int
	CallsUsed  int
	Status     string
}

// LogMsg is sent to add a log entry
type LogMsg struct {
	Message string
	Level   string // INFO, WARN, ERROR, SUCCESS
}

// StateChangeMsg is sent to change TUI state
type StateChangeMsg struct {
	State State
}

// StatusMsg is sent to update status text
type StatusMsg struct {
	Status string
}

// TickMsg is sent periodically for animations
type TickMsg time.Time

// ControllerEventMsg wraps events from the loop controller
type ControllerEventMsg struct {
	Event loop.LoopEvent
}

// Task represents a task from @fix_plan.md
type Task struct {
	Text      string
	Completed bool
	Active    bool // Currently being worked on
}

// Model represents main TUI model
type Model struct {
	state        State
	status       string
	loopNumber   int
	maxCalls     int
	callsUsed    int
	circuitState string
	logs         []string
	activeView   string
	quitting     bool
	err          error
	helpVisible  bool
	startTime    time.Time
	tick         int      // Animation tick counter
	width        int      // Terminal width
	height       int      // Terminal height
	tasks        []Task   // Tasks from @fix_plan.md
	activity     string   // Current activity description
	controller    *loop.Controller
	ctx           context.Context
	cancel        context.CancelFunc
	activeTaskIdx int // Index of currently active task (-1 if none)
}

// Init initializes model
func (m Model) Init() tea.Cmd {
	// Start the tick timer for animations
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// styledLogEntry returns a styled log entry with emoji prefix (no background styling)
func styledLogEntry(level string, message string) string {
	switch level {
	case "INFO":
		return StyleHelpDesc.Render("ℹ️  " + message)
	case "WARN":
		return StyleCircuitHalfOpen.Render("⚠️  " + message)
	case "ERROR":
		return StyleErrorMsg.Render("❌ " + message)
	case "SUCCESS":
		return StyleCircuitClosed.Render("✅ " + message)
	default:
		return StyleHelpDesc.Render("   " + message)
	}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlQ:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyRunes:
			switch msg.String() {
			case "q":
				m.quitting = true
				return m, tea.Quit

			case "r":
				if m.state != StateRunning && m.controller != nil {
					m.state = StateRunning
					m.activeTaskIdx = 0 // Start with first task
					m.ctx, m.cancel = context.WithCancel(context.Background())
					go m.runController()
					return m, nil
				}

			case "p":
				if m.controller != nil {
					if m.state == StateRunning {
						m.state = StatePaused
						m.controller.Pause()
					} else if m.state == StatePaused {
						m.state = StateRunning
						m.controller.Resume()
					}
				}
				return m, nil

			case "?":
				m.helpVisible = !m.helpVisible
				if m.helpVisible {
					m.activeView = "help"
				} else {
					m.activeView = "status"
				}
				return m, nil

			case "c":
				if m.activeView == "status" {
					m.activeView = "circuit"
				} else {
					m.activeView = "status"
				}
				return m, nil

			case "R":
				// Reset circuit breaker - send message to controller
				m.circuitState = "CLOSED"
				m.addLog("INFO", "Circuit breaker reset")
				return m, nil

			}
		}

	case LoopUpdateMsg:
		m.loopNumber = msg.LoopNumber
		m.callsUsed = msg.CallsUsed
		m.status = msg.Status
		m.updateActiveTask()
		return m, nil

	case LogMsg:
		m.addLog(msg.Level, msg.Message)
		return m, nil

	case StateChangeMsg:
		m.state = msg.State
		if msg.State == StateComplete {
			m.activeTaskIdx = -1
		}
		return m, nil

	case StatusMsg:
		m.status = msg.Status
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case TickMsg:
		// Increment tick for animations
		m.tick++
		return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})

	case ControllerEventMsg:
		event := msg.Event
		switch event.Type {
		case "loop_update":
			m.loopNumber = event.LoopNumber
			m.callsUsed = event.CallsUsed
			m.status = event.Status
			m.circuitState = event.CircuitState
			m.updateActiveTask()
		case "log":
			m.addLog(event.LogLevel, event.LogMessage)
		case "state_change":
			// Handle state changes if needed
		}
		return m, nil
	}

	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

// addLog adds a log entry
func (m *Model) addLog(level, message string) {
	formattedLog := styledLogEntry(level, message)
	m.logs = append(m.logs, formattedLog)
	if len(m.logs) > 500 {
		m.logs = m.logs[len(m.logs)-500:]
	}
}

// updateActiveTask updates the active task based on loop progress
func (m *Model) updateActiveTask() {
	if m.state != StateRunning {
		return
	}
	// Simple heuristic: advance to next incomplete task when loop progresses
	for i := range m.tasks {
		if !m.tasks[i].Completed {
			m.activeTaskIdx = i
			m.tasks[i].Active = true
			// Deactivate previous tasks
			for j := 0; j < i; j++ {
				m.tasks[j].Active = false
			}
			break
		}
	}
}

func (m *Model) runController() {
	if m.controller == nil {
		return
	}
	m.controller.Run(m.ctx)
}

// View renders TUI - fills entire terminal
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	width := m.width
	height := m.height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}

	// Get content based on active view
	var content string
	switch m.activeView {
	case "help":
		content = m.renderHelpView()
	case "circuit":
		content = m.renderCircuitView()
	default:
		content = m.renderStatusView()
	}

	// Show error view if there's an error
	if m.err != nil && m.activeView != "help" {
		content = m.renderErrorView()
	}

	// Pad content to fill entire screen
	return m.padToFullScreen(content, width, height)
}

// padToFullScreen pads content to fill the entire terminal
func (m Model) padToFullScreen(content string, width, height int) string {
	lines := strings.Split(content, "\n")

	// Pad each line to full width
	var paddedLines []string
	for _, line := range lines {
		lineLen := lipgloss.Width(line)
		if lineLen < width {
			line = line + strings.Repeat(" ", width-lineLen)
		}
		paddedLines = append(paddedLines, line)
	}

	// Add empty lines to fill height
	for len(paddedLines) < height {
		paddedLines = append(paddedLines, strings.Repeat(" ", width))
	}

	// Apply background color to entire output
	result := strings.Join(paddedLines[:height], "\n")
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Background(BgPrimary).
		Render(result)
}

func (m Model) renderRateLimitProgress() string {
	if m.maxCalls == 0 {
		return "Calls: 0/0"
	}

	total := m.maxCalls
	progress := float64(m.callsUsed) / float64(total)
	if progress > 1.0 {
		progress = 1.0
	}

	width := 20
	filled := int(progress * float64(width))

	emptyWidth := width - filled
	if emptyWidth < 0 {
		emptyWidth = 0
	}

	// Use simple colored characters without background width issues
	filledBar := StyleProgressFilled.Render(strings.Repeat("█", filled))
	emptyBar := StyleProgressEmpty.Render(strings.Repeat("░", emptyWidth))

	bar := fmt.Sprintf("Calls: %d/%d [%s%s]",
		m.callsUsed, total,
		filledBar,
		emptyBar,
	)

	return bar
}

func (m Model) renderStatusView() string {
	width := m.width
	if width < 60 {
		width = 60
	}

	var sections []string

	// Header with title
	header := StyleHeader.Width(width).Render("  Ralph Codex")
	sections = append(sections, header)

	// Status bar with state, loop, calls, circuit
	circuitState := m.circuitState
	if circuitState == "" {
		circuitState = "CLOSED"
	}

	var circuitBadge string
	switch circuitState {
	case "CLOSED":
		circuitBadge = StyleCircuitClosed.Render(circuitState)
	case "HALF_OPEN":
		circuitBadge = StyleCircuitHalfOpen.Render(circuitState)
	case "OPEN":
		circuitBadge = StyleCircuitOpen.Render(circuitState)
	default:
		circuitBadge = circuitState
	}

	var stateBadge string
	switch m.state {
	case StateInitializing:
		stateBadge = StyleStatusInitializing.Render(" INIT ")
	case StateRunning:
		stateBadge = StyleStatusRunning.Render(" RUNNING ")
	case StatePaused:
		stateBadge = StyleStatusPaused.Render(" PAUSED ")
	case StateComplete:
		stateBadge = StyleStatusComplete.Render(" COMPLETE ")
	case StateError:
		stateBadge = StyleStatusError.Render(" ERROR ")
	}

	// Simple spinner chars (avoid bubbles spinner for now)
	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerStr := " "
	if m.state == StateRunning {
		spinnerStr = StyleSpinner.Render(spinnerFrames[m.tick%len(spinnerFrames)])
	}

	progressBar := m.renderRateLimitProgress()
	elapsed := time.Since(m.startTime).Round(time.Second)

	statusLine := StyleStatus.Width(width).Render(fmt.Sprintf(
		" %s %s Loop: %d  %s  Circuit: %s  Elapsed: %s",
		stateBadge, spinnerStr, m.loopNumber, progressBar, circuitBadge, elapsed))
	sections = append(sections, statusLine)

	// Spacer
	sections = append(sections, "")

	// Tasks section
	taskSection := m.renderTaskSection(width - 4)
	sections = append(sections, taskSection)

	// Spacer
	sections = append(sections, "")

	// Live log feed
	logSection := m.renderLiveLogFeed(width - 4)
	sections = append(sections, logSection)

	// Spacer
	sections = append(sections, "")

	// Footer
	footer := StyleStatus.Width(width).Render(fmt.Sprintf(
		" %s Run  %s Pause  %s Circuit  %s Help  %s Quit",
		StyleHelpKey.Render("r"),
		StyleHelpKey.Render("p"),
		StyleHelpKey.Render("c"),
		StyleHelpKey.Render("?"),
		StyleHelpKey.Render("q"),
	))
	sections = append(sections, footer)

	return strings.Join(sections, "\n")
}

// renderLiveLogFeed shows the last few log entries in the main view
func (m Model) renderLiveLogFeed(width int) string {
	var lines []string
	lines = append(lines, StyleInfoMsg.Render("  Activity Log:"))

	if len(m.logs) == 0 {
		lines = append(lines, StyleHelpDesc.Render("    Waiting for activity..."))
		return strings.Join(lines, "\n")
	}

	// Show last 6 log entries
	start := len(m.logs) - 6
	if start < 0 {
		start = 0
	}

	for i := start; i < len(m.logs); i++ {
		log := m.logs[i]
		lines = append(lines, "  "+log)
	}

	// Show count if more logs exist
	if len(m.logs) > 6 {
		lines = append(lines, StyleHelpDesc.Render(fmt.Sprintf("    ... %d total entries", len(m.logs))))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderTaskSection(width int) string {
	if len(m.tasks) == 0 {
		return StyleHelpDesc.Render("  No tasks loaded. Check @fix_plan.md")
	}

	// Count completed tasks
	completed := 0
	for _, t := range m.tasks {
		if t.Completed {
			completed++
		}
	}

	var lines []string

	// Task progress header
	progressPct := (completed * 100) / len(m.tasks)
	lines = append(lines, StyleInfoMsg.Render(fmt.Sprintf("  Tasks: %d/%d (%d%%)", completed, len(m.tasks), progressPct)))

	// Progress bar
	barWidth := 30
	filledCount := (completed * barWidth) / len(m.tasks)
	if filledCount > barWidth {
		filledCount = barWidth
	}
	emptyCount := barWidth - filledCount
	taskProgressBar := "  " + StyleProgressFilled.Render(strings.Repeat("█", filledCount)) +
		StyleProgressEmpty.Render(strings.Repeat("░", emptyCount))
	lines = append(lines, taskProgressBar)
	lines = append(lines, "")

	// Spinner frames for active task
	spinnerFrames := []string{">", ">>", ">>>", ">>", ">"}

	shown := 0
	for i, task := range m.tasks {
		if shown >= 6 {
			remaining := len(m.tasks) - shown
			if remaining > 0 {
				lines = append(lines, StyleHelpDesc.Render(fmt.Sprintf("       ... and %d more tasks", remaining)))
			}
			break
		}

		var line string
		if task.Completed {
			line = StyleCircuitClosed.Render(fmt.Sprintf("  [x] %s", task.Text))
		} else if i == m.activeTaskIdx && m.state == StateRunning {
			// Show animated indicator for active task
			indicator := spinnerFrames[m.tick%len(spinnerFrames)]
			line = StyleInfoMsg.Render(fmt.Sprintf("  %s [ ] %s", indicator, task.Text))
		} else {
			line = StyleHelpDesc.Render(fmt.Sprintf("  [ ] %s", task.Text))
		}

		// Truncate if needed (after styling to preserve content)
		if width > 10 && len(task.Text) > width-10 {
			if task.Completed {
				line = StyleCircuitClosed.Render(fmt.Sprintf("  [x] %s...", task.Text[:width-15]))
			} else if i == m.activeTaskIdx && m.state == StateRunning {
				indicator := spinnerFrames[m.tick%len(spinnerFrames)]
				line = StyleInfoMsg.Render(fmt.Sprintf("  %s [ ] %s...", indicator, task.Text[:width-18]))
			} else {
				line = StyleHelpDesc.Render(fmt.Sprintf("  [ ] %s...", task.Text[:width-15]))
			}
		}

		lines = append(lines, line)
		shown++
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderErrorView() string {
	width := m.width
	if width < 60 {
		width = 60
	}

	header := StyleHeader.Copy().Width(width).Render("Ralph Codex - Error")

	errorMsg := StyleErrorMsg.Render(fmt.Sprintf("\nError: %v\n", m.err))

	helpText := StyleHelpDesc.Render(`
Press 'r' to retry
Press 'q' to quit
`)

	footer := StyleStatus.Copy().Width(width).Render(
		fmt.Sprintf(" %s Retry  %s Quit", StyleHelpKey.Render("r"), StyleHelpKey.Render("q")),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		errorMsg,
		helpText,
		"",
		footer,
	)
}

func (m Model) renderCircuitView() string {
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

	header := StyleHeader.Copy().Width(width).Render("Circuit Breaker Status")

	// Current state badge
	circuitState := "CLOSED"
	if m.circuitState != "" {
		circuitState = m.circuitState
	}

	var stateBadge string
	var stateDesc string

	switch circuitState {
	case "CLOSED":
		stateBadge = StyleCircuitClosed.Render(circuitState)
		stateDesc = "Circuit is operational. Normal loop execution is allowed."
	case "HALF_OPEN":
		stateBadge = StyleCircuitHalfOpen.Render(circuitState)
		stateDesc = "Circuit is monitoring. Loop may be paused if no progress continues."
	case "OPEN":
		stateBadge = StyleCircuitOpen.Render(circuitState)
		stateDesc = "Circuit is open! Loop execution is halted due to repeated failures."
	default:
		stateBadge = circuitState
		stateDesc = "Unknown circuit state."
	}

	// Circuit info
	circuitInfo := fmt.Sprintf(`
%s

%s
%s

State Explanation:
  %s
`,
		StyleInfoMsg.Render("Current State"),
		stateBadge,
		StyleDivider.Copy().Width(width-4).Render(strings.Repeat("─", width-4)),
		StyleHelpDesc.Render(stateDesc))

	middleHeight := height - headerHeight - footerHeight - 2
	if middleHeight < 10 {
		middleHeight = 10
	}

	middleContainer := lipgloss.NewStyle().
		Width(width).
		Height(middleHeight).
		Render(circuitInfo)

	// Footer
	footer := StyleStatus.Copy().Width(width).Render(
		fmt.Sprintf(" %s Return to status  %s Reset circuit breaker",
			StyleHelpKey.Render("l"),
			StyleHelpKey.Render("R")),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		middleContainer,
		footer,
	)
}
