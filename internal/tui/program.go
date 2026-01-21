package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/brainwhocodes/ralph-codex/internal/codex"
	"github.com/brainwhocodes/ralph-codex/internal/loop"
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

// PlanFileInfo holds info about loaded plan file
type PlanFileInfo struct {
	Filename string
	Tasks    []Task
}

// loadTasksForMode reads tasks from the plan file for the given mode
func loadTasksForMode(mode loop.ProjectMode) PlanFileInfo {
	planFile := loop.GetPlanFileForMode(mode)
	if planFile == "" {
		return PlanFileInfo{
			Filename: "",
			Tasks:    []Task{},
		}
	}

	data, err := os.ReadFile(planFile)
	if err != nil {
		return PlanFileInfo{
			Filename: "",
			Tasks:    []Task{},
		}
	}

	tasks := parseTasksFromData(string(data))
	return PlanFileInfo{
		Filename: planFile,
		Tasks:    tasks,
	}
}

// loadTasks reads tasks - kept for backwards compatibility, tries all plan files
func loadTasks() PlanFileInfo {
	// Try plan files in order of preference
	planFiles := []string{
		"IMPLEMENTATION_PLAN.md",
		"REFACTOR_PLAN.md",
		"@fix_plan.md",
	}

	for _, planFile := range planFiles {
		data, err := os.ReadFile(planFile)
		if err != nil {
			continue
		}

		tasks := parseTasksFromData(string(data))
		return PlanFileInfo{
			Filename: planFile,
			Tasks:    tasks,
		}
	}

	return PlanFileInfo{
		Filename: "",
		Tasks:    []Task{},
	}
}

// parseTasksFromData extracts checklist tasks from plan file content
func parseTasksFromData(data string) []Task {
	var tasks []Task
	scanner := bufio.NewScanner(strings.NewReader(data))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse checkbox items: - [ ] or - [x]
		if strings.HasPrefix(line, "- [") {
			completed := strings.HasPrefix(line, "- [x]") || strings.HasPrefix(line, "- [X]")

			// Extract task text (skip "- [ ] " or "- [x] ")
			text := ""
			if len(line) > 6 {
				text = strings.TrimSpace(line[6:])
			}

			if text != "" {
				tasks = append(tasks, Task{
					Text:      text,
					Completed: completed,
				})
			}
		}
	}

	return tasks
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
