package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFixPlan(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a sample fix_plan.md
	fixPlan := `
- [ ] First task
- [ ] Second task
- [x] Completed task
`
	fixPlanPath := filepath.Join(tmpDir, "@fix_plan.md")
	os.WriteFile(fixPlanPath, []byte(fixPlan), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	tasks, err := LoadFixPlan()

	if err != nil {
		t.Errorf("LoadFixPlan() error = %v, want nil", err)
	}

	if len(tasks) != 3 {
		t.Errorf("LoadFixPlan() tasks = %d, want 3", len(tasks))
	}

	// Tasks include checkbox state prefix
	if tasks[0] != "[ ] First task" {
		t.Errorf("LoadFixPlan() tasks[0] = %s, want '[ ] First task'", tasks[0])
	}

	if tasks[1] != "[ ] Second task" {
		t.Errorf("LoadFixPlan() tasks[1] = %s, want '[ ] Second task'", tasks[1])
	}

	if tasks[2] != "[x] Completed task" {
		t.Errorf("LoadFixPlan() tasks[2] = %s, want '[x] Completed task'", tasks[2])
	}
}

func TestBuildContext(t *testing.T) {
	// Test with remaining tasks
	tasks := []string{"Task 1", "Task 2"}
	context, _ := BuildContext(1, tasks, "CLOSED", "Previous summary")

	expectedContains := []string{"Loop: 1", "Circuit Breaker: CLOSED", "Remaining Tasks (not yet marked [x]):", "Previous Loop Output"}

	for _, expected := range expectedContains {
		if !strings.Contains(context, expected) {
			t.Errorf("BuildContext() missing '%s'", expected)
		}
	}

	if !strings.Contains(context, "Task 1") {
		t.Errorf("BuildContext() should include Task 1")
	}

	if !strings.Contains(context, "Task 2") {
		t.Errorf("BuildContext() should include Task 2")
	}
}

func TestBuildContextNoTasks(t *testing.T) {
	// Test with no remaining tasks
	context, _ := BuildContext(1, []string{}, "CLOSED", "Summary")

	if strings.Contains(context, "Remaining Tasks:") {
		t.Errorf("BuildContext() should not show Remaining Tasks when empty")
	}

	if !strings.Contains(context, "Previous Loop Output") {
		t.Errorf("BuildContext() should show previous summary")
	}
}

func TestInjectContext(t *testing.T) {
	prompt := "Main prompt here"
	context := "\n--- RALPH CONTEXT ---\nTest context\n--- END ---"

	injected := InjectContext(prompt, context)

	if !strings.HasPrefix(injected, context) {
		t.Errorf("InjectContext() should prefix with context")
	}

	if !strings.HasSuffix(injected, prompt) {
		t.Errorf("InjectContext() should end with original prompt")
	}
}

func TestGetPrompt(t *testing.T) {
	tmpDir := t.TempDir()

	promptContent := "Test prompt content"
	// Create fix mode files (PROMPT.md + @fix_plan.md)
	promptPath := filepath.Join(tmpDir, "PROMPT.md")
	fixPlanPath := filepath.Join(tmpDir, "@fix_plan.md")
	os.WriteFile(promptPath, []byte(promptContent), 0644)
	os.WriteFile(fixPlanPath, []byte("- [ ] Task 1"), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	prompt, err := GetPrompt()

	if err != nil {
		t.Errorf("GetPrompt() error = %v, want nil", err)
	}

	if prompt != promptContent {
		t.Errorf("GetPrompt() content mismatch")
	}
}

func TestDetectProjectMode(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	t.Run("implementation mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Chdir(tmpDir)
		os.WriteFile("PRD.md", []byte("PRD"), 0644)
		os.WriteFile("IMPLEMENTATION_PLAN.md", []byte("Plan"), 0644)

		mode := DetectProjectMode()
		if mode != ModeImplement {
			t.Errorf("DetectProjectMode() = %v, want %v", mode, ModeImplement)
		}
	})

	t.Run("refactor mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Chdir(tmpDir)
		os.WriteFile("REFACTOR_PLAN.md", []byte("Refactor"), 0644)

		mode := DetectProjectMode()
		if mode != ModeRefactor {
			t.Errorf("DetectProjectMode() = %v, want %v", mode, ModeRefactor)
		}
	})

	t.Run("fix mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Chdir(tmpDir)
		os.WriteFile("PROMPT.md", []byte("Prompt"), 0644)
		os.WriteFile("@fix_plan.md", []byte("Fix"), 0644)

		mode := DetectProjectMode()
		if mode != ModeFix {
			t.Errorf("DetectProjectMode() = %v, want %v", mode, ModeFix)
		}
	})

	t.Run("unknown mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.Chdir(tmpDir)

		mode := DetectProjectMode()
		if mode != ModeUnknown {
			t.Errorf("DetectProjectMode() = %v, want %v", mode, ModeUnknown)
		}
	})
}

func TestGetProjectRoot(t *testing.T) {
	// Save original dir
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Create test directory structure
	tmpDir := t.TempDir()
	testProject := filepath.Join(tmpDir, "test-project")
	os.MkdirAll(testProject, 0755)
	os.Chdir(testProject)

	// Create required files
	os.WriteFile("PROMPT.md", []byte("Test"), 0644)
	os.WriteFile("@fix_plan.md", []byte("- [ ] Task"), 0644)
	os.WriteFile(".git/config", []byte("test"), 0644)
	os.MkdirAll(".git", 0755)

	root, err := GetProjectRoot()

	if err != nil {
		t.Errorf("GetProjectRoot() error = %v, want nil", err)
	}

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	expectedPath, _ := filepath.EvalSymlinks(testProject)
	actualPath, _ := filepath.EvalSymlinks(root)

	if actualPath != expectedPath {
		t.Errorf("GetProjectRoot() = %s, want %s", actualPath, expectedPath)
	}

	os.Chdir(origDir)
}

func TestGetProjectRootNested(t *testing.T) {
	// Save original dir
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Create nested directory
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "level1", "level2", "test-project")
	os.MkdirAll(nestedDir, 0755)
	os.Chdir(nestedDir)

	// Create required files
	os.WriteFile("PROMPT.md", []byte("Test"), 0644)
	os.WriteFile("@fix_plan.md", []byte("- [ ] Task"), 0644)
	os.WriteFile(".git/config", []byte("test"), 0644)
	os.MkdirAll(".git", 0755)

	root, err := GetProjectRoot()

	if err != nil {
		t.Errorf("GetProjectRoot() error = %v, want nil", err)
	}

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	expected := filepath.Join(tmpDir, "level1", "level2", "test-project")
	expectedPath, _ := filepath.EvalSymlinks(expected)
	actualPath, _ := filepath.EvalSymlinks(root)

	if actualPath != expectedPath {
		t.Errorf("GetProjectRoot() nested = %s, want %s", actualPath, expectedPath)
	}

	os.Chdir(origDir)
}

func TestCheckProjectRoot(t *testing.T) {
	// Save original dir
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Test with no required files
	tmpDir := t.TempDir()
	testProject := filepath.Join(tmpDir, "test-project")
	os.MkdirAll(testProject, 0755)
	os.Chdir(testProject)

	err := CheckProjectRoot()

	if err == nil {
		t.Error("CheckProjectRoot() should return error without required files")
	}

	// Create required files for fix mode (PROMPT.md + @fix_plan.md)
	os.WriteFile(filepath.Join(testProject, "PROMPT.md"), []byte("Test"), 0644)
	os.WriteFile(filepath.Join(testProject, "@fix_plan.md"), []byte("- [ ] Task"), 0644)

	err = CheckProjectRoot()

	if err != nil {
		t.Errorf("CheckProjectRoot() should return nil with required files: %v", err)
	}

	os.Chdir(origDir)
}

func TestCheckProjectRootImplementationMode(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	testProject := filepath.Join(tmpDir, "impl-project")
	os.MkdirAll(testProject, 0755)
	os.Chdir(testProject)

	// Create implementation mode files
	os.WriteFile("PRD.md", []byte("# PRD"), 0644)
	os.WriteFile("IMPLEMENTATION_PLAN.md", []byte("- [ ] Task"), 0644)

	err := CheckProjectRoot()
	if err != nil {
		t.Errorf("CheckProjectRoot() should pass for implementation mode: %v", err)
	}
}

func TestCheckProjectRootRefactorMode(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	testProject := filepath.Join(tmpDir, "refactor-project")
	os.MkdirAll(testProject, 0755)
	os.Chdir(testProject)

	// Create refactor mode files
	os.WriteFile("REFACTOR.md", []byte("# Refactor Goals"), 0644)
	os.WriteFile("REFACTOR_PLAN.md", []byte("- [ ] Task"), 0644)

	err := CheckProjectRoot()
	if err != nil {
		t.Errorf("CheckProjectRoot() should pass for refactor mode: %v", err)
	}
}
