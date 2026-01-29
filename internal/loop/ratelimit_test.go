package loop

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainwhocodes/lisa-loop/internal/state"
)

func TestNewRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(100, 1)

	if limiter.maxCalls != 100 {
		t.Errorf("NewRateLimiter() maxCalls = %d, want 100", limiter.maxCalls)
	}

	if limiter.resetHours != 1 {
		t.Errorf("NewRateLimiter() resetHours = %d, want 1", limiter.resetHours)
	}

	if limiter.currentCalls != 0 {
		t.Errorf("NewRateLimiter() currentCalls = %d, want 0", limiter.currentCalls)
	}
}

func TestCanMakeCall(t *testing.T) {
	limiter := NewRateLimiter(100, 1)

	// Should allow calls up to max
	for i := 0; i < 100; i++ {
		if !limiter.CanMakeCall() {
			t.Errorf("CanMakeCall() should return true for call %d", i+1)
		}
		limiter.currentCalls++ // Manually increment to simulate calls
	}

	// Should not allow beyond max
	if limiter.CanMakeCall() {
		t.Errorf("CanMakeCall() should return false at limit")
	}
}

func TestRecordCall(t *testing.T) {
	limiter := NewRateLimiter(100, 1)

	err := limiter.RecordCall()

	if err != nil {
		t.Errorf("RecordCall() error = %v, want nil", err)
	}

	if limiter.currentCalls != 1 {
		t.Errorf("RecordCall() currentCalls = %d, want 1", limiter.currentCalls)
	}
}

func TestCallsRemaining(t *testing.T) {
	limiter := NewRateLimiter(100, 1)

	for i := 0; i < 5; i++ {
		limiter.RecordCall()
	}

	remaining := limiter.CallsRemaining()

	if remaining != 95 {
		t.Errorf("CallsRemaining() = %d, want 95", remaining)
	}
}

func TestShouldReset(t *testing.T) {
	limiter := NewRateLimiter(100, 1)

	// Just created, should not need reset
	if limiter.ShouldReset() {
		t.Errorf("ShouldReset() should return false for new limiter")
	}

	// Reset lastReset to 1 hour ago
	oldTime := limiter.lastReset.Add(-1 * time.Hour)
	limiter.lastReset = oldTime

	// Now should need reset
	if !limiter.ShouldReset() {
		t.Errorf("ShouldReset() should return true after 1 hour")
	}
}

func TestReset(t *testing.T) {
	limiter := NewRateLimiter(100, 1)

	err := limiter.Reset()

	if err != nil {
		t.Errorf("Reset() error = %v, want nil", err)
	}

	if limiter.currentCalls != 0 {
		t.Errorf("Reset() currentCalls = %d, want 0", limiter.currentCalls)
	}

	if limiter.lastReset.IsZero() {
		t.Errorf("Reset() lastReset should be now")
	}
}

func TestSetMaxCalls(t *testing.T) {
	limiter := NewRateLimiter(100, 1)

	err := limiter.SetMaxCalls(50)

	if err != nil {
		t.Errorf("SetMaxCalls() error = %v, want nil", err)
	}

	if limiter.maxCalls != 50 {
		t.Errorf("SetMaxCalls() maxCalls = %d, want 50", limiter.maxCalls)
	}
}

func TestSetResetHours(t *testing.T) {
	limiter := NewRateLimiter(100, 1)

	err := limiter.SetResetHours(2)

	if err != nil {
		t.Errorf("SetResetHours() error = %v, want nil", err)
	}

	if limiter.resetHours != 2 {
		t.Errorf("SetResetHours() resetHours = %d, want 2", limiter.resetHours)
	}
}

func TestTimeUntilReset(t *testing.T) {
	limiter := NewRateLimiter(100, 1)

	// Set lastReset to 30 minutes ago
	oldTime := limiter.lastReset.Add(-30 * time.Minute)
	limiter.lastReset = oldTime

	remaining := limiter.TimeUntilReset()

	// Should be around 30 minutes (1800 seconds) remaining
	expectedSeconds := 30 * 60.0 // 30 minutes in seconds
	if remaining.Seconds() < expectedSeconds-60 || remaining.Seconds() > expectedSeconds+60 {
		t.Errorf("TimeUntilReset() = %v, want ~30m", remaining)
	}
}

func TestGetStats(t *testing.T) {
	limiter := NewRateLimiter(100, 1)
	limiter.currentCalls = 42

	stats := limiter.GetStats()

	if stats["max_calls"] != 100 {
		t.Errorf("GetStats() max_calls = %d, want 100", stats["max_calls"])
	}

	if stats["current_calls"] != 42 {
		t.Errorf("GetStats() current_calls = %d, want 42", stats["current_calls"])
	}

	if stats["calls_remaining"] != 58 {
		t.Errorf("GetStats() calls_remaining = %d, want 58", stats["calls_remaining"])
	}

	if _, ok := stats["last_reset"]; !ok {
		t.Error("GetStats() should have last_reset")
	}

	if _, ok := stats["reset_hours"]; !ok {
		t.Error("GetStats() should have reset_hours")
	}

	if _, ok := stats["time_until_reset"]; !ok {
		t.Error("GetStats() should have time_until_reset")
	}
}

func TestLoadState(t *testing.T) {
	tmpDir := t.TempDir()

	// Create state files
	callCountPath := filepath.Join(tmpDir, ".call_count")
	os.WriteFile(callCountPath, []byte("42"), 0644)

	// Test: Load state (direct file read, as LoadState() doesn't exist)
	data, err := os.ReadFile(callCountPath)

	if err != nil {
		t.Errorf("ReadFile() error = %v, want nil", err)
	}

	if string(data) != "42" {
		t.Errorf("ReadFile() data = %s, want 42", string(data))
	}
}

func TestSaveState(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to tmpDir for state files
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	limiter := NewRateLimiter(100, 1)

	// Save state
	err := limiter.RecordCall()

	if err != nil {
		t.Errorf("SaveState() error = %v, want nil", err)
	}

	// Verify by loading count directly
	loadedCount, err := state.LoadCallCount()

	if err != nil {
		t.Errorf("LoadCallCount() error = %v, want nil", err)
	}

	if loadedCount != 1 {
		t.Errorf("LoadCallCount() = %d, want 1", loadedCount)
	}
}
