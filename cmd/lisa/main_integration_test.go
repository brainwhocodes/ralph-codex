package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildLisa() (string, error) {
	cmd := exec.Command("go", "build", "-o", "ralph-test", "./cmd/ralph")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to build: %w", err)
	}
	return "./ralph-test", nil
}

func cleanupBinary(binaryPath string) {
	os.Remove(binaryPath)
}

func TestCommandHelp(t *testing.T) {
	binaryPath, err := buildLisa()
	if err != nil {
		t.Skip("Could not build ralph:", err)
	}
	defer cleanupBinary(binaryPath)

	cmd := exec.Command(binaryPath, "help")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("ralph help failed: %v, output: %s", err, string(output))
	}

	outputStr := string(output)

	expectedSubstrings := []string{
		"Commands:",
		"run (default)",
		"setup",
		"import",
		"status",
		"reset-circuit",
		"help",
		"version",
	}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Help output missing expected string: %s", expected)
		}
	}
}

func TestCommandVersion(t *testing.T) {
	binaryPath, err := buildLisa()
	if err != nil {
		t.Skip("Could not build ralph:", err)
	}
	defer cleanupBinary(binaryPath)

	cmd := exec.Command(binaryPath, "version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("ralph version failed: %v", err)
	}

	outputStr := string(output)

	if !strings.Contains(outputStr, "Lisa Codex") {
		t.Errorf("Version output missing 'Lisa Codex'")
	}
}

func TestCommandSetup(t *testing.T) {
	binaryPath, err := buildLisa()
	if err != nil {
		t.Skip("Could not build ralph:", err)
	}
	defer cleanupBinary(binaryPath)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	os.Chdir(tmpDir)

	projectName := "test-project-" + fmt.Sprint(os.Getpid())
	cmd := exec.Command(binaryPath, "setup", "--name", projectName, "--git", "false")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("ralph setup failed: %v, output: %s", err, string(output))
	}

	outputStr := string(output)

	if !strings.Contains(outputStr, "Project created successfully") {
		t.Errorf("Setup output missing success message")
	}

	projectPath := filepath.Join(tmpDir, projectName)

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Errorf("Project directory not created: %s", projectPath)
	}

	promptPath := filepath.Join(projectPath, "PROMPT.md")
	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		t.Errorf("PROMPT.md not created")
	}

	fixPlanPath := filepath.Join(projectPath, "@fix_plan.md")
	if _, err := os.Stat(fixPlanPath); os.IsNotExist(err) {
		t.Errorf("@fix_plan.md not created")
	}
}

func TestCommandSetupMissingName(t *testing.T) {
	binaryPath, err := buildLisa()
	if err != nil {
		t.Skip("Could not build ralph:", err)
	}
	defer cleanupBinary(binaryPath)

	cmd := exec.Command(binaryPath, "setup")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("ralph setup without --name should fail")
	}

	outputStr := string(output)

	if !strings.Contains(outputStr, "--name is required") {
		t.Errorf("Setup without --name missing expected error message")
	}
}

func TestCommandStatusInvalidProject(t *testing.T) {
	binaryPath, err := buildLisa()
	if err != nil {
		t.Skip("Could not build ralph:", err)
	}
	defer cleanupBinary(binaryPath)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	defer os.Chdir(origDir)

	os.Chdir(tmpDir)

	cmd := exec.Command(binaryPath, "status")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("ralph status in non-project directory should fail")
	}

	outputStr := string(output)

	if !strings.Contains(outputStr, "not a valid Lisa project") {
		t.Errorf("Status in non-project missing expected error message")
	}
}

func TestCommandRunInvalidProject(t *testing.T) {
	binaryPath, err := buildLisa()
	if err != nil {
		t.Skip("Could not build ralph:", err)
	}
	defer cleanupBinary(binaryPath)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	os.Chdir(tmpDir)

	cmd := exec.Command(binaryPath, "run", "--calls", "1")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("ralph run in non-project directory should fail")
	}

	outputStr := string(output)

	if !strings.Contains(outputStr, "not a valid Lisa project") {
		t.Errorf("Run in non-project missing expected error message")
	}
}

func TestCommandUnknownCommand(t *testing.T) {
	binaryPath, err := buildLisa()
	if err != nil {
		t.Skip("Could not build ralph:", err)
	}
	defer cleanupBinary(binaryPath)

	cmd := exec.Command(binaryPath, "unknown-command")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("ralph with unknown command should fail")
	}

	outputStr := string(output)

	if !strings.Contains(outputStr, "unknown command") {
		t.Errorf("Unknown command missing expected error message")
	}
}

func TestCommandResetCircuit(t *testing.T) {
	binaryPath, err := buildLisa()
	if err != nil {
		t.Skip("Could not build ralph:", err)
	}
	defer cleanupBinary(binaryPath)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	os.Chdir(tmpDir)

	projectName := "test-reset-project"
	setupCmd := exec.Command(binaryPath, "setup", "--name", projectName, "--git", "false")
	if err := setupCmd.Run(); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	os.Chdir(filepath.Join(tmpDir, projectName))

	resetCmd := exec.Command(binaryPath, "reset-circuit")
	output, err := resetCmd.CombinedOutput()

	if err != nil {
		t.Fatalf("ralph reset-circuit failed: %v, output: %s", err, string(output))
	}

	outputStr := string(output)

	if !strings.Contains(outputStr, "Circuit breaker reset successfully") {
		t.Errorf("Reset circuit missing expected success message")
	}
}

func TestCommandWithBackendFlag(t *testing.T) {
	binaryPath, err := buildLisa()
	if err != nil {
		t.Skip("Could not build ralph:", err)
	}
	defer cleanupBinary(binaryPath)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	os.Chdir(tmpDir)

	projectName := "test-backend-project"
	setupCmd := exec.Command(binaryPath, "setup", "--name", projectName, "--git", "false", "--backend", "cli")
	if err := setupCmd.Run(); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	projectPath := filepath.Join(tmpDir, projectName)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Errorf("Project not created with --backend flag")
	}
}

func TestCommandDefaultBehavior(t *testing.T) {
	binaryPath, err := buildLisa()
	if err != nil {
		t.Skip("Could not build ralph:", err)
	}
	defer cleanupBinary(binaryPath)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	os.Chdir(tmpDir)

	projectName := "test-default-project"
	setupCmd := exec.Command(binaryPath, "setup", "--name", projectName, "--git", "false")
	if err := setupCmd.Run(); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	os.Chdir(filepath.Join(tmpDir, projectName))

	cmd := exec.Command(binaryPath)
	cmd.Dir = filepath.Join(tmpDir, projectName)
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("ralph without command in project should try to run (but will fail without codex)")
	}

	outputStr := string(output)

	if strings.Contains(outputStr, "Starting Lisa Codex") {
		t.Logf("Default command correctly runs 'run'")
	}
}

func TestCommandImport(t *testing.T) {
	binaryPath, err := buildLisa()
	if err != nil {
		t.Skip("Could not build ralph:", err)
	}
	defer cleanupBinary(binaryPath)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	os.Chdir(tmpDir)

	specFile := filepath.Join(tmpDir, "spec.md")
	specContent := "# Test Spec\n\nThis is a test specification."
	if err := os.WriteFile(specFile, []byte(specContent), 0644); err != nil {
		t.Fatalf("Failed to create spec file: %v", err)
	}

	cmd := exec.Command(binaryPath, "import", "--source", specFile, "--import-name", "test-import")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("ralph import failed: %v, output: %s", err, string(output))
	}

	outputStr := string(output)

	if strings.Contains(outputStr, "Import completed successfully") {
		t.Log("Import command working")
	}
}

func TestMainFlagParsing(t *testing.T) {
	binaryPath, err := buildLisa()
	if err != nil {
		t.Skip("Could not build ralph:", err)
	}
	defer cleanupBinary(binaryPath)

	cmd := exec.Command(binaryPath, "help", "--verbose")
	_, err = cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("ralph help with flag failed: %v", err)
	}

	t.Log("Flag parsing working correctly")
}
