package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/brainwhocodes/lisa-loop/internal/codex"
	"github.com/brainwhocodes/lisa-loop/internal/loop"
)

// Program wraps the Bubble Tea program
type Program struct {
	model      Model
	controller *loop.Controller
}

// NewProgram creates a new TUI program
// If explicitMode is provided (non-empty), use it instead of auto-detecting
func NewProgram(config codex.Config, controller *loop.Controller, explicitMode ...loop.ProjectMode) *Program {
	var projectMode loop.ProjectMode
	if len(explicitMode) > 0 && explicitMode[0] != "" {
		projectMode = explicitMode[0]
	} else {
		projectMode = loop.DetectProjectMode()
	}
	planInfo := loadTasksForMode(projectMode)

	// Determine initial state and status based on loaded files
	initialState := StateInitializing
	initialStatus := "Ready to start"
	var initialErr error
	var logs []string

	// Validate project mode
	if projectMode == loop.ModeUnknown {
		initialState = StateError
		initialStatus = "Invalid project - no mode detected"
		initialErr = fmt.Errorf("could not detect project mode")
		logs = append(logs, formatLog("ERROR", "No valid project mode detected. Need PRD.md+IMPLEMENTATION_PLAN.md, REFACTOR_PLAN.md, or PROMPT.md+@fix_plan.md"))
	} else {
		// Try to load prompt for the detected mode
		prompt, err := loop.GetPromptForMode(projectMode)
		if err != nil {
			initialState = StateError
			initialStatus = "Failed to load prompt"
			initialErr = err
			logs = append(logs, formatLog("ERROR", err.Error()))
		} else {
			logs = append(logs, formatLog("INFO", fmt.Sprintf("Loaded %s mode", projectMode)))
			logs = append(logs, formatLog("INFO", fmt.Sprintf("Prompt size: %d bytes", len(prompt))))
		}

		// Log plan file status
		if planInfo.Filename != "" {
			logs = append(logs, formatLog("INFO", fmt.Sprintf("Loaded %d tasks from %s", len(planInfo.Tasks), planInfo.Filename)))
		} else {
			logs = append(logs, formatLog("WARN", "No plan file found"))
		}
	}

	model := Model{
		state:          initialState,
		status:         initialStatus,
		loopNumber:     0,
		maxCalls:       config.MaxCalls,
		callsUsed:      0,
		circuitState:   "CLOSED",
		logs:           logs,
		activeView:     "status",
		viewMode:       ViewModeSplit,
		quitting:       false,
		err:            initialErr,
		startTime:      time.Now(),
		width:          80,
		height:         24,
		tasks:          planInfo.Tasks,
		phases:         planInfo.Phases,
		currentPhase:   findFirstIncompletePhase(planInfo.Phases),
		planFile:       planInfo.Filename,
		projectMode:    projectMode,
		activity:       "",
		controller:     controller,
		activeTaskIdx:  -1,
		backend:        config.Backend,
		outputLines:    []string{},
		reasoningLines: []string{},
	}

	return &Program{
		model:      model,
		controller: controller,
	}
}

// formatLog formats a log entry with timestamp
func formatLog(level, message string) string {
	return fmt.Sprintf("[%s] %s: %s", time.Now().Format("15:04:05"), level, message)
}

// findFirstIncompletePhase returns the index of the first incomplete phase
func findFirstIncompletePhase(phases []Phase) int {
	for i, phase := range phases {
		if !phase.Completed {
			return i
		}
	}
	// All phases complete, return last one
	if len(phases) > 0 {
		return len(phases) - 1
	}
	return 0
}

// PlanFileInfo holds info about loaded plan file
type PlanFileInfo struct {
	Filename string
	Tasks    []Task
	Phases   []Phase
}

// loadTasksForMode reads tasks from the plan file for the given mode
func loadTasksForMode(mode loop.ProjectMode) PlanFileInfo {
	planFile := loop.GetPlanFileForMode(mode)
	if planFile == "" {
		return PlanFileInfo{
			Filename: "",
			Tasks:    []Task{},
			Phases:   []Phase{},
		}
	}

	data, err := os.ReadFile(planFile)
	if err != nil {
		return PlanFileInfo{
			Filename: "",
			Tasks:    []Task{},
			Phases:   []Phase{},
		}
	}

	phases := parsePhasesFromData(string(data))
	tasks := parseTasksFromData(string(data))
	return PlanFileInfo{
		Filename: planFile,
		Tasks:    tasks,
		Phases:   phases,
	}
}

// parseTasksFromData extracts checklist tasks from plan file content
// Returns flat task list for backward compatibility
func parseTasksFromData(data string) []Task {
	phases := parsePhasesFromData(data)
	var tasks []Task
	for _, phase := range phases {
		tasks = append(tasks, phase.Tasks...)
	}
	return tasks
}

// parsePhasesFromData extracts tasks grouped by phase from plan file content
// Supports multiple plan formats:
// - REFACTOR_PLAN.md: ## Phase N: ... headers
// - IMPLEMENTATION_PLAN.md: ## Phase N: ... or ### N) atomic commit headers
// - @fix_plan.md: ## Critical Fixes, ## High Priority, ## Medium Priority, etc.
func parsePhasesFromData(data string) []Phase {
	var phases []Phase
	var currentPhase *Phase
	scanner := bufio.NewScanner(strings.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Detect phase/section headers
		if isPhaseHeader(trimmed) {
			header := extractPhaseHeader(trimmed)
			// Save previous phase if it has tasks
			if currentPhase != nil && len(currentPhase.Tasks) > 0 {
				phases = append(phases, *currentPhase)
			}
			currentPhase = &Phase{
				Name:  header,
				Tasks: []Task{},
			}
			continue
		}

		// Parse checkbox items: - [ ] or - [x]
		if strings.HasPrefix(trimmed, "- [") {
			completed := strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]")

			// Extract task text (skip "- [ ] " or "- [x] ")
			text := ""
			if len(trimmed) > 6 {
				text = strings.TrimSpace(trimmed[6:])
			}

			if text != "" {
				task := Task{
					Text:      text,
					Completed: completed,
				}

				if currentPhase != nil {
					currentPhase.Tasks = append(currentPhase.Tasks, task)
				} else {
					// No phase yet, create a default one
					currentPhase = &Phase{
						Name:  "Tasks",
						Tasks: []Task{task},
					}
				}
			}
		}
	}

	// Don't forget the last phase
	if currentPhase != nil && len(currentPhase.Tasks) > 0 {
		phases = append(phases, *currentPhase)
	}

	// Update phase completion status
	for i := range phases {
		allComplete := true
		for _, task := range phases[i].Tasks {
			if !task.Completed {
				allComplete = false
				break
			}
		}
		phases[i].Completed = allComplete
	}

	return phases
}

// isPhaseHeader checks if a line is a phase/section header
func isPhaseHeader(line string) bool {
	lower := strings.ToLower(line)

	// ## Phase N: ... (REFACTOR_PLAN.md, IMPLEMENTATION_PLAN.md)
	if strings.HasPrefix(line, "## ") {
		header := strings.TrimPrefix(line, "## ")
		headerLower := strings.ToLower(header)

		// Phase headers
		if strings.Contains(headerLower, "phase") {
			return true
		}

		// Fix plan priority headers
		if strings.HasPrefix(headerLower, "critical") ||
			strings.HasPrefix(headerLower, "high priority") ||
			strings.HasPrefix(headerLower, "medium priority") ||
			strings.HasPrefix(headerLower, "low priority") ||
			strings.HasPrefix(headerLower, "testing") ||
			strings.HasPrefix(headerLower, "nice to have") {
			return true
		}

		// Verification/Success criteria sections
		if strings.Contains(headerLower, "verification") ||
			strings.Contains(headerLower, "success criteria") {
			return true
		}
	}

	// ### N) Atomic commit headers (IMPLEMENTATION_PLAN.md)
	if strings.HasPrefix(line, "### ") {
		header := strings.TrimPrefix(line, "### ")
		// Check for numbered headers like "1) Config..." or "2) OpenCode..."
		if len(header) >= 2 && header[0] >= '1' && header[0] <= '9' && header[1] == ')' {
			return true
		}
	}

	// ## Atomic Commits section header
	if strings.HasPrefix(lower, "## atomic") {
		return true
	}

	return false
}

// extractPhaseHeader extracts the display name from a phase header line
func extractPhaseHeader(line string) string {
	if strings.HasPrefix(line, "### ") {
		header := strings.TrimPrefix(line, "### ")
		// For "1) Config..." make it "Step 1: Config..."
		if len(header) >= 2 && header[0] >= '1' && header[0] <= '9' && header[1] == ')' {
			return "Step " + string(header[0]) + ":" + header[2:]
		}
		return header
	}

	if strings.HasPrefix(line, "## ") {
		return strings.TrimPrefix(line, "## ")
	}

	return line
}

// Run starts the TUI program
func (p *Program) Run() error {
	program := tea.NewProgram(
		p.model,
		tea.WithAltScreen(),       // Full-screen alternate buffer mode
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Set up controller event callback to send messages to the TUI
	if p.controller != nil {
		p.controller.SetEventCallback(func(event loop.LoopEvent) {
			program.Send(ControllerEventMsg{Event: event})
		})
	}

	_, err := program.Run()
	return err
}
