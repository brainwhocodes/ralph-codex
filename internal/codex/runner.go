package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/brainwhocodes/lisa-loop/internal/state"
)

// OutputCallback is called for each line of streaming output
type OutputCallback func(event Event)

// Runner executes Codex commands
type Runner struct {
	config         Config
	outputCallback OutputCallback
}

// NewRunner creates a new Codex runner
func NewRunner(config Config) *Runner {
	return &Runner{config: config}
}

// SetOutputCallback sets the callback for streaming output
func (r *Runner) SetOutputCallback(cb OutputCallback) {
	r.outputCallback = cb
}

// Run executes a Codex command using the CLI with streaming
func (r *Runner) Run(prompt string) (output string, threadID string, err error) {
	return r.runCLI(prompt)
}

// runCLI executes Codex CLI in non-interactive mode with streaming
func (r *Runner) runCLI(prompt string) (string, string, error) {
	args := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
		"--sandbox", "danger-full-access",
	}

	// Add thread ID if session exists for conversation continuity
	if id, err := LoadSessionID(); err == nil && id != "" {
		args = append(args, "resume", "--last")
	}

	cmd := exec.Command("codex", args...)
	cmd.Stdin = strings.NewReader(prompt)

	if r.config.Verbose {
		fmt.Printf("Executing: codex %s\n", strings.Join(args, " "))
	}

	// Set up pipes for streaming
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return "", "", fmt.Errorf("failed to start codex: %w", err)
	}

	// Read output streams
	var outputBuilder strings.Builder
	var threadID string
	var message strings.Builder

	// Process stdout (JSONL events)
	// Use 1MB buffer to handle large JSONL lines from Codex
	const maxScannerBuffer = 1024 * 1024 // 1MB
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, maxScannerBuffer), maxScannerBuffer)

	for scanner.Scan() {
		line := scanner.Text()
		outputBuilder.WriteString(line)
		outputBuilder.WriteString("\n")

		// Parse and handle event
		event, err := ParseJSONLLine(line)
		if err != nil {
			continue
		}

		// Extract thread ID
		if EventType(event) == "thread.started" {
			tid := ThreadID(event)
			if tid != "" {
				threadID = tid
			}
		}

		// Accumulate message text
		if MessageType(event) == "message" || MessageType(event) == "text" {
			msg := MessageText(event)
			if msg != "" {
				message.WriteString(msg)
				message.WriteString("\n")
			}
		}

		// Call output callback for real-time updates
		if r.outputCallback != nil && event != nil {
			r.outputCallback(event)
		}
	}

	// Check for scanner errors (e.g., token too long)
	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("error reading codex output: %w", err)
	}

	// Read stderr with same buffer size
	stderrScanner := bufio.NewScanner(stderr)
	stderrScanner.Buffer(make([]byte, maxScannerBuffer), maxScannerBuffer)
	var stderrOutput strings.Builder
	for stderrScanner.Scan() {
		stderrOutput.WriteString(stderrScanner.Text())
		stderrOutput.WriteString("\n")
	}
	if err := stderrScanner.Err(); err != nil {
		// Log stderr scanner error but don't fail - stderr is secondary
		if r.config.Verbose {
			fmt.Printf("Warning: error reading stderr: %v\n", err)
		}
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		errMsg := stderrOutput.String()
		if errMsg == "" {
			errMsg = outputBuilder.String()
		}
		return "", "", fmt.Errorf("codex execution failed: %w\nOutput: %s", err, errMsg)
	}

	// Save session ID if we got one
	if threadID != "" {
		if err := SaveSessionID(threadID); err != nil {
			return outputBuilder.String(), threadID, fmt.Errorf("failed to save session ID: %w", err)
		}
	}

	// Return message content instead of full output
	if message.Len() > 0 {
		return strings.TrimSpace(message.String()), threadID, nil
	}

	return outputBuilder.String(), threadID, nil
}

// Event represents a single JSONL event from Codex
type Event map[string]interface{}

// ParseJSONLLine parses a single JSONL line
func ParseJSONLLine(line string) (Event, error) {
	if line == "" || strings.TrimSpace(line) == "" {
		return nil, nil
	}

	var event Event
	err := json.Unmarshal([]byte(line), &event)
	if err != nil {
		return nil, err
	}

	return event, nil
}

// EventType extracts the "event" field from an event
func EventType(event Event) string {
	if val, ok := event["event"].(string); ok {
		return val
	}
	return ""
}

// ThreadID extracts the "thread_id" field from an event
func ThreadID(event Event) string {
	if val, ok := event["thread_id"].(string); ok {
		return val
	}
	return ""
}

// MessageType extracts the "type" field from an event
func MessageType(event Event) string {
	if val, ok := event["type"].(string); ok {
		return val
	}
	return ""
}

// MessageText extracts the "text" field from an event
func MessageText(event Event) string {
	if val, ok := event["text"].(string); ok {
		return val
	}
	return ""
}

// ParseJSONLStream parses a complete JSONL stream
func ParseJSONLStream(lines []string) (threadID string, message string, events []Event) {
	events = make([]Event, 0, len(lines))
	b := strings.Builder{}

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		event, err := ParseJSONLLine(line)
		if err != nil {
			continue
		}

		events = append(events, event)

		if EventType(event) == "thread.started" {
			tid := ThreadID(event)
			if tid != "" {
				threadID = tid
			}
		}

		if MessageType(event) == "message" || MessageType(event) == "text" {
			msg := MessageText(event)
			if msg != "" {
				b.WriteString(msg)
				b.WriteString("\n")
			}
		}
	}

	return threadID, strings.TrimSpace(b.String()), events
}

// IsJSONL checks if a line looks like JSON
func IsJSONL(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}

// LoadSessionID loads Codex session ID
func LoadSessionID() (string, error) {
	return state.LoadCodexSession()
}

// SaveSessionID saves Codex session ID
func SaveSessionID(id string) error {
	return state.SaveCodexSession(id)
}

// NewSession creates a new session by clearing the session ID
func NewSession() error {
	return SaveSessionID("")
}

// SessionExists checks if a session ID exists
func SessionExists() bool {
	id, err := LoadSessionID()
	if err != nil {
		return false
	}
	return id != ""
}

// SessionAgeHours calculates the age of the session in hours
func SessionAgeHours() (int, error) {
	if !SessionExists() {
		return 0, nil
	}

	sessionFile := ".codex_session_id"
	info, err := os.Stat(sessionFile)
	if err != nil {
		return 0, err
	}

	age := time.Since(info.ModTime()).Hours()
	return int(age), nil
}

// IsSessionExpired checks if the session has expired
func IsSessionExpired(expiryHours int) bool {
	if expiryHours <= 0 {
		return false
	}

	age, err := SessionAgeHours()
	if err != nil {
		return false
	}

	return age >= expiryHours
}

// SessionMetadata represents session metadata
type SessionMetadata struct {
	ID        string
	CreatedAt time.Time
	LastUsed  time.Time
}

// LoadSessionMetadata loads session metadata
func LoadSessionMetadata() (*SessionMetadata, error) {
	sess, err := state.LoadLisaSession()
	if err != nil {
		return nil, err
	}

	meta := &SessionMetadata{
		ID: "",
	}

	if id, ok := sess["id"].(string); ok {
		meta.ID = id
	}

	if created, ok := sess["created_at"].(string); ok {
		meta.CreatedAt, _ = time.Parse(time.RFC3339, created)
	}

	if lastUsed, ok := sess["last_used"].(string); ok {
		meta.LastUsed, _ = time.Parse(time.RFC3339, lastUsed)
	}

	return meta, nil
}

// SaveSessionMetadata saves session metadata
func SaveSessionMetadata(meta *SessionMetadata) error {
	sess := map[string]interface{}{
		"id":         meta.ID,
		"created_at": meta.CreatedAt.Format(time.RFC3339),
		"last_used":  meta.LastUsed.Format(time.RFC3339),
	}

	return state.SaveLisaSession(sess)
}
