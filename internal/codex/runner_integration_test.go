// +build integration

package codex

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestCodexExecRawOutput runs codex exec directly to see raw JSONL output
func TestCodexExecRawOutput(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// Change to test project directory
	testDir := os.Getenv("TEST_PROJECT_DIR")
	if testDir == "" {
		testDir = "."
	}

	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	prompt := "What tasks are in the @fix_plan.md file? List them."

	args := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
		"--sandbox", "danger-full-access",
	}

	cmd := exec.Command("codex", args...)
	cmd.Stdin = strings.NewReader(prompt)

	fmt.Printf("=== Running: codex %s ===\n", strings.Join(args, " "))
	fmt.Printf("=== Prompt: %s ===\n\n", prompt)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Command error (may be expected): %v", err)
	}

	fmt.Println("=== RAW OUTPUT ===")
	fmt.Println(string(output))
	fmt.Println("=== END RAW OUTPUT ===")

	// Parse each line as JSONL
	fmt.Println("\n=== PARSED EVENTS ===")
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		event, err := ParseJSONLLine(line)
		if err != nil {
			fmt.Printf("Line %d (not JSON): %s\n", i, line)
			continue
		}

		eventType := EventType(event)
		msgType := MessageType(event)
		text := MessageText(event)

		fmt.Printf("Line %d:\n", i)
		fmt.Printf("  event: %s\n", eventType)
		fmt.Printf("  type: %s\n", msgType)
		if text != "" {
			fmt.Printf("  text: %s\n", text)
		}
		fmt.Printf("  raw: %v\n\n", event)
	}
	fmt.Println("=== END PARSED EVENTS ===")
}

// TestRunnerWithCallback tests the runner with a callback to see events
func TestRunnerWithCallback(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	testDir := os.Getenv("TEST_PROJECT_DIR")
	if testDir == "" {
		testDir = "."
	}

	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	config := Config{
		Backend:     "cli",
		ProjectPath: ".",
		PromptPath:  "PROMPT.md",
		MaxCalls:    10,
		Timeout:     60,
		Verbose:     true,
	}

	runner := NewRunner(config)

	fmt.Println("=== EVENTS FROM CALLBACK ===")
	runner.SetOutputCallback(func(event Event) {
		eventType := EventType(event)
		msgType := MessageType(event)
		text := MessageText(event)

		fmt.Printf("EVENT:\n")
		fmt.Printf("  event: %q\n", eventType)
		fmt.Printf("  type: %q\n", msgType)
		if text != "" {
			fmt.Printf("  text: %q\n", text)
		}
		fmt.Println()
	})

	prompt := "Say hello and describe what you see in the project."
	output, threadID, err := runner.Run(prompt)

	fmt.Println("=== FINAL OUTPUT ===")
	fmt.Printf("Thread ID: %s\n", threadID)
	fmt.Printf("Error: %v\n", err)
	fmt.Printf("Output:\n%s\n", output)
	fmt.Println("=== END ===")
}
