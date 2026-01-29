package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// TaskEvidence represents evidence that a task may be completed
type TaskEvidence struct {
	TaskIndex   int
	TaskText    string
	FilesFound  []string // Files that MUST be created by the task (not pre-existing)
	Confidence  float64
	ShouldMark  bool
	Reason      string
}

// SyncResult holds the result of syncing tasks with filesystem
type SyncResult struct {
	PlanFile      string
	TasksUpdated  int
	Evidence      []TaskEvidence
	UpdatedPlan   string
}

// SyncTasksWithFilesystem checks which tasks appear completed based on NEW file creation
// It only marks tasks complete if specific NEW files were created (not pre-existing files)
func SyncTasksWithFilesystem(projectDir string) (*SyncResult, error) {
	// Load the plan file
	tasks, planFile, err := LoadPlanWithFile()
	if err != nil {
		return nil, err
	}

	result := &SyncResult{
		PlanFile: planFile,
		Evidence: make([]TaskEvidence, 0),
	}

	// Read the full plan content
	planContent, err := os.ReadFile(planFile)
	if err != nil {
		return nil, err
	}
	planText := string(planContent)

	// Look for tasks that create NEW files (indicated by "Add" or "Create" + filepath)
	// These are the only ones we can reliably auto-detect
	newFileRegex := regexp.MustCompile("`((?:src|server|lib|tests?|__tests__)/[^`]+\\.(ts|tsx|js|jsx|go|py|css))`")

	for i, task := range tasks {
		// Skip already completed tasks
		if strings.HasPrefix(task, "[x]") {
			continue
		}

		evidence := TaskEvidence{
			TaskIndex:  i,
			TaskText:   task,
			FilesFound: make([]string, 0),
			Confidence: 0,
		}

		taskLower := strings.ToLower(task)

		// Only auto-detect tasks that CREATE new files
		// Keywords: "Add", "Create", "Introduce", "Extract...to"
		isCreationTask := strings.Contains(taskLower, "add ") ||
			strings.Contains(taskLower, "create ") ||
			strings.Contains(taskLower, "introduce ") ||
			strings.Contains(taskLower, "extract") && strings.Contains(taskLower, " to ")

		if !isCreationTask {
			// Skip non-creation tasks - they need manual verification
			continue
		}

		// Find NEW file paths in task text
		matches := newFileRegex.FindAllStringSubmatch(task, -1)
		newFilesFound := 0
		for _, match := range matches {
			if len(match) >= 2 {
				filePath := match[1]
				fullPath := filepath.Join(projectDir, filePath)
				if _, err := os.Stat(fullPath); err == nil {
					evidence.FilesFound = append(evidence.FilesFound, filePath)
					newFilesFound++
				}
			}
		}

		// Special checks for test tooling tasks
		if strings.Contains(taskLower, "test") && (strings.Contains(taskLower, "vitest") || strings.Contains(taskLower, "jest")) {
			// Must have test config file to count as complete
			configFiles := []string{"vitest.config.ts", "vitest.config.js", "jest.config.ts", "jest.config.js"}
			for _, cfg := range configFiles {
				cfgPath := filepath.Join(projectDir, cfg)
				if _, err := os.Stat(cfgPath); err == nil {
					evidence.FilesFound = append(evidence.FilesFound, cfg)
					evidence.Confidence = 0.9
					evidence.ShouldMark = true
					evidence.Reason = "Test config file found"
					break
				}
			}
		} else if newFilesFound > 0 && len(matches) > 0 {
			// For creation tasks, only mark complete if the specific NEW file exists
			evidence.Confidence = float64(newFilesFound) / float64(len(matches))
			// Don't auto-mark based on file existence alone - too error-prone
			// Files may already exist from before the task
			evidence.Reason = fmt.Sprintf("%d/%d files found", newFilesFound, len(matches))
		}

		if len(evidence.FilesFound) > 0 {
			result.Evidence = append(result.Evidence, evidence)
		}
	}

	// Update the plan file if we have evidence
	updatedPlan := planText
	for _, ev := range result.Evidence {
		if ev.ShouldMark {
			// Find the task line and mark it complete
			taskPattern := regexp.MustCompile(`(?m)^(\s*)-\s*\[\s*\]\s*` + regexp.QuoteMeta(strings.TrimPrefix(ev.TaskText, "[ ] ")))
			if taskPattern.MatchString(updatedPlan) {
				updatedPlan = taskPattern.ReplaceAllString(updatedPlan, "$1- [x] "+strings.TrimPrefix(ev.TaskText, "[ ] "))
				result.TasksUpdated++
			}
		}
	}

	result.UpdatedPlan = updatedPlan

	return result, nil
}

// ApplySyncResult writes the updated plan to disk
func ApplySyncResult(result *SyncResult) error {
	if result.TasksUpdated == 0 {
		return nil
	}
	return os.WriteFile(result.PlanFile, []byte(result.UpdatedPlan), 0644)
}

// DetectCompletedTasksByGit uses git diff to detect which tasks may have been completed
func DetectCompletedTasksByGit(projectDir string) ([]string, error) {
	// This could be enhanced to use git diff to see what files changed
	// and match them against task descriptions
	return nil, nil
}
