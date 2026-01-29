package project

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/brainwhocodes/lisa-loop/internal/codex"
)

func TestParseCodexJSONL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantEvents int
	}{
		{
			name: "valid JSONL with message events",
			input: `{"type": "message", "role": "user", "content": "Hello"}
{"type": "message", "role": "assistant", "content": "Hi there"}`,
			wantErr:    false,
			wantEvents: 2,
		},
		{
			name:       "empty input",
			input:      "",
			wantErr:    false,
			wantEvents: 0,
		},
		{
			name: "mixed valid and invalid lines",
			input: `{"type": "message", "content": "Valid"}
This is not JSON
{"type": "tool_call", "name": "read_file"}`,
			wantErr:    false,
			wantEvents: 2, // Only valid JSON lines are parsed
		},
		{
			name: "content_block_delta event",
			input: `{"type": "content_block_delta", "delta": {"type": "text_delta", "text": "Hello"}}`,
			wantErr:    false,
			wantEvents: 1,
		},
		{
			name: "assistant content array",
			input: `{"type": "assistant", "content": [{"type": "text", "text": "First part"}, {"type": "text", "text": "Second part"}]}`,
			wantErr:    false,
			wantEvents: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			events, err := ParseCodexJSONL(reader)

			if tt.wantErr && err == nil {
				t.Errorf("ParseCodexJSONL() error = nil, wantErr %v", tt.wantErr)
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ParseCodexJSONL() unexpected error = %v", err)
				return
			}

			if len(events) != tt.wantEvents {
				t.Errorf("ParseCodexJSONL() returned %d events, want %d", len(events), tt.wantEvents)
			}
		})
	}
}

func TestParseCodexJSONL_LargeLine(t *testing.T) {
	// Create a very large JSON line (larger than default 64KB scanner buffer)
	largeContent := strings.Repeat("a", 100*1024) // 100KB of text
	input := `{"type": "message", "role": "assistant", "content": "` + largeContent + `"}`

	reader := strings.NewReader(input)
	events, err := ParseCodexJSONL(reader)

	if err != nil {
		t.Errorf("ParseCodexJSONL() failed with large line: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("ParseCodexJSONL() returned %d events, want 1", len(events))
	}

	// Verify the content was captured
	if content, ok := events[0]["content"].(string); ok {
		if len(content) != len(largeContent) {
			t.Errorf("ParseCodexJSONL() content length = %d, want %d", len(content), len(largeContent))
		}
	} else {
		t.Error("ParseCodexJSONL() content field missing or wrong type")
	}
}

func TestParseCodexJSONL_MultilineJSON(t *testing.T) {
	// Test handling of pretty-printed JSON (not typical for JSONL but should handle)
	input := `{
		"type": "message",
		"role": "assistant",
		"content": "Hello"
	}`

	reader := strings.NewReader(input)
	events, err := ParseCodexJSONL(reader)

	// Scanner will read line by line, so multiline JSON won't parse correctly
	// This is expected behavior for JSONL format
	if err != nil {
		t.Errorf("ParseCodexJSONL() unexpected error: %v", err)
	}

	// Should have 0 valid events since multiline JSON isn't valid JSONL
	if len(events) != 0 {
		t.Errorf("ParseCodexJSONL() returned %d events for multiline JSON, expected 0", len(events))
	}
}

func TestCodexEventParsing(t *testing.T) {
	// Test that parsed events work with the codex.ParseEvent function
	// Using actual Codex event format from the real API
	input := `{"type": "message", "role": "assistant", "content": [{"type": "text", "text": "Hello world"}]}
{"type": "item.completed", "item": {"type": "reasoning", "text": "Thinking..."}}
{"type": "tool_use", "name": "read_file", "input": {"file_path": "/path/to/file"}}`

	reader := strings.NewReader(input)
	events, err := ParseCodexJSONL(reader)
	if err != nil {
		t.Fatalf("ParseCodexJSONL() error = %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	// Test each event with the codex parser
	for i, event := range events {
		parsed := codex.ParseEvent(event)
		if parsed == nil {
			t.Errorf("Event %d: codex.ParseEvent returned nil", i)
			continue
		}

		switch i {
		case 0:
			if parsed.Type != "message" && parsed.Type != "delta" {
				t.Errorf("Event 0: expected type message/delta, got %s", parsed.Type)
			}
		case 1:
			if parsed.Type != "reasoning" {
				t.Errorf("Event 1: expected type reasoning, got %s", parsed.Type)
			}
		case 2:
			if parsed.Type != "tool_call" && parsed.Type != "tool_result" {
				t.Errorf("Event 2: expected type tool_call/tool_result, got %s", parsed.Type)
			}
		}
	}
}

func TestRunCodexSimple(t *testing.T) {
	// This is an integration test that requires the codex CLI to be installed
	// Skip if codex is not available
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex CLI not found in PATH, skipping integration test")
	}

	// Test with a simple prompt
	prompt := "Say 'test passed' and nothing else"
	content, err := RunCodexSimple(prompt, false)

	if err != nil {
		t.Errorf("RunCodexSimple() error = %v", err)
		return
	}

	if content == "" {
		t.Error("RunCodexSimple() returned empty content")
	}

	// Content should contain "test passed" (case insensitive)
	if !strings.Contains(strings.ToLower(content), "test passed") {
		t.Errorf("RunCodexSimple() content = %q, expected to contain 'test passed'", content)
	}
}

