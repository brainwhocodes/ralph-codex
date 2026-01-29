package tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainwhocodes/lisa-loop/internal/circuit"
	"github.com/brainwhocodes/lisa-loop/internal/config"
	"github.com/brainwhocodes/lisa-loop/internal/loop"
	"github.com/brainwhocodes/lisa-loop/internal/runner"
)

// fakeRunner is a test double that simulates task completion
// It marks one task as complete on each call
type fakeRunner struct {
	projectDir   string
	planFile     string
	callCount    int
	maxCalls     int
}

func (f *fakeRunner) Run(prompt string) (string, string, error) {
	f.callCount++

	if f.callCount > f.maxCalls {
		return "", "", nil
	}

	// Mark one task as complete
	if err := f.completeOneTask(); err != nil {
		return "", "", err
	}

	return "Simulated response - tasks completed", "fake-session-" + string(rune('0'+f.callCount)), nil
}

func (f *fakeRunner) Stop() error {
	return nil
}

func (f *fakeRunner) SetOutputCallback(cb runner.OutputCallback) {}

func (f *fakeRunner) completeOneTask() error {
	planPath := filepath.Join(f.projectDir, f.planFile)
	data, err := os.ReadFile(planPath)
	if err != nil {
		return err
	}

	content := string(data)
	// Find first unchecked task and mark it complete
	if strings.Contains(content, "- [ ]") {
		content = strings.Replace(content, "- [ ]", "- [x]", 1)
		return os.WriteFile(planPath, []byte(content), 0644)
	}

	return nil
}

// setupTestProject creates a temp project with fixtures
type testProject struct {
	Dir      string
	Mode     string
	PlanFile string
}

func setupTestProject(t *testing.T, mode string) *testProject {
	t.Helper()

	tmpDir := t.TempDir()
	fixtureDir := filepath.Join("fixtures", mode)

	// Copy fixture files
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("Failed to read fixtures: %v", err)
	}

	planFile := ""
	for _, entry := range entries {
		src := filepath.Join(fixtureDir, entry.Name())
		dst := filepath.Join(tmpDir, entry.Name())

		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("Failed to read fixture %s: %v", src, err)
		}

		if err := os.WriteFile(dst, data, 0644); err != nil {
			t.Fatalf("Failed to write fixture %s: %v", dst, err)
		}

		// Track which file is the plan file
		if entry.Name() == "@fix_plan.md" || entry.Name() == "IMPLEMENTATION_PLAN.md" || entry.Name() == "REFACTOR_PLAN.md" {
			planFile = entry.Name()
		}
	}

	return &testProject{
		Dir:      tmpDir,
		Mode:     mode,
		PlanFile: planFile,
	}
}

// countRemainingTasks counts unchecked tasks in plan file
func countRemainingTasks(projectDir, planFile string) int {
	data, err := os.ReadFile(filepath.Join(projectDir, planFile))
	if err != nil {
		return -1
	}
	return strings.Count(string(data), "- [ ]")
}

// countCompletedTasks counts checked tasks in plan file
func countCompletedTasks(projectDir, planFile string) int {
	data, err := os.ReadFile(filepath.Join(projectDir, planFile))
	if err != nil {
		return -1
	}
	return strings.Count(string(data), "- [x]")
}

func TestE2E_FixMode_RunLoop(t *testing.T) {
	project := setupTestProject(t, "fix")

	// Change to project directory
	origDir, _ := os.Getwd()
	os.Chdir(project.Dir)
	defer os.Chdir(origDir)

	// Create controller
	cfg := config.Config{
		MaxCalls: 10,
		Backend:  "test",
		Timeout:  60,
	}

	rateLimiter := loop.NewRateLimiter(100, 1)
	breaker := circuit.NewBreaker(3, 5)
	controller := loop.NewController(cfg, rateLimiter, breaker)

	// Inject fake runner that marks tasks complete
	fake := &fakeRunner{
		projectDir: project.Dir,
		planFile:   project.PlanFile,
		maxCalls:   5, // More than enough for 3 tasks
	}
	controller.SetRunner(fake)

	// Collect events
	var preflightEvents []*loop.PreflightSummary
	var outcomeEvents []*loop.LoopOutcome
	controller.SetEventCallback(func(event loop.LoopEvent) {
		if event.Preflight != nil {
			preflightEvents = append(preflightEvents, event.Preflight)
		}
		if event.Outcome != nil {
			outcomeEvents = append(outcomeEvents, event.Outcome)
		}
	})

	// Run the loop with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := controller.Run(ctx)

	// Verify loop completed successfully
	if err != nil {
		t.Errorf("Loop failed: %v", err)
	}

	// Verify all tasks are now complete
	remaining := countRemainingTasks(project.Dir, project.PlanFile)
	completed := countCompletedTasks(project.Dir, project.PlanFile)

	if remaining != 0 {
		t.Errorf("Expected 0 remaining tasks, got %d", remaining)
	}
	if completed != 3 {
		t.Errorf("Expected 3 completed tasks, got %d", completed)
	}

	// Verify events were emitted
	if len(preflightEvents) == 0 {
		t.Error("Expected preflight events to be emitted")
	}
	if len(outcomeEvents) == 0 {
		t.Error("Expected outcome events to be emitted")
	}

	// Verify fake runner was called
	if fake.callCount == 0 {
		t.Error("Expected fake runner to be called")
	}

	t.Logf("Loop ran %d iterations", fake.callCount)
	t.Logf("Captured %d preflight events, %d outcome events", len(preflightEvents), len(outcomeEvents))
}

func TestE2E_ImplementMode_RunLoop(t *testing.T) {
	project := setupTestProject(t, "implement")

	// Change to project directory
	origDir, _ := os.Getwd()
	os.Chdir(project.Dir)
	defer os.Chdir(origDir)

	cfg := config.Config{
		MaxCalls: 10,
		Backend:  "test",
		Timeout:  60,
	}

	rateLimiter := loop.NewRateLimiter(100, 1)
	breaker := circuit.NewBreaker(3, 5)
	controller := loop.NewController(cfg, rateLimiter, breaker)

	fake := &fakeRunner{
		projectDir: project.Dir,
		planFile:   project.PlanFile,
		maxCalls:   10,
	}
	controller.SetRunner(fake)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := controller.Run(ctx)

	if err != nil {
		t.Errorf("Loop failed: %v", err)
	}

	remaining := countRemainingTasks(project.Dir, project.PlanFile)
	completed := countCompletedTasks(project.Dir, project.PlanFile)

	if remaining != 0 {
		t.Errorf("Expected 0 remaining tasks, got %d", remaining)
	}
	if completed != 6 {
		t.Errorf("Expected 6 completed tasks, got %d", completed)
	}

	t.Logf("Loop ran %d iterations to complete 6 tasks", fake.callCount)
}

func TestE2E_RefactorMode_RunLoop(t *testing.T) {
	project := setupTestProject(t, "refactor")

	// Change to project directory
	origDir, _ := os.Getwd()
	os.Chdir(project.Dir)
	defer os.Chdir(origDir)

	cfg := config.Config{
		MaxCalls: 10,
		Backend:  "test",
		Timeout:  60,
	}

	rateLimiter := loop.NewRateLimiter(100, 1)
	breaker := circuit.NewBreaker(3, 5)
	controller := loop.NewController(cfg, rateLimiter, breaker)

	fake := &fakeRunner{
		projectDir: project.Dir,
		planFile:   project.PlanFile,
		maxCalls:   10,
	}
	controller.SetRunner(fake)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := controller.Run(ctx)

	if err != nil {
		t.Errorf("Loop failed: %v", err)
	}

	remaining := countRemainingTasks(project.Dir, project.PlanFile)
	completed := countCompletedTasks(project.Dir, project.PlanFile)

	if remaining != 0 {
		t.Errorf("Expected 0 remaining tasks, got %d", remaining)
	}
	if completed != 4 {
		t.Errorf("Expected 4 completed tasks, got %d", completed)
	}

	t.Logf("Loop ran %d iterations to complete 4 tasks", fake.callCount)
}

func TestE2E_LoopExitsEarly_WhenAllTasksComplete(t *testing.T) {
	project := setupTestProject(t, "fix")

	// Mark all tasks as complete initially
	planPath := filepath.Join(project.Dir, project.PlanFile)
	content, _ := os.ReadFile(planPath)
	completed := strings.ReplaceAll(string(content), "- [ ]", "- [x]")
	os.WriteFile(planPath, []byte(completed), 0644)

	// Change to project directory
	origDir, _ := os.Getwd()
	os.Chdir(project.Dir)
	defer os.Chdir(origDir)

	cfg := config.Config{
		MaxCalls: 10,
		Backend:  "test",
		Timeout:  60,
	}

	rateLimiter := loop.NewRateLimiter(100, 1)
	breaker := circuit.NewBreaker(3, 5)
	controller := loop.NewController(cfg, rateLimiter, breaker)

	fake := &fakeRunner{
		projectDir: project.Dir,
		planFile:   project.PlanFile,
		maxCalls:   5,
	}
	controller.SetRunner(fake)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := controller.Run(ctx)

	// Should complete without error
	if err != nil {
		t.Errorf("Loop failed: %v", err)
	}

	// Fake runner should NOT have been called (loop exits early)
	if fake.callCount != 0 {
		t.Errorf("Expected runner to not be called when all tasks complete, but was called %d times", fake.callCount)
	}

	t.Log("Loop correctly exited early without calling runner")
}

func TestE2E_PreflightAndOutcomeEvents(t *testing.T) {
	project := setupTestProject(t, "fix")

	// Change to project directory
	origDir, _ := os.Getwd()
	os.Chdir(project.Dir)
	defer os.Chdir(origDir)

	cfg := config.Config{
		MaxCalls: 10,
		Backend:  "test",
		Timeout:  60,
	}

	rateLimiter := loop.NewRateLimiter(100, 1)
	breaker := circuit.NewBreaker(3, 5)
	controller := loop.NewController(cfg, rateLimiter, breaker)

	fake := &fakeRunner{
		projectDir: project.Dir,
		planFile:   project.PlanFile,
		maxCalls:   5,
	}
	controller.SetRunner(fake)

	// Collect all events
	var preflightEvents []*loop.PreflightSummary
	var outcomeEvents []*loop.LoopOutcome
	var logMessages []string

	controller.SetEventCallback(func(event loop.LoopEvent) {
		switch event.Type {
		case loop.EventTypePreflight:
			if event.Preflight != nil {
				preflightEvents = append(preflightEvents, event.Preflight)
			}
		case loop.EventTypeOutcome:
			if event.Outcome != nil {
				outcomeEvents = append(outcomeEvents, event.Outcome)
			}
		case loop.EventTypeLog:
			logMessages = append(logMessages, event.LogMessage)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := controller.Run(ctx)

	if err != nil {
		t.Errorf("Loop failed: %v", err)
	}

	// Verify preflight events
	if len(preflightEvents) == 0 {
		t.Error("Expected preflight events")
	}

	// First preflight should show 3 remaining
	if len(preflightEvents) > 0 {
		first := preflightEvents[0]
		if first.RemainingCount != 3 {
			t.Errorf("First preflight expected 3 remaining, got %d", first.RemainingCount)
		}
		if first.Mode != "fix" {
			t.Errorf("Expected mode 'fix', got '%s'", first.Mode)
		}
	}

	// Last preflight should show 0 or 1 remaining (preflight runs before iteration)
	if len(preflightEvents) > 0 {
		last := preflightEvents[len(preflightEvents)-1]
		// Preflight runs BEFORE the iteration, so if we captured 3 prefights for 3 tasks,
		// the last preflight might show 1 remaining (before final task is marked complete)
		if last.RemainingCount > 1 {
			t.Errorf("Last preflight expected 0 or 1 remaining, got %d", last.RemainingCount)
		}
	}

	// Verify outcome events were emitted
	if len(outcomeEvents) == 0 {
		t.Error("Expected outcome events")
	}

	// All outcomes should be successful
	for i, outcome := range outcomeEvents {
		if !outcome.Success {
			t.Errorf("Outcome %d was not successful", i)
		}
	}

	t.Logf("Captured %d preflight events, %d outcome events", len(preflightEvents), len(outcomeEvents))
	t.Logf("Captured %d log messages", len(logMessages))
}

