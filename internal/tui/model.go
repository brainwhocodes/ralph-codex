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

// ViewMode represents the current view mode
type ViewMode string

const (
	ViewModeSplit  ViewMode = "split"  // Split pane (tasks + output)
	ViewModeTasks  ViewMode = "tasks"  // Full tasks view
	ViewModeOutput ViewMode = "output" // Full output view
	ViewModeLogs   ViewMode = "logs"   // Full logs view
	ViewModeHelp   ViewMode = "help"   // Help view
	ViewModeCircuit ViewMode = "circuit" // Circuit breaker view
)

// Model represents main TUI model
type Model struct {
	state         State
	status        string
	loopNumber    int
	maxCalls      int
	callsUsed     int
	circuitState  string
	logs          []string
	activeView    string
	viewMode      ViewMode // Current view mode for split/full views
	quitting      bool
	err           error
	helpVisible   bool
	startTime     time.Time
	tick          int    // Animation tick counter
	width         int    // Terminal width
	height        int    // Terminal height
	tasks         []Task           // Tasks from plan file
	planFile      string           // Name of loaded plan file (e.g., REFACTOR_PLAN.md)
	projectMode   loop.ProjectMode // Current project mode (implementation, refactor, fix)
	activity      string           // Current activity description
	controller    *loop.Controller
	ctx           context.Context
	cancel        context.CancelFunc
	activeTaskIdx int // Index of currently active task (-1 if none)

	// Backend and output streaming
	backend        string   // Backend name (cli or opencode)
	outputLines    []string // Live output lines from backend
	reasoningLines []string // Reasoning/thinking output
	currentTool    string   // Current tool being executed

	// Deduplication tracking (SSE sends cumulative updates)
	seenMessages     map[string]bool // Hash of seen message content
	currentReasoning string          // Current reasoning text (replace, don't append)
	currentMessage   string          // Current message text (for cumulative update detection)
	lastToolCall     string          // Last tool call ID to avoid duplicates

	// Analysis results (from RALPH_STATUS block)
	analysisStatus  string  // WORKING, COMPLETE, BLOCKED
	tasksCompleted  int     // Tasks completed this loop
	filesModified   int     // Files modified this loop
	testsStatus     string  // PASSING, FAILING, UNKNOWN
	exitSignal      bool    // Whether exit was signaled
	confidenceScore float64 // Confidence in completion (0-1)

	// Context window tracking
	contextUsagePercent float64 // Current usage (0-1)
	contextTotalTokens  int     // Total tokens used
	contextLimit        int     // Context window limit
	contextThreshold    bool    // True if threshold reached
	contextWasCompacted bool    // True if OpenCode compacted
}

// Init initializes model
func (m Model) Init() tea.Cmd {
	// Start the tick timer for animations
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
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
				switch m.state {
				case StateRunning:
					m.state = StatePaused
					if m.controller != nil {
						m.controller.Pause()
					}
				case StatePaused:
					m.state = StateRunning
					if m.controller != nil {
						m.controller.Resume()
					}
				}
				return m, nil

			case "l":
				// Toggle logs full view
				if m.viewMode == ViewModeLogs {
					m.viewMode = ViewModeSplit
					m.activeView = "status"
				} else {
					m.viewMode = ViewModeLogs
					m.activeView = "logs"
				}
				return m, nil

			case "t":
				// Toggle tasks full view
				if m.viewMode == ViewModeTasks {
					m.viewMode = ViewModeSplit
					m.activeView = "status"
				} else {
					m.viewMode = ViewModeTasks
					m.activeView = "tasks"
				}
				return m, nil

			case "o":
				// Toggle output full view
				if m.viewMode == ViewModeOutput {
					m.viewMode = ViewModeSplit
					m.activeView = "status"
				} else {
					m.viewMode = ViewModeOutput
					m.activeView = "output"
				}
				return m, nil

			case "?":
				m.helpVisible = !m.helpVisible
				if m.helpVisible {
					m.viewMode = ViewModeHelp
					m.activeView = "help"
				} else {
					m.viewMode = ViewModeSplit
					m.activeView = "status"
				}
				return m, nil

			case "c":
				if m.viewMode == ViewModeCircuit {
					m.viewMode = ViewModeSplit
					m.activeView = "status"
				} else {
					m.viewMode = ViewModeCircuit
					m.activeView = "circuit"
				}
				return m, nil

			case "R":
				// Reset circuit breaker - send message to controller
				m.circuitState = "CLOSED"
				m.addLog(string(loop.LogLevelInfo), "Circuit breaker reset")
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
		case loop.EventTypeLoopUpdate:
			m.loopNumber = event.LoopNumber
			m.callsUsed = event.CallsUsed
			m.status = event.Status
			m.circuitState = event.CircuitState
			m.updateActiveTask()
		case loop.EventTypeLog:
			m.addLog(string(event.LogLevel), event.LogMessage)
		case loop.EventTypeStateChange:
			// Handle state changes if needed
		case loop.EventTypeCodexOutput:
			m.addOutputLine(event.OutputLine, string(event.OutputType))
		case loop.EventTypeCodexReasoning:
			m.addReasoningLine(event.ReasoningText)
		case loop.EventTypeCodexTool:
			// Deduplicate tool calls
			toolID := fmt.Sprintf("%s:%s:%s", event.ToolName, event.ToolTarget, event.ToolStatus)
			if toolID == m.lastToolCall {
				return m, nil
			}
			m.lastToolCall = toolID
			m.currentTool = event.ToolName
			if event.ToolStatus == loop.ToolStatusStarted {
				m.addOutputLine(fmt.Sprintf("> %s %s...", event.ToolName, event.ToolTarget), "tool_call")
			} else {
				m.addOutputLine(fmt.Sprintf("  Done: %s", event.ToolTarget), "tool_call")
			}
		case loop.EventTypeAnalysis:
			// Update analysis results from RALPH_STATUS block
			m.analysisStatus = event.AnalysisStatus
			m.tasksCompleted = event.TasksCompleted
			m.filesModified = event.FilesModified
			m.testsStatus = event.TestsStatus
			m.exitSignal = event.ExitSignal
			m.confidenceScore = event.ConfidenceScore
			// Update state if complete
			if m.exitSignal || (m.confidenceScore >= 0.9 && m.analysisStatus == "COMPLETE") {
				m.state = StateComplete
			}
		case loop.EventTypeContextUsage:
			// Update context window usage
			m.contextUsagePercent = event.ContextUsagePercent
			m.contextTotalTokens = event.ContextTotalTokens
			m.contextLimit = event.ContextLimit
			m.contextThreshold = event.ContextThreshold
			m.contextWasCompacted = event.ContextWasCompacted
		}
		return m, nil

	case CodexOutputMsg:
		m.addOutputLine(msg.Line, msg.Type)
		return m, nil

	case CodexReasoningMsg:
		m.addReasoningLine(msg.Text)
		return m, nil

	case CodexToolCallMsg:
		// Deduplicate tool calls
		toolID := fmt.Sprintf("%s:%s:%s", msg.Tool, msg.Target, msg.Status)
		if toolID == m.lastToolCall {
			return m, nil
		}
		m.lastToolCall = toolID
		m.currentTool = msg.Tool
		if msg.Status == "started" {
			m.addOutputLine(fmt.Sprintf("> %s %s...", msg.Tool, msg.Target), "tool_call")
		} else {
			m.addOutputLine(fmt.Sprintf("  Done: %s", msg.Target), "tool_call")
		}
		return m, nil

	case TaskStartedMsg:
		if msg.TaskIndex >= 0 && msg.TaskIndex < len(m.tasks) {
			m.activeTaskIdx = msg.TaskIndex
			m.tasks[msg.TaskIndex].Active = true
			m.addLog(string(loop.LogLevelInfo), fmt.Sprintf("Started: %s", msg.TaskText))
		}
		return m, nil

	case TaskCompletedMsg:
		if msg.TaskIndex >= 0 && msg.TaskIndex < len(m.tasks) {
			m.tasks[msg.TaskIndex].Completed = true
			m.tasks[msg.TaskIndex].Active = false
			m.addLog(string(loop.LogLevelSuccess), fmt.Sprintf("Completed: %s", msg.TaskText))
		}
		return m, nil

	case TaskFailedMsg:
		if msg.TaskIndex >= 0 && msg.TaskIndex < len(m.tasks) {
			m.tasks[msg.TaskIndex].Active = false
			m.addLog(string(loop.LogLevelError), fmt.Sprintf("Failed: %s - %s", msg.TaskText, msg.Error))
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
	formattedLog := StyledLogEntry(level, message)
	m.logs = append(m.logs, formattedLog)
	if len(m.logs) > 500 {
		m.logs = m.logs[len(m.logs)-500:]
	}
}

// addOutputLine adds a line to the output buffer with deduplication
func (m *Model) addOutputLine(line, lineType string) {
	// Initialize map if needed
	if m.seenMessages == nil {
		m.seenMessages = make(map[string]bool)
	}

	// For agent messages, detect cumulative SSE updates and replace instead of append
	// SSE sends: "I'll" → "I'll continue" → "I'll continue fixing" etc.
	if lineType == "agent_message" || lineType == "" {
		// Check if this is a cumulative update (new line starts with or extends current)
		if m.currentMessage != "" {
			// If new line starts with current message, it's a cumulative update - replace
			if strings.HasPrefix(line, m.currentMessage) || strings.HasPrefix(m.currentMessage, line) {
				// Update current message to the longer one
				if len(line) > len(m.currentMessage) {
					m.currentMessage = line
					// Replace the last output line
					if len(m.outputLines) > 0 {
						m.outputLines[len(m.outputLines)-1] = line
					}
				}
				return
			}
		}
		// New message - track it
		m.currentMessage = line
	}

	// Skip exact duplicates
	key := lineType + ":" + line
	if m.seenMessages[key] {
		return
	}
	m.seenMessages[key] = true

	m.outputLines = append(m.outputLines, line)
	// Keep last 200 lines
	if len(m.outputLines) > 200 {
		m.outputLines = m.outputLines[len(m.outputLines)-200:]
	}
}

// addReasoningLine replaces reasoning (SSE sends cumulative text, not deltas)
func (m *Model) addReasoningLine(line string) {
	// Skip if same as current reasoning
	if line == m.currentReasoning {
		return
	}
	m.currentReasoning = line

	// Replace the reasoning lines entirely (cumulative update)
	m.reasoningLines = []string{line}
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
	if err := m.controller.Run(m.ctx); err != nil {
		// Controller errors are already logged via event callbacks
		// Just ensure state is updated
		m.state = StateError
		m.err = err
	}
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

	// Show error view if there's an error
	if m.err != nil && m.viewMode != ViewModeHelp {
		content := m.renderErrorView()
		return m.padToFullScreen(content, width, height)
	}

	// Get content based on view mode
	var content string
	switch m.viewMode {
	case ViewModeHelp:
		content = m.renderHelpView()
	case ViewModeCircuit:
		content = m.renderCircuitView()
	case ViewModeTasks:
		content = m.renderTasksFullView()
	case ViewModeOutput:
		content = m.renderOutputFullView()
	case ViewModeLogs:
		content = m.renderLogsFullView()
	default:
		// Default to split view
		content = m.renderSplitView()
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

	// Apply background color to entire output (Charmtone Pepper)
	result := strings.Join(paddedLines[:height], "\n")
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Background(Pepper).
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

func (m Model) renderErrorView() string {
	width := m.width
	if width < 60 {
		width = 60
	}

	header := m.renderHeader(width)

	// Error content with Crush-style icon
	errorIcon := StyleErrorMsg.Render(IconError)
	errorMsg := fmt.Sprintf("\n %s Error: %v\n", errorIcon, m.err)

	helpText := StyleTextMuted.Render(`
 Press 'r' to retry
 Press 'q' to quit
`)

	footer := StyleFooter.Width(width).Render(
		fmt.Sprintf(" %s retry%s%s quit",
			StyleHelpKey.Render("r"),
			StyleTextSubtle.Render(MetaDotSeparator),
			StyleHelpKey.Render("q")),
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

	const headerHeight = 1
	const footerHeight = 1

	header := m.renderHeader(width)

	// Current state
	circuitState := "closed"
	if m.circuitState != "" {
		circuitState = strings.ToLower(m.circuitState)
	}

	var stateIcon, stateLabel, stateDesc string
	var stateStyle lipgloss.Style

	switch circuitState {
	case "closed":
		stateIcon = IconCheck
		stateLabel = "closed"
		stateStyle = StyleCircuitClosed
		stateDesc = "Circuit is operational. Normal loop execution is allowed."
	case "half_open":
		stateIcon = IconWarning
		stateLabel = "half-open"
		stateStyle = StyleCircuitHalfOpen
		stateDesc = "Circuit is monitoring. Loop may pause if no progress continues."
	case "open":
		stateIcon = IconError
		stateLabel = "open"
		stateStyle = StyleCircuitOpen
		stateDesc = "Circuit is open! Loop execution halted due to repeated failures."
	default:
		stateIcon = IconPending
		stateLabel = circuitState
		stateStyle = StyleTextMuted
		stateDesc = "Unknown circuit state."
	}

	// Circuit info with Crush-style layout
	var lines []string
	lines = append(lines, "")
	lines = append(lines, StyleTextBase.Render(" Circuit Breaker"))
	lines = append(lines, "")
	lines = append(lines, StyleDivider.Render(strings.Repeat(DividerChar, width-4)))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf(" %s %s",
		stateStyle.Render(stateIcon),
		stateStyle.Render(stateLabel)))
	lines = append(lines, "")
	lines = append(lines, StyleTextMuted.Render(" "+stateDesc))
	lines = append(lines, "")
	lines = append(lines, StyleDividerSubtle.Render(strings.Repeat(DividerCharSubtle, width-4)))

	circuitInfo := strings.Join(lines, "\n")

	middleHeight := height - headerHeight - footerHeight - 2
	if middleHeight < 10 {
		middleHeight = 10
	}

	middleContainer := lipgloss.NewStyle().
		Width(width).
		Height(middleHeight).
		Render(circuitInfo)

	// Footer with Crush-style
	footer := StyleFooter.Width(width).Render(
		fmt.Sprintf(" %s return%s%s reset",
			StyleHelpKey.Render("c"),
			StyleTextSubtle.Render(MetaDotSeparator),
			StyleHelpKey.Render("R")),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		middleContainer,
		footer,
	)
}
