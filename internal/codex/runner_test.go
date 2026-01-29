package codex

import (
	"os"
	"strings"
	"testing"

	"github.com/brainwhocodes/lisa-loop/internal/state"
)

func TestParseJSONLLineValid(t *testing.T) {
	line := `{"event": "thread.started", "thread_id": "test-123"}`

	event, err := ParseJSONLLine(line)

	if err != nil {
		t.Errorf("ParseJSONLLine() error = %v", err)
	}

	if EventType(event) != "thread.started" {
		t.Errorf("EventType() = %s, want thread.started", EventType(event))
	}

	if ThreadID(event) != "test-123" {
		t.Errorf("ThreadID() = %s, want test-123", ThreadID(event))
	}
}

func TestParseJSONLLineMessage(t *testing.T) {
	line := `{"type": "message", "text": "Hello world"}`

	event, err := ParseJSONLLine(line)

	if err != nil {
		t.Errorf("ParseJSONLLine() error = %v", err)
	}

	if MessageType(event) != "message" {
		t.Errorf("MessageType() = %s, want message", MessageType(event))
	}

	if MessageText(event) != "Hello world" {
		t.Errorf("MessageText() = %s, want Hello world", MessageText(event))
	}
}

func TestParseJSONLStream(t *testing.T) {
	lines := []string{
		`{"event": "thread.started", "thread_id": "thread-abc"}`,
		`{"type": "message", "text": "First message"}`,
		`{"type": "message", "text": "Second message"}`,
	}

	threadID, message, _ := ParseJSONLStream(lines)

	if threadID != "thread-abc" {
		t.Errorf("ParseJSONLStream() threadID = %s, want thread-abc", threadID)
	}

	if message != "First message\nSecond message" {
		t.Errorf("ParseJSONLStream() message = %s, want both messages", message)
	}
}

func TestSessionSaveLoad(t *testing.T) {
	// Setup: create temporary dir
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Test: Load non-existent session
	os.Chdir(tmpDir)
	id, err := LoadSessionID()

	if err != nil {
		t.Errorf("LoadSessionID() error = %v, want nil", err)
	}

	if id != "" {
		t.Errorf("LoadSessionID() id = %s, want empty", id)
	}

	// Test: Save and verify
	testID := "session-test-456"
	err = SaveSessionID(testID)

	if err != nil {
		t.Errorf("SaveSessionID() error = %v, want nil", err)
	}

	loaded, _ := LoadSessionID()

	if loaded != testID {
		t.Errorf("LoadSessionID() loaded = %s, want %s", loaded, testID)
	}
}

func TestParseJSONLLineInvalid(t *testing.T) {
	line := `{invalid json}`

	_, err := ParseJSONLLine(line)

	if err == nil {
		t.Error("ParseJSONLLine() expected error for invalid JSON")
	}
}

func TestParseJSONLLineEmpty(t *testing.T) {
	line := ""

	event, err := ParseJSONLLine(line)

	if err != nil {
		t.Errorf("ParseJSONLLine() empty line error = %v", err)
	}

	if event != nil {
		t.Error("ParseJSONLLine() empty line expected nil event")
	}
}

func TestEventType(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  string
	}{
		{
			name:  "thread.started event",
			event: Event{"event": "thread.started", "thread_id": "test-123"},
			want:  "thread.started",
		},
		{
			name:  "message event",
			event: Event{"type": "message", "text": "hello"},
			want:  "",
		},
		{
			name:  "no event field",
			event: Event{"thread_id": "test-123"},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EventType(tt.event); got != tt.want {
				t.Errorf("EventType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestThreadID(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  string
	}{
		{
			name:  "has thread_id",
			event: Event{"event": "thread.started", "thread_id": "test-123"},
			want:  "test-123",
		},
		{
			name:  "no thread_id",
			event: Event{"event": "thread.started"},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ThreadID(tt.event); got != tt.want {
				t.Errorf("ThreadID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageType(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  string
	}{
		{
			name:  "message type",
			event: Event{"type": "message", "text": "hello"},
			want:  "message",
		},
		{
			name:  "text type",
			event: Event{"type": "text", "text": "world"},
			want:  "text",
		},
		{
			name:  "no type field",
			event: Event{"text": "hello"},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MessageType(tt.event); got != tt.want {
				t.Errorf("MessageType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageText(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  string
	}{
		{
			name:  "has text",
			event: Event{"type": "message", "text": "hello world"},
			want:  "hello world",
		},
		{
			name:  "no text field",
			event: Event{"type": "message"},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MessageText(tt.event); got != tt.want {
				t.Errorf("MessageText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsJSONL(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "object start",
			line: `{"event": "test"}`,
			want: true,
		},
		{
			name: "array start",
			line: `[1, 2, 3]`,
			want: true,
		},
		{
			name: "whitespace and object",
			line: `  {"event": "test"}`,
			want: true,
		},
		{
			name: "plain text",
			line: "hello world",
			want: false,
		},
		{
			name: "empty",
			line: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsJSONL(tt.line); got != tt.want {
				t.Errorf("IsJSONL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSession(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	os.Chdir(tmpDir)

	SaveSessionID("test-session")

	err := NewSession()
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	id, err := LoadSessionID()
	if err != nil {
		t.Fatalf("LoadSessionID() after NewSession() error = %v", err)
	}

	if id != "" {
		t.Errorf("LoadSessionID() after NewSession() = %v, want empty", id)
	}
}

func TestSessionExists(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	os.Chdir(tmpDir)

	if SessionExists() {
		t.Error("SessionExists() with no session = true, want false")
	}

	SaveSessionID("test-session")

	if !SessionExists() {
		t.Error("SessionExists() with session = false, want true")
	}
}

func TestRunnerCreate(t *testing.T) {
	config := Config{
		Backend: "cli",
		Timeout: 600,
		Verbose: true,
	}

	runner := NewRunner(config)

	if runner == nil {
		t.Fatal("NewRunner() returned nil")
	}

	if runner.config.Backend != "cli" {
		t.Errorf("NewRunner() backend = %v, want cli", runner.config.Backend)
	}
}

func TestParseComplexJSONL(t *testing.T) {
	jsonlStream := `{"event": "thread.started", "thread_id": "thread-abc-123"}
{"type": "message", "text": "First line"}
{"type": "message", "text": "Second line"}
{"event": "thread.completed", "thread_id": "thread-abc-123"}`

	lines := strings.Split(jsonlStream, "\n")
	threadID, message, events := ParseJSONLStream(lines)

	if threadID != "thread-abc-123" {
		t.Errorf("ParseJSONLStream() threadID = %v, want thread-abc-123", threadID)
	}

	expectedMessage := "First line\nSecond line"
	if message != expectedMessage {
		t.Errorf("ParseJSONLStream() message = %v, want %v", message, expectedMessage)
	}

	if len(events) != 4 {
		t.Errorf("ParseJSONLStream() events count = %v, want 4", len(events))
	}
}

func TestParseLargeJSONL(t *testing.T) {
	// Generate a large JSON payload (> 64KB default scanner buffer)
	largeText := strings.Repeat("x", 100000) // 100KB of text
	jsonLine := `{"type": "message", "text": "` + largeText + `"}`

	event, err := ParseJSONLLine(jsonLine)
	if err != nil {
		t.Errorf("ParseJSONLLine() large line error = %v", err)
	}

	if MessageText(event) != largeText {
		t.Errorf("ParseJSONLLine() large line text length = %d, want %d", len(MessageText(event)), len(largeText))
	}
}

func TestParseStreamWithLargeLines(t *testing.T) {
	// Test ParseJSONLStream with large lines
	largeText := strings.Repeat("y", 100000) // 100KB of text
	lines := []string{
		`{"event": "thread.started", "thread_id": "thread-large"}`,
		`{"type": "message", "text": "` + largeText + `"}`,
		`{"type": "message", "text": "normal message"}`,
	}

	threadID, message, events := ParseJSONLStream(lines)

	if threadID != "thread-large" {
		t.Errorf("ParseJSONLStream() threadID = %v, want thread-large", threadID)
	}

	if !strings.Contains(message, largeText) {
		t.Error("ParseJSONLStream() did not preserve large text content")
	}

	if len(events) != 3 {
		t.Errorf("ParseJSONLStream() events count = %d, want 3", len(events))
	}
}

func TestStatePersistence(t *testing.T) {
	t.Run("Save and Load Codex Session", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)

		os.Chdir(tmpDir)

		sessionID := "session-" + strings.Repeat("x", 20)

		err := state.SaveCodexSession(sessionID)
		if err != nil {
			t.Fatalf("state.SaveCodexSession() error = %v", err)
		}

		loadedID, err := state.LoadCodexSession()
		if err != nil {
			t.Fatalf("state.LoadCodexSession() error = %v", err)
		}

		if loadedID != sessionID {
			t.Errorf("state.LoadCodexSession() = %v, want %v", loadedID, sessionID)
		}
	})

	t.Run("Save and Load Lisa Session", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)

		os.Chdir(tmpDir)

		sessionData := map[string]interface{}{
			"id":         "ralph-session-123",
			"created_at": "2024-01-01T00:00:00Z",
			"last_used":  "2024-01-01T01:00:00Z",
		}

		err := state.SaveLisaSession(sessionData)
		if err != nil {
			t.Fatalf("state.SaveLisaSession() error = %v", err)
		}

		loadedData, err := state.LoadLisaSession()
		if err != nil {
			t.Fatalf("state.LoadLisaSession() error = %v", err)
		}

		if loadedData["id"] != sessionData["id"] {
			t.Errorf("state.LoadLisaSession() id = %v, want %v", loadedData["id"], sessionData["id"])
		}
	})
}
