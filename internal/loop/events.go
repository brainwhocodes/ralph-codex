package loop

// EventType represents the type of a loop event
type EventType string

// Event types emitted by the loop controller
const (
	EventTypeLoopUpdate     EventType = "loop_update"
	EventTypeLog            EventType = "log"
	EventTypeStateChange    EventType = "state_change"
	EventTypeStatus         EventType = "status"
	EventTypeCodexOutput    EventType = "codex_output"
	EventTypeCodexReasoning EventType = "codex_reasoning"
	EventTypeCodexTool      EventType = "codex_tool"
	EventTypeAnalysis       EventType = "analysis"      // RALPH_STATUS analysis results
	EventTypeContextUsage   EventType = "context_usage" // Context window usage tracking
	EventTypePreflight      EventType = "preflight"     // Preflight check summary
	EventTypeOutcome        EventType = "outcome"       // Loop iteration outcome
)

// LogLevel represents the severity level of a log entry
type LogLevel string

// Log levels used throughout the loop controller
const (
	LogLevelDebug   LogLevel = "DEBUG"
	LogLevelInfo    LogLevel = "INFO"
	LogLevelWarn    LogLevel = "WARN"
	LogLevelError   LogLevel = "ERROR"
	LogLevelSuccess LogLevel = "SUCCESS"
)

// OutputType represents the type of codex output
type OutputType string

// Output types for codex streaming events
const (
	OutputTypeReasoning    OutputType = "reasoning"
	OutputTypeAgentMessage OutputType = "agent_message"
	OutputTypeToolCall     OutputType = "tool_call"
	OutputTypeRaw          OutputType = "raw"
)

// ToolStatus represents the status of a tool execution
type ToolStatus string

// Tool execution statuses
const (
	ToolStatusStarted   ToolStatus = "started"
	ToolStatusCompleted ToolStatus = "completed"
)
