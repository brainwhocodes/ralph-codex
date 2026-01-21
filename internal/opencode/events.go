package opencode

// EventType constants for OpenCode SSE events
const (
	EventTypeMessage   = "message"
	EventTypeReasoning = "reasoning"
	EventTypeToolUse   = "tool_use"
	EventTypeLifecycle = "lifecycle"
)

// ParsedEvent represents a parsed OpenCode SSE event
type ParsedEvent struct {
	Type       string // message, reasoning, tool_use, lifecycle
	Text       string // Text content for message/reasoning
	ToolName   string // For tool events
	ToolTarget string // File path or command
	ToolStatus string // started, completed
	RawType    string // Original event type
}

// ParseEvent extracts meaningful content from an OpenCode event
func ParseEvent(event map[string]interface{}) *ParsedEvent {
	if event == nil {
		return nil
	}

	result := &ParsedEvent{
		Type: "unknown",
	}

	// Get the event type
	eventType, _ := event["type"].(string)
	result.RawType = eventType

	switch eventType {
	case "message":
		result.Type = EventTypeMessage
		if content, ok := event["content"].(string); ok {
			result.Text = content
		}

	case "item.completed":
		parseItemCompleted(event, result)

	case "tool_use":
		result.Type = EventTypeToolUse
		result.ToolName, _ = event["name"].(string)
		result.ToolTarget, _ = event["target"].(string)
		if status, ok := event["status"].(string); ok {
			result.ToolStatus = status
		} else {
			result.ToolStatus = "started"
		}

	default:
		// Lifecycle or unknown event
		result.Type = EventTypeLifecycle
	}

	return result
}

// parseItemCompleted handles item.completed events from OpenCode
func parseItemCompleted(event map[string]interface{}, result *ParsedEvent) {
	item, ok := event["item"].(map[string]interface{})
	if !ok {
		return
	}

	itemType, _ := item["type"].(string)
	text, _ := item["text"].(string)

	switch itemType {
	case "reasoning":
		result.Type = EventTypeReasoning
		result.Text = text
	case "message", "agent_message":
		result.Type = EventTypeMessage
		result.Text = text
	default:
		if text != "" {
			result.Type = EventTypeMessage
			result.Text = text
		}
	}
}
