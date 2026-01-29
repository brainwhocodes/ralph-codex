package tui

import (
	"os"
	"testing"

	"github.com/brainwhocodes/lisa-loop/internal/codex"
	"github.com/brainwhocodes/lisa-loop/internal/loop"
)

func TestNewProgram_ExplicitMode(t *testing.T) {
	// Save original dir
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	t.Run("explicit refactor mode ignores implementation files", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Chdir(tmpDir)

		// Create files for BOTH implementation and refactor mode
		os.WriteFile("PRD.md", []byte("PRD content"), 0644)
		os.WriteFile("IMPLEMENTATION_PLAN.md", []byte("- [ ] Impl task 1\n- [ ] Impl task 2"), 0644)
		os.WriteFile("REFACTOR_PLAN.md", []byte("- [ ] Refactor task 1\n- [ ] Refactor task 2\n- [ ] Refactor task 3"), 0644)

		config := codex.Config{MaxCalls: 3}
		// Explicitly set refactor mode
		program := NewProgram(config, nil, loop.ModeRefactor)

		if program.model.projectMode != loop.ModeRefactor {
			t.Errorf("Expected mode %s, got %s", loop.ModeRefactor, program.model.projectMode)
		}

		if program.model.planFile != "REFACTOR_PLAN.md" {
			t.Errorf("Expected planFile REFACTOR_PLAN.md, got %s", program.model.planFile)
		}

		// Refactor plan has 3 tasks
		if len(program.model.tasks) != 3 {
			t.Errorf("Expected 3 tasks from REFACTOR_PLAN.md, got %d", len(program.model.tasks))
		}
	})

	t.Run("explicit implement mode ignores refactor files", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Chdir(tmpDir)

		// Create files for BOTH modes
		os.WriteFile("PRD.md", []byte("PRD content"), 0644)
		os.WriteFile("IMPLEMENTATION_PLAN.md", []byte("- [ ] Impl task 1\n- [ ] Impl task 2"), 0644)
		os.WriteFile("REFACTOR_PLAN.md", []byte("- [ ] Refactor task 1\n- [ ] Refactor task 2\n- [ ] Refactor task 3"), 0644)

		config := codex.Config{MaxCalls: 3}
		// Explicitly set implement mode
		program := NewProgram(config, nil, loop.ModeImplement)

		if program.model.projectMode != loop.ModeImplement {
			t.Errorf("Expected mode %s, got %s", loop.ModeImplement, program.model.projectMode)
		}

		if program.model.planFile != "IMPLEMENTATION_PLAN.md" {
			t.Errorf("Expected planFile IMPLEMENTATION_PLAN.md, got %s", program.model.planFile)
		}

		// Implementation plan has 2 tasks
		if len(program.model.tasks) != 2 {
			t.Errorf("Expected 2 tasks from IMPLEMENTATION_PLAN.md, got %d", len(program.model.tasks))
		}
	})

	t.Run("explicit fix mode loads fix plan", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Chdir(tmpDir)

		// Create files for multiple modes
		os.WriteFile("PRD.md", []byte("PRD content"), 0644)
		os.WriteFile("IMPLEMENTATION_PLAN.md", []byte("- [ ] Impl task 1"), 0644)
		os.WriteFile("PROMPT.md", []byte("Fix prompt"), 0644)
		os.WriteFile("@fix_plan.md", []byte("- [ ] Fix task 1\n- [ ] Fix task 2\n- [ ] Fix task 3\n- [ ] Fix task 4"), 0644)

		config := codex.Config{MaxCalls: 3}
		// Explicitly set fix mode
		program := NewProgram(config, nil, loop.ModeFix)

		if program.model.projectMode != loop.ModeFix {
			t.Errorf("Expected mode %s, got %s", loop.ModeFix, program.model.projectMode)
		}

		if program.model.planFile != "@fix_plan.md" {
			t.Errorf("Expected planFile @fix_plan.md, got %s", program.model.planFile)
		}

		// Fix plan has 4 tasks
		if len(program.model.tasks) != 4 {
			t.Errorf("Expected 4 tasks from @fix_plan.md, got %d", len(program.model.tasks))
		}
	})

	t.Run("no explicit mode auto-detects", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Chdir(tmpDir)

		// Only create refactor mode files
		os.WriteFile("REFACTOR_PLAN.md", []byte("- [ ] Refactor task 1"), 0644)

		config := codex.Config{MaxCalls: 3}
		// No explicit mode - should auto-detect
		program := NewProgram(config, nil)

		if program.model.projectMode != loop.ModeRefactor {
			t.Errorf("Expected auto-detected mode %s, got %s", loop.ModeRefactor, program.model.projectMode)
		}

		if program.model.planFile != "REFACTOR_PLAN.md" {
			t.Errorf("Expected planFile REFACTOR_PLAN.md, got %s", program.model.planFile)
		}
	})
}

func TestLoadTasksForMode(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	t.Run("loads correct file for each mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Chdir(tmpDir)

		// Create all plan files with different task counts
		os.WriteFile("IMPLEMENTATION_PLAN.md", []byte("- [ ] Task A"), 0644)
		os.WriteFile("REFACTOR_PLAN.md", []byte("- [ ] Task B\n- [ ] Task C"), 0644)
		os.WriteFile("@fix_plan.md", []byte("- [ ] Task D\n- [ ] Task E\n- [ ] Task F"), 0644)

		// Test implement mode
		implInfo := loadTasksForMode(loop.ModeImplement)
		if implInfo.Filename != "IMPLEMENTATION_PLAN.md" {
			t.Errorf("Implement mode: expected IMPLEMENTATION_PLAN.md, got %s", implInfo.Filename)
		}
		if len(implInfo.Tasks) != 1 {
			t.Errorf("Implement mode: expected 1 task, got %d", len(implInfo.Tasks))
		}

		// Test refactor mode
		refactorInfo := loadTasksForMode(loop.ModeRefactor)
		if refactorInfo.Filename != "REFACTOR_PLAN.md" {
			t.Errorf("Refactor mode: expected REFACTOR_PLAN.md, got %s", refactorInfo.Filename)
		}
		if len(refactorInfo.Tasks) != 2 {
			t.Errorf("Refactor mode: expected 2 tasks, got %d", len(refactorInfo.Tasks))
		}

		// Test fix mode
		fixInfo := loadTasksForMode(loop.ModeFix)
		if fixInfo.Filename != "@fix_plan.md" {
			t.Errorf("Fix mode: expected @fix_plan.md, got %s", fixInfo.Filename)
		}
		if len(fixInfo.Tasks) != 3 {
			t.Errorf("Fix mode: expected 3 tasks, got %d", len(fixInfo.Tasks))
		}
	})
}

func TestGetPlanFileForMode(t *testing.T) {
	tests := []struct {
		mode     loop.ProjectMode
		expected string
	}{
		{loop.ModeImplement, "IMPLEMENTATION_PLAN.md"},
		{loop.ModeRefactor, "REFACTOR_PLAN.md"},
		{loop.ModeFix, "@fix_plan.md"},
		{loop.ModeUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			result := loop.GetPlanFileForMode(tt.mode)
			if result != tt.expected {
				t.Errorf("GetPlanFileForMode(%s) = %s, want %s", tt.mode, result, tt.expected)
			}
		})
	}
}
