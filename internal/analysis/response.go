package analysis

import (
	"fmt"
	"regexp"
	"strings"
)

// OutputFormat represents Codex output format
type OutputFormat string

const (
	FormatJSON OutputFormat = "json"
	FormatText OutputFormat = "text"
)

// RALPHStatus represents a parsed RALPH_STATUS block
type RALPHStatus struct {
	Status         string
	CurrentTask    string // The task currently being worked on or just completed
	TasksCompleted int
	FilesModified  int
	TestsStatus    string
	WorkType       string
	ExitSignal     bool
	Recommendation string
}

// Analysis represents the result of analyzing Codex output
type Analysis struct {
	Format               OutputFormat
	Status               *RALPHStatus
	CompletionIndicators int
	ExitSignal           bool
	ConfidenceScore      float64
	HasErrors            bool
	ErrorMessages        []string
}

// Analyze analyzes Codex output and extracts status information
func Analyze(output string, exitSignals []string) (*Analysis, error) {
	format := DetectFormat(output)

	var status *RALPHStatus
	var completionCount int

	if format == FormatJSON {
		// JSON analysis
		status, completionCount = analyzeJSONOutput(output)
	} else {
		// Text analysis
		status, completionCount = analyzeTextOutput(output)
	}

	// Calculate confidence using the helper function
	confidenceScore := calculateConfidence(status, completionCount, output)

	// Extract error messages from output
	errorMessages := ExtractErrors(output)

	// Determine if there are errors based on status or extracted errors
	hasErrors := status.Status == "BLOCKED" || status.TestsStatus == "FAILING" || len(errorMessages) > 0

	return &Analysis{
		Format:               format,
		Status:               status,
		CompletionIndicators: completionCount,
		ExitSignal:           status.ExitSignal,
		ConfidenceScore:      confidenceScore,
		HasErrors:            hasErrors,
		ErrorMessages:        errorMessages,
	}, nil
}

// DetectFormat determines if output is JSON or text format
func DetectFormat(output string) OutputFormat {
	// Check if output starts with JSON structure
	trimmed := strings.TrimSpace(output)
	lines := strings.Split(trimmed, "\n")

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "{") || strings.HasPrefix(trimmedLine, "[") {
			return FormatJSON
		}

		// Check for RALPH_STATUS block in text format
		if strings.Contains(trimmedLine, "---RALPH_STATUS---") {
			return FormatText
		}
	}

	// Default to text
	return FormatText
}

// ParseRALPHStatus extracts status information from a RALPH_STATUS block
func ParseRALPHStatus(output string) *RALPHStatus {
	// Find RALPH_STATUS block
	statusBlockRegex := regexp.MustCompile(`(?s)---RALPH_STATUS---([\s\S]+?)---END_RALPH_STATUS---`)
	matches := statusBlockRegex.FindStringSubmatch(output)

	if len(matches) < 2 {
		return &RALPHStatus{
			Status:         "UNKNOWN",
			TasksCompleted: 0,
			FilesModified:  0,
			TestsStatus:    "UNKNOWN",
			WorkType:       "UNKNOWN",
			ExitSignal:     false,
			Recommendation: "",
		}
	}

	content := matches[1]

	// Parse key-value pairs
	status := &RALPHStatus{
		Status:         "UNKNOWN",
		CurrentTask:    "",
		TasksCompleted: 0,
		FilesModified:  0,
		TestsStatus:    "UNKNOWN",
		WorkType:       "UNKNOWN",
		ExitSignal:     false,
		Recommendation: "",
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split on first colon
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "STATUS":
			status.Status = value
		case "CURRENT_TASK":
			status.CurrentTask = value
		case "TASKS_COMPLETED_THIS_LOOP":
			// Parse number
			if n, err := parseNumber(value); err == nil {
				status.TasksCompleted = n
			}
		case "FILES_MODIFIED":
			if n, err := parseNumber(value); err == nil {
				status.FilesModified = n
			}
		case "TESTS_STATUS":
			status.TestsStatus = value
		case "WORK_TYPE":
			status.WorkType = value
		case "EXIT_SIGNAL":
			status.ExitSignal = strings.ToLower(value) == "true"
		case "RECOMMENDATION":
			status.Recommendation = value
		}
	}

	return status
}

// analyzeJSONOutput analyzes JSON-format output
func analyzeJSONOutput(output string) (*RALPHStatus, int) {
	// Try to find RALPH_STATUS in JSON
	status := ParseRALPHStatus(output)

	completionCount := 0

	// Look for completion keywords in JSON output
	completionKeywords := []string{
		"done", "complete", "finished", "ready",
		"all set", "all done", "all complete",
		"no more work", "nothing to do",
	}

	for _, keyword := range completionKeywords {
		if strings.Contains(strings.ToLower(output), keyword) {
			completionCount++
		}
	}

	// Also check RALPH_STATUS block for explicit status
	if status.Status == "COMPLETE" {
		completionCount += 3
	}

	return status, completionCount
}

// analyzeTextOutput analyzes text-format output
func analyzeTextOutput(output string) (*RALPHStatus, int) {
	status := ParseRALPHStatus(output)

	completionCount := DetectCompletionKeywords(output)

	return status, completionCount
}

// DetectCompletionKeywords counts completion indicator keywords
func DetectCompletionKeywords(text string) int {
	text = strings.ToLower(text)

	// Completion keywords (case-insensitive)
	keywords := []string{
		"done", "complete", "completed", "finished", "ready",
		"success", "all set", "all done", "all complete",
		"finished all", "ready to review", "no more work",
		"nothing to do", "completed successfully",
	}

	count := 0
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			count++
		}
	}

	return count
}

// calculateConfidence computes a confidence score for analysis
func calculateConfidence(status *RALPHStatus, completionCount int, output string) float64 {
	confidence := 0.5 // Base confidence

	// Boost for explicit EXIT_SIGNAL
	if status.ExitSignal {
		confidence += 0.4
	}

	// Boost for explicit COMPLETE status
	if status.Status == "COMPLETE" {
		confidence += 0.3
	}

	// Boost for multiple completion indicators
	if completionCount >= 3 {
		confidence += 0.2
	}

	// Reduce for BLOCKED status
	if status.Status == "BLOCKED" {
		confidence -= 0.3
	}

	// Clamp between 0 and 1
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// ExtractErrors extracts error messages from output
func ExtractErrors(output string) []string {
	lines := strings.Split(output, "\n")
	errors := []string{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Filter JSON field false positives
		line = filterJSONFieldErrors(line)

		// Check for error patterns
		if isErrorLine(line) {
			errors = append(errors, line)
		}
	}

	return errors
}

// isErrorLine checks if a line contains an error indicator
func isErrorLine(line string) bool {
	// Error prefixes
	errorPrefixes := []string{
		"Error:", "ERROR:", "error:", "Error occurred",
		"failed with error", "exception", "Exception",
		"Fatal", "FATAL",
	}

	for _, prefix := range errorPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}

	// Context-specific errors
	if strings.Contains(line, "] error") {
		return true
	}

	return false
}

// filterJSONFieldErrors removes JSON field lines that contain "error" as a field name
func filterJSONFieldErrors(line string) string {
	// Skip lines that look like: "is_error": false
	if strings.Contains(line, "\"is_error\": false") {
		return ""
	}

	return line
}

// parseNumber extracts a number from a string
func parseNumber(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
