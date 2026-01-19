package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LoadState is a generic helper for loading JSON state files
func LoadState[T any](filename string, defaultVal T) (T, error) {
	data, err := ReadStateFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultVal, nil
		}
		return defaultVal, err
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return defaultVal, fmt.Errorf("failed to parse %s: %w", filename, err)
	}
	return result, nil
}

// SaveState is a generic helper for saving JSON state files
func SaveState[T any](filename string, value T) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %w", filename, err)
	}
	return WriteStateFile(filename, data)
}

// ReadStateFile reads a JSON state file
func ReadStateFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file %s: %w", path, err)
	}
	return data, nil
}

// WriteStateFile writes data to a file atomically (write to temp, then rename)
func WriteStateFile(path string, data []byte) error {
	// Write to temporary file
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file %s: %w", tmpPath, err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file if rename fails
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename %s to %s: %w", tmpPath, path, err)
	}

	return nil
}

// AtomicWrite is a generic atomic write function
func AtomicWrite(path string, data []byte) error {
	return WriteStateFile(path, data)
}

// LoadCallCount loads the call count from .call_count
func LoadCallCount() (int, error) {
	return LoadState(".call_count", 0)
}

// SaveCallCount saves the call count to .call_count atomically
func SaveCallCount(count int) error {
	return SaveState(".call_count", count)
}

// LoadLastReset loads last reset time from .last_reset
func LoadLastReset() (time.Time, error) {
	return LoadState(".last_reset", time.Now())
}

// SaveLastReset saves last reset time to .last_reset atomically
func SaveLastReset(t time.Time) error {
	return SaveState(".last_reset", t)
}

// LoadCodexSession loads Codex session ID from .codex_session_id
func LoadCodexSession() (string, error) {
	data, err := os.ReadFile(".codex_session_id")
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// SaveCodexSession saves Codex session ID to .codex_session_id atomically
func SaveCodexSession(id string) error {
	return AtomicWrite(".codex_session_id", []byte(id))
}

// LoadRalphSession loads Ralph session metadata from .ralph_session
func LoadRalphSession() (map[string]interface{}, error) {
	return LoadState(".ralph_session", map[string]interface{}{})
}

// SaveRalphSession saves Ralph session metadata atomically
func SaveRalphSession(session map[string]interface{}) error {
	return SaveState(".ralph_session", session)
}

// LoadExitSignals loads recent exit signals from .exit_signals
func LoadExitSignals() ([]string, error) {
	return LoadState(".exit_signals", []string{})
}

// SaveExitSignals saves exit signals atomically
func SaveExitSignals(signals []string) error {
	return SaveState(".exit_signals", signals)
}

// LoadCircuitBreakerState loads circuit breaker state from .circuit_breaker_state
func LoadCircuitBreakerState() (map[string]interface{}, error) {
	defaultState := map[string]interface{}{
		"state":           "CLOSED",
		"last_check_time": time.Now().Format(time.RFC3339),
	}
	return LoadState(".circuit_breaker_state", defaultState)
}

// SaveCircuitBreakerState saves circuit breaker state atomically
func SaveCircuitBreakerState(state map[string]interface{}) error {
	return SaveState(".circuit_breaker_state", state)
}

// EnsureStateDir ensures the directory for state files exists
func EnsureStateDir() error {
	if err := os.MkdirAll(".", 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	return nil
}

// CleanupOldFiles removes old temporary state files
func CleanupOldFiles() error {
	entries, err := os.ReadDir(".")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" {
			os.Remove(filepath.Join(".", entry.Name()))
		}
	}

	return nil
}
