package codex

// ParsedEvent represents a parsed Codex event with extracted content
type ParsedEvent struct {
	Type        string // "reasoning", "message", "tool_call", "tool_result", "delta", "lifecycle", "unknown"
	Text        string // Extracted text content
	ToolName    string // For tool events
	ToolTarget  string // File path or command for tool events
	ToolStatus  string // "started" or "completed"
	RawType     string // Original event type from JSON
}

// ParseEvent extracts meaningful content from a Codex JSONL event
// This unifies the parsing logic used across the codebase
func ParseEvent(event Event) *ParsedEvent {
	if event == nil {
		return nil
	}

	result := &ParsedEvent{
		Type:    "unknown",
		RawType: MessageType(event),
	}

	msgType := MessageType(event)

	switch msgType {
	case "item.completed":
		parseItemCompleted(event, result)

	case "content_block_delta":
		parseDelta(event, result)

	case "message":
		parseMessage(event, result)

	case "assistant":
		parseAssistant(event, result)

	case "tool_use":
		parseToolUse(event, result)
		// Check for explicit status, default to "started"
		if status, ok := event["status"].(string); ok && status != "" {
			result.ToolStatus = status
		} else {
			result.ToolStatus = "started"
		}

	case "tool_result":
		parseToolResult(event, result)
		result.ToolStatus = "completed"

	default:
		// Try to extract any text
		if text := MessageText(event); text != "" {
			result.Type = "message"
			result.Text = text
		} else {
			result.Type = "lifecycle"
		}
	}

	// Also check for direct message field
	if msg, ok := event["message"].(string); ok && msg != "" && result.Text == "" {
		result.Type = "message"
		result.Text = msg
	}

	return result
}

// parseItemCompleted handles item.completed events
func parseItemCompleted(event Event, result *ParsedEvent) {
	item, ok := event["item"].(map[string]interface{})
	if !ok {
		return
	}

	itemType, _ := item["type"].(string)
	text, _ := item["text"].(string)

	switch itemType {
	case "reasoning":
		result.Type = "reasoning"
		result.Text = text
	case "agent_message", "message":
		result.Type = "message"
		result.Text = text
	case "tool_call", "function_call":
		result.Type = "tool_call"
		result.ToolName, _ = item["name"].(string)
		result.ToolTarget = extractToolTarget(item)
		result.ToolStatus = "completed"
	default:
		if text != "" {
			result.Type = "message"
			result.Text = text
		}
	}
}

// parseDelta handles content_block_delta events
func parseDelta(event Event, result *ParsedEvent) {
	if delta, ok := event["delta"].(map[string]interface{}); ok {
		if text, ok := delta["text"].(string); ok && text != "" {
			result.Type = "delta"
			result.Text = text
		}
	}
}

// parseMessage handles message events
func parseMessage(event Event, result *ParsedEvent) {
	// Try direct content string
	if content, ok := event["content"].(string); ok && content != "" {
		result.Type = "message"
		result.Text = content
		return
	}

	// Try content array
	if contentArr, ok := event["content"].([]interface{}); ok {
		result.Text = extractTextFromContentArray(contentArr)
		if result.Text != "" {
			result.Type = "message"
		}
	}
}

// parseAssistant handles assistant events
func parseAssistant(event Event, result *ParsedEvent) {
	if contentArr, ok := event["content"].([]interface{}); ok {
		result.Text = extractTextFromContentArray(contentArr)
		if result.Text != "" {
			result.Type = "message"
		}
	}
}

// parseToolUse handles tool_use events
func parseToolUse(event Event, result *ParsedEvent) {
	result.Type = "tool_call"
	result.ToolName, _ = event["name"].(string)
	result.ToolTarget = extractToolTarget(event)
}

// parseToolResult handles tool_result events
func parseToolResult(event Event, result *ParsedEvent) {
	result.Type = "tool_result"
	result.ToolName, _ = event["name"].(string)

	// Try to get tool name from tool_use_id or nested structure
	if result.ToolName == "" {
		if toolUse, ok := event["tool_use"].(map[string]interface{}); ok {
			result.ToolName, _ = toolUse["name"].(string)
		}
	}

	result.ToolTarget = extractToolTarget(event)
}

// extractTextFromContentArray extracts text from a content array
func extractTextFromContentArray(contentArr []interface{}) string {
	var text string
	for _, item := range contentArr {
		if contentMap, ok := item.(map[string]interface{}); ok {
			if t, ok := contentMap["text"].(string); ok && t != "" {
				text += t
			}
		}
	}
	return text
}

// extractToolTarget extracts the target (file path or command) from tool data
func extractToolTarget(data map[string]interface{}) string {
	// Try direct target field first (used by OpenCode runner)
	if target, ok := data["target"].(string); ok && target != "" {
		return target
	}

	// Try arguments field
	if args, ok := data["arguments"].(map[string]interface{}); ok {
		if target := extractTargetFromArgs(args); target != "" {
			return target
		}
	}

	// Try input field
	if input, ok := data["input"].(map[string]interface{}); ok {
		if target := extractTargetFromArgs(input); target != "" {
			return target
		}
	}

	// Try parameters field
	if params, ok := data["parameters"].(map[string]interface{}); ok {
		if target := extractTargetFromArgs(params); target != "" {
			return target
		}
	}

	return ""
}

// extractTargetFromArgs extracts target from argument map
func extractTargetFromArgs(args map[string]interface{}) string {
	// Common argument patterns for file paths (both snake_case and camelCase)
	for _, key := range []string{"file_path", "filePath", "path", "filename", "file"} {
		if path, ok := args[key].(string); ok && path != "" {
			return path
		}
	}

	// Command execution
	if cmd, ok := args["command"].(string); ok && cmd != "" {
		if len(cmd) > 50 {
			return cmd[:50] + "..."
		}
		return cmd
	}

	return ""
}

// EventCallback is a callback function for streaming events
type StreamCallback func(parsed *ParsedEvent)

// ProcessEventStream processes a stream of events and calls the callback for each
func ProcessEventStream(events []Event, callback StreamCallback) {
	for _, event := range events {
		if parsed := ParseEvent(event); parsed != nil {
			callback(parsed)
		}
	}
}
