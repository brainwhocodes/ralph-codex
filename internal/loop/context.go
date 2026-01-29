package loop

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/brainwhocodes/lisa-loop/internal/project"
)

// LoadPlan loads remaining tasks from the plan file based on detected project mode
// Alias for LoadPlanWithFile for backward compatibility
func LoadPlan() ([]string, error) {
	tasks, _, err := LoadPlanWithFile()
	return tasks, err
}

// LoadFixPlan loads remaining tasks from the plan file (deprecated, use LoadPlan)
// Deprecated: Use LoadPlan instead
func LoadFixPlan() ([]string, error) {
	return LoadPlan()
}

// LoadPlanWithFile loads tasks and returns the plan file path
func LoadPlanWithFile() ([]string, string, error) {
	// Detect mode and get the appropriate plan file
	mode := DetectProjectMode()
	planFile := GetPlanFileForMode(mode)

	if planFile == "" {
		// Fallback: try plan files in order of preference
		planFiles := []string{
			"REFACTOR_PLAN.md",
			"IMPLEMENTATION_PLAN.md",
			"@fix_plan.md",
		}

		for _, pf := range planFiles {
			if _, err := os.Stat(pf); err == nil {
				planFile = pf
				break
			}
		}
	}

	if planFile == "" {
		return nil, "", fmt.Errorf("failed to find plan file - need REFACTOR_PLAN.md, IMPLEMENTATION_PLAN.md, or @fix_plan.md")
	}

	data, err := os.ReadFile(planFile)
	if err != nil {
		return nil, planFile, fmt.Errorf("failed to read plan file %s: %w", planFile, err)
	}

	tasks, err := parseTasksFromPlan(string(data), planFile)
	return tasks, planFile, err
}

// parseTasksFromPlan extracts checklist tasks from a plan file
// Supports multiple checklist formats:
//   - "- [ ] task" or "- [x] task" (Markdown)
//   - "* [ ] task" or "* [x] task" (Alternative bullet)
//   - "1. [ ] task" or "1. [x] task" (Numbered)
//   - Indented checklists (nested tasks)
func parseTasksFromPlan(content, filename string) ([]string, error) {
	var tasks []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Match various checklist formats
		checked, taskText, found := extractChecklistItem(trimmed)
		if found {
			// Preserve the checkbox state in the task
			if checked {
				tasks = append(tasks, "[x] "+taskText)
			} else {
				tasks = append(tasks, "[ ] "+taskText)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	// Warn if no tasks found
	if len(tasks) == 0 {
		// Return empty tasks with no error - caller can decide if this is a problem
		return tasks, nil
	}

	return tasks, nil
}

// extractChecklistItem attempts to extract a checklist item from a line
// Returns (isChecked, taskText, found)
func extractChecklistItem(line string) (bool, string, bool) {
	// Pattern: "- [ ] task" or "- [x] task"
	if strings.HasPrefix(line, "- [") && len(line) >= 6 {
		checkbox := line[2:5] // "[ ]" or "[x]" or "[X]"
		if checkbox == "[ ]" || checkbox == "[x]" || checkbox == "[X]" {
			taskText := strings.TrimSpace(line[5:])
			return checkbox != "[ ]", taskText, true
		}
	}

	// Pattern: "* [ ] task" or "* [x] task"
	if strings.HasPrefix(line, "* [") && len(line) >= 6 {
		checkbox := line[2:5]
		if checkbox == "[ ]" || checkbox == "[x]" || checkbox == "[X]" {
			taskText := strings.TrimSpace(line[5:])
			return checkbox != "[ ]", taskText, true
		}
	}

	// Pattern: "1. [ ] task" or "1. [x] task" (numbered lists)
	// Match regex: ^\d+\.\s*\[([ xX])\]\s*(.+)$
	if idx := strings.Index(line, ". ["); idx > 0 && idx < 5 {
		// Check if prefix is a number
		prefix := line[:idx]
		if isNumber(prefix) {
			checkboxStart := idx + 2
			if len(line) >= checkboxStart+3 {
				checkbox := line[checkboxStart : checkboxStart+3]
				if checkbox == "[ ]" || checkbox == "[x]" || checkbox == "[X]" {
					taskText := strings.TrimSpace(line[checkboxStart+3:])
					return checkbox != "[ ]", taskText, true
				}
			}
		}
	}

	// Pattern: "[ ] task" or "[x] task" (bare checkbox, no bullet)
	if strings.HasPrefix(line, "[ ") || strings.HasPrefix(line, "[x") || strings.HasPrefix(line, "[X") {
		if len(line) >= 4 && line[2] == ']' {
			checkbox := line[:3]
			taskText := strings.TrimSpace(line[3:])
			return checkbox != "[ ]", taskText, true
		}
	}

	return false, "", false
}

// isNumber checks if a string consists only of digits
func isNumber(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// ProjectMode is an alias to the unified project mode type
type ProjectMode = project.ProjectMode

// Mode constants delegated to project package
const (
	ModeImplement = project.ProjectModeImplement
	ModeRefactor  = project.ProjectModeRefactor
	ModeFix       = project.ProjectModeFix
	ModeUnknown   = project.ProjectModeUnknown
)

// DetectProjectMode determines the project mode based on files present
// Delegates to the unified project.DetectMode function
func DetectProjectMode() ProjectMode {
	return project.DetectMode(".")
}

// GetPlanFileForMode returns the plan file path for a given mode
// Delegates to the unified project.GetPlanFile function
func GetPlanFileForMode(mode ProjectMode) string {
	return project.GetPlanFile(mode)
}

// GetPromptForMode loads the appropriate prompt based on project mode
func GetPromptForMode(mode ProjectMode) (string, error) {
	switch mode {
	case ModeImplement:
		// Try PRD.md first, fall back to IMPLEMENTATION_PLAN.md
		data, err := os.ReadFile("PRD.md")
		if err != nil {
			// PRD.md is optional, use IMPLEMENTATION_PLAN.md instead
			data, err = os.ReadFile("IMPLEMENTATION_PLAN.md")
			if err != nil {
				return "", fmt.Errorf("failed to read IMPLEMENTATION_PLAN.md: %w", err)
			}
		}
		return string(data), nil

	case ModeRefactor:
		// Use REFACTOR_PLAN.md as context for refactor mode
		data, err := os.ReadFile("REFACTOR_PLAN.md")
		if err != nil {
			return "", fmt.Errorf("failed to read REFACTOR_PLAN.md: %w", err)
		}
		return string(data), nil

	case ModeFix:
		// Use PROMPT.md for fix mode
		data, err := os.ReadFile("PROMPT.md")
		if err != nil {
			return "", fmt.Errorf("failed to read PROMPT.md: %w", err)
		}
		return string(data), nil

	default:
		return "", fmt.Errorf("unknown project mode")
	}
}

// GetPrompt loads the main prompt based on detected project mode
func GetPrompt() (string, error) {
	mode := DetectProjectMode()
	if mode == ModeUnknown {
		return "", fmt.Errorf("could not detect project mode - need PRD.md, REFACTOR_PLAN.md, or PROMPT.md")
	}
	return GetPromptForMode(mode)
}

// BuildContext builds loop context for Codex
func BuildContext(loopNum int, remainingTasks []string, circuitState string, prevSummary string) (string, error) {
	return BuildContextWithPlanFile(loopNum, remainingTasks, circuitState, prevSummary, "")
}

// BuildContextWithPlanFile builds loop context with explicit plan file path
func BuildContextWithPlanFile(loopNum int, remainingTasks []string, circuitState string, prevSummary string, planFile string) (string, error) {
	var ctxBuilder strings.Builder

	ctxBuilder.WriteString("\n--- RALPH LOOP CONTEXT ---\n")
	fmt.Fprintf(&ctxBuilder, "Loop: %d\n", loopNum)
	fmt.Fprintf(&ctxBuilder, "Circuit Breaker: %s\n", circuitState)

	// Determine plan file name for instructions
	if planFile == "" {
		planFile = "REFACTOR_PLAN.md, IMPLEMENTATION_PLAN.md, or @fix_plan.md"
	}

	// CRITICAL instruction at the top
	ctxBuilder.WriteString("\n** CRITICAL: MARK COMPLETED TASKS **\n")
	fmt.Fprintf(&ctxBuilder, "After completing each task, you MUST edit %s to change `- [ ]` to `- [x]`\n", planFile)
	ctxBuilder.WriteString("This is how Lisa tracks progress. Tasks not marked [x] will be repeated!\n")

	if len(remainingTasks) > 0 && len(remainingTasks) <= 5 {
		ctxBuilder.WriteString("\nRemaining Tasks (not yet marked [x]):\n")
		for i, task := range remainingTasks {
			fmt.Fprintf(&ctxBuilder, "  %d. %s\n", i+1, task)
		}
	}

	if prevSummary != "" {
		ctxBuilder.WriteString("\nPrevious Loop Output (for context only, do not respond to this):\n")
		fmt.Fprintf(&ctxBuilder, "```\n%s\n```\n", prevSummary)
	}

	// Add task completion and status reporting reminder
	ctxBuilder.WriteString("\n** WORKFLOW REQUIREMENTS **\n")
	ctxBuilder.WriteString("1. Work on ONE task from the plan\n")
	fmt.Fprintf(&ctxBuilder, "2. After completing the task, EDIT %s to mark it `- [x]`\n", planFile)
	ctxBuilder.WriteString("3. End your response with a RALPH_STATUS block:\n")
	ctxBuilder.WriteString("---RALPH_STATUS---\n")
	ctxBuilder.WriteString("STATUS: WORKING | COMPLETE | BLOCKED\n")
	ctxBuilder.WriteString("CURRENT_TASK: <exact text of task you just completed or are working on>\n")
	ctxBuilder.WriteString("TASKS_COMPLETED_THIS_LOOP: <number>\n")
	ctxBuilder.WriteString("FILES_MODIFIED: <number>\n")
	ctxBuilder.WriteString("TESTS_STATUS: PASSING | FAILING | UNKNOWN\n")
	ctxBuilder.WriteString("EXIT_SIGNAL: true (if ALL tasks [x]) | false (if work remains)\n")
	ctxBuilder.WriteString("---END_RALPH_STATUS---\n")

	ctxBuilder.WriteString("--- END LOOP CONTEXT ---\n\n")

	return ctxBuilder.String(), nil
}

// InjectContext prepends context to prompt
func InjectContext(prompt string, ctx string) string {
	return ctx + prompt
}

// GetProjectRoot returns the project root directory
// Delegates to the unified project.FindProjectRoot function
func GetProjectRoot() (string, error) {
	root, _, err := project.FindProjectRoot("")
	return root, err
}

// CheckProjectRoot verifies we're in a valid Lisa project
// Delegates to the unified project.ValidateProjectDir function
func CheckProjectRoot() error {
	return project.ValidateProjectDir(".")
}
