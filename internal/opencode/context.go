package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Default context window sizes for known models (in tokens)
var ModelContextLimits = map[string]int{
	"glm-4.7":                 128000,
	"glm-4":                   128000,
	"claude-3-opus":           200000,
	"claude-3-sonnet":         200000,
	"claude-3-haiku":          200000,
	"claude-3.5-sonnet":       200000,
	"claude-opus-4":           200000,
	"gpt-4":                   128000,
	"gpt-4-turbo":             128000,
	"gpt-4o":                  128000,
	"o1":                      200000,
	"o1-mini":                 128000,
	"anthropic/claude-sonnet": 200000,
	"anthropic/claude-opus":   200000,
}

// DefaultContextLimit is used when model is not in the map
const DefaultContextLimit = 128000

// DefaultSaveThreshold is the percentage at which to auto-save (0.8 = 80%)
const DefaultSaveThreshold = 0.80

// ContextTracker monitors token usage and triggers auto-save
type ContextTracker struct {
	mu               sync.RWMutex
	modelID          string
	contextLimit     int
	saveThreshold    float64
	promptTokens     int
	completionTokens int
	wasCompacted     bool
	lastUpdate       time.Time
	onThreshold      func(usage ContextUsage) // Callback when threshold reached
}

// ContextUsage represents current context usage stats
type ContextUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	ContextLimit     int
	UsagePercent     float64
	ThresholdReached bool
	WasCompacted     bool
}

// NewContextTracker creates a new context tracker for a model
func NewContextTracker(modelID string) *ContextTracker {
	limit := DefaultContextLimit
	if l, ok := ModelContextLimits[modelID]; ok {
		limit = l
	}

	return &ContextTracker{
		modelID:       modelID,
		contextLimit:  limit,
		saveThreshold: DefaultSaveThreshold,
	}
}

// SetThreshold sets the save threshold (0.0-1.0)
func (ct *ContextTracker) SetThreshold(threshold float64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.saveThreshold = threshold
}

// SetOnThreshold sets the callback for when threshold is reached
func (ct *ContextTracker) SetOnThreshold(cb func(ContextUsage)) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.onThreshold = cb
}

// Update updates token counts and checks threshold
func (ct *ContextTracker) Update(promptTokens, completionTokens int, wasCompacted bool) ContextUsage {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.promptTokens = promptTokens
	ct.completionTokens = completionTokens
	ct.wasCompacted = wasCompacted
	ct.lastUpdate = time.Now()

	usage := ct.getUsageUnsafe()

	// Trigger callback if threshold reached
	if usage.ThresholdReached && ct.onThreshold != nil {
		go ct.onThreshold(usage)
	}

	return usage
}

// GetUsage returns current context usage
func (ct *ContextTracker) GetUsage() ContextUsage {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.getUsageUnsafe()
}

// getUsageUnsafe returns usage without locking (caller must hold lock)
func (ct *ContextTracker) getUsageUnsafe() ContextUsage {
	total := ct.promptTokens + ct.completionTokens
	percent := float64(total) / float64(ct.contextLimit)

	return ContextUsage{
		PromptTokens:     ct.promptTokens,
		CompletionTokens: ct.completionTokens,
		TotalTokens:      total,
		ContextLimit:     ct.contextLimit,
		UsagePercent:     percent,
		ThresholdReached: percent >= ct.saveThreshold,
		WasCompacted:     ct.wasCompacted,
	}
}

// Reset resets the tracker (after session save/new session)
func (ct *ContextTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.promptTokens = 0
	ct.completionTokens = 0
	ct.wasCompacted = false
}

// SessionArchive represents a saved session
type SessionArchive struct {
	SessionID        string        `json:"session_id"`
	ModelID          string        `json:"model_id"`
	PromptTokens     int           `json:"prompt_tokens"`
	CompletionTokens int           `json:"completion_tokens"`
	LoopNumber       int           `json:"loop_number"`
	SavedAt          time.Time     `json:"saved_at"`
	Reason           string        `json:"reason"` // "threshold", "compacted", "manual"
	Summary          string        `json:"summary,omitempty"`
	Tasks            []TaskStatus  `json:"tasks,omitempty"`
}

// TaskStatus represents a task's completion status
type TaskStatus struct {
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// SessionArchiver handles saving and loading session archives
type SessionArchiver struct {
	archiveDir string
}

// NewSessionArchiver creates a new archiver with the given directory
func NewSessionArchiver(projectDir string) *SessionArchiver {
	archiveDir := filepath.Join(projectDir, ".ralph", "sessions")
	return &SessionArchiver{archiveDir: archiveDir}
}

// EnsureDir creates the archive directory if it doesn't exist
func (sa *SessionArchiver) EnsureDir() error {
	return os.MkdirAll(sa.archiveDir, 0755)
}

// Save saves a session archive to file
func (sa *SessionArchiver) Save(archive SessionArchive) (string, error) {
	if err := sa.EnsureDir(); err != nil {
		return "", fmt.Errorf("failed to create archive dir: %w", err)
	}

	// Generate filename with timestamp
	timestamp := archive.SavedAt.Format("20060102_150405")
	filename := fmt.Sprintf("session_%s_%s.json", timestamp, archive.SessionID[:8])
	filepath := filepath.Join(sa.archiveDir, filename)

	data, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal archive: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write archive: %w", err)
	}

	return filepath, nil
}

// List returns all archived sessions
func (sa *SessionArchiver) List() ([]SessionArchive, error) {
	if err := sa.EnsureDir(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(sa.archiveDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read archive dir: %w", err)
	}

	var archives []SessionArchive
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(sa.archiveDir, entry.Name()))
		if err != nil {
			continue
		}

		var archive SessionArchive
		if err := json.Unmarshal(data, &archive); err != nil {
			continue
		}

		archives = append(archives, archive)
	}

	return archives, nil
}

// GetLatest returns the most recent archived session
func (sa *SessionArchiver) GetLatest() (*SessionArchive, error) {
	archives, err := sa.List()
	if err != nil {
		return nil, err
	}

	if len(archives) == 0 {
		return nil, nil
	}

	// Find most recent
	latest := &archives[0]
	for i := range archives {
		if archives[i].SavedAt.After(latest.SavedAt) {
			latest = &archives[i]
		}
	}

	return latest, nil
}
