package tui

import (
	"bufio"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/brainwhocodes/ralph-codex/internal/codex"
	"github.com/brainwhocodes/ralph-codex/internal/loop"
)

// Program wraps the Bubble Tea program
type Program struct {
	model      Model
	controller *loop.Controller
}

// NewProgram creates a new TUI program
func NewProgram(config codex.Config, controller *loop.Controller) *Program {
	// Load tasks from @fix_plan.md
	tasks := loadTasks()

	// Initialize spinner with dots style
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorAccent)

	// Initialize viewport for logs
	vp := viewport.New(76, 15) // Default size, will be resized
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Padding(0, 1)

	model := Model{
		state:         StateInitializing,
		status:        "Ready to start",
		loopNumber:    0,
		maxCalls:      config.MaxCalls,
		callsUsed:     0,
		circuitState:  "CLOSED",
		logs:          []string{},
		activeView:    "status",
		quitting:      false,
		err:           nil,
		startTime:     time.Now(), // Initialize startTime here!
		width:         80,         // Default width
		height:        24,         // Default height
		tasks:         tasks,
		activity:      "",
		controller:    controller,
		logViewport:   vp,
		taskSpinner:   s,
		activeTaskIdx: -1, // No active task initially
	}

	return &Program{
		model:      model,
		controller: controller,
	}
}

// loadTasks reads tasks from @fix_plan.md
func loadTasks() []Task {
	data, err := os.ReadFile("@fix_plan.md")
	if err != nil {
		return []Task{}
	}

	var tasks []Task
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

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
