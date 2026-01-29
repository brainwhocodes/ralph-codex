// Command test-opencode is an integration test for the OpenCode backend.
// It requires a running OpenCode server and tests the full flow.
//
// Usage:
//
//	export OPENCODE_SERVER_URL="http://localhost:8080"
//	export OPENCODE_SERVER_PASSWORD="your-password"
//	go run ./cmd/test-opencode
package main

import (
	"fmt"
	"os"

	"github.com/brainwhocodes/lisa-loop/internal/config"
	"github.com/brainwhocodes/lisa-loop/internal/opencode"
	"github.com/charmbracelet/log"
)

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
	})
	logger.SetLevel(log.DebugLevel)

	// Load config from environment
	serverURL := os.Getenv("OPENCODE_SERVER_URL")
	if serverURL == "" {
		logger.Fatal("OPENCODE_SERVER_URL is required")
	}

	password := os.Getenv("OPENCODE_SERVER_PASSWORD")
	// Password is optional - server may be unsecured

	username := os.Getenv("OPENCODE_SERVER_USERNAME")
	if username == "" {
		username = "opencode"
	}

	modelID := os.Getenv("OPENCODE_MODEL_ID")
	if modelID == "" {
		modelID = "glm-4.7"
	}

	logger.Info("OpenCode Integration Test",
		"server", serverURL,
		"username", username,
		"model", modelID,
	)

	// Clear any existing session
	logger.Info("Clearing existing session...")
	if err := opencode.ClearSession(); err != nil {
		logger.Warn("Failed to clear session", "error", err)
	}

	// Test 1: Create client and session
	logger.Info("Test 1: Creating OpenCode client...")
	client := opencode.NewClient(opencode.Config{
		ServerURL: serverURL,
		Username:  username,
		Password:  password,
		ModelID:   modelID,
	})

	logger.Info("Test 1: Creating session...")
	sessionID, err := client.CreateSession()
	if err != nil {
		logger.Fatal("Failed to create session", "error", err)
	}
	logger.Info("Test 1: Session created", "session_id", sessionID)

	// Test 2: Send a simple message
	logger.Info("Test 2: Sending message...")
	resp, err := client.SendMessage(sessionID, "Say 'Hello from OpenCode integration test' and nothing else.")
	if err != nil {
		logger.Fatal("Failed to send message", "error", err)
	}
	logger.Info("Test 2: Response received",
		"content_length", len(resp.Content()),
		"session_id", resp.SessionID(),
	)
	logger.Debug("Response content", "content", resp.Content())

	// Test 3: Test runner with output callback
	logger.Info("Test 3: Testing runner...")

	cfg := config.Config{
		Backend:           "opencode",
		OpenCodeServerURL: serverURL,
		OpenCodeUsername:  username,
		OpenCodePassword:  password,
		OpenCodeModelID:   modelID,
		Timeout:           300,
		Verbose:           true,
	}

	runner := opencode.NewRunner(cfg)

	var events []map[string]interface{}
	runner.SetOutputCallback(func(event map[string]interface{}) {
		events = append(events, event)
		logger.Debug("Event received", "event", event["event"])
	})

	output, sid, err := runner.Run("What is 2 + 2? Reply with just the number.")
	if err != nil {
		logger.Fatal("Runner failed", "error", err)
	}

	logger.Info("Test 3: Runner completed",
		"output_length", len(output),
		"session_id", sid,
		"events_count", len(events),
	)
	logger.Debug("Runner output", "output", output)

	// Test 4: Session persistence
	logger.Info("Test 4: Testing session persistence...")
	savedID, err := opencode.LoadSessionID()
	if err != nil {
		logger.Fatal("Failed to load session ID", "error", err)
	}
	if savedID == "" {
		logger.Fatal("Session ID not persisted")
	}
	logger.Info("Test 4: Session persisted", "session_id", savedID)

	// Test 5: Resume session
	logger.Info("Test 5: Testing session resume...")
	output2, sid2, err := runner.Run("What was the previous question I asked?")
	if err != nil {
		logger.Fatal("Failed to resume session", "error", err)
	}
	logger.Info("Test 5: Session resumed",
		"output_length", len(output2),
		"session_id", sid2,
	)
	logger.Debug("Resume output", "output", output2)

	// Cleanup
	logger.Info("Cleaning up...")
	if err := opencode.ClearSession(); err != nil {
		logger.Warn("Failed to clear session", "error", err)
	}

	fmt.Println()
	logger.Info("All integration tests passed!")
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Println("  - Session creation: OK")
	fmt.Println("  - Message sending: OK")
	fmt.Println("  - Runner execution: OK")
	fmt.Println("  - Session persistence: OK")
	fmt.Println("  - Session resume: OK")
}
