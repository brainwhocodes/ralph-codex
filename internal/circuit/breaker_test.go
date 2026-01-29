package circuit

import (
	"os"
	"testing"
	"time"

	"github.com/brainwhocodes/lisa-loop/internal/state"
)

func TestNewBreaker(t *testing.T) {
	breaker := NewBreaker(3, 5)

	if breaker.state != StateClosed {
		t.Errorf("NewBreaker() state = %s, want CLOSED", breaker.state)
	}

	if breaker.noProgressThreshold != 3 {
		t.Errorf("NewBreaker() noProgressThreshold = %d, want 3", breaker.noProgressThreshold)
	}

	if breaker.sameErrorThreshold != 5 {
		t.Errorf("NewBreaker() sameErrorThreshold = %d, want 5", breaker.sameErrorThreshold)
	}

	if breaker.noProgressCount != 0 {
		t.Errorf("NewBreaker() noProgressCount = %d, want 0", breaker.noProgressCount)
	}
}

func TestRecordResultWithProgress(t *testing.T) {
	breaker := NewBreaker(3, 5)

	err := breaker.RecordResult(1, 5, false)

	if err != nil {
		t.Errorf("RecordResult() error = %v, want nil", err)
	}

	if breaker.noProgressCount != 0 {
		t.Errorf("RecordResult() should reset noProgressCount on progress")
	}
}

func TestRecordResultNoProgress(t *testing.T) {
	breaker := NewBreaker(3, 5)

	err := breaker.RecordResult(1, 0, false)

	if err != nil {
		t.Errorf("RecordResult() error = %v, want nil", err)
	}

	if breaker.noProgressCount != 1 {
		t.Errorf("RecordResult() noProgressCount = %d, want 1", breaker.noProgressCount)
	}
}

func TestNoProgressTriggerHalfOpen(t *testing.T) {
	breaker := NewBreaker(3, 5)

	breaker.RecordResult(1, 0, false)
	breaker.RecordResult(2, 0, false)
	breaker.RecordResult(3, 0, false)

	if breaker.state != StateHalfOpen {
		t.Errorf("RecordResult() should trigger HALF_OPEN at 3, got %s", breaker.state)
	}
}

func TestNoProgressTriggerOpen(t *testing.T) {
	breaker := NewBreaker(3, 5)

	for i := 0; i < 6; i++ {
		breaker.RecordResult(1, 0, false)
	}

	if breaker.state != StateOpen {
		t.Errorf("RecordResult() should trigger OPEN at 6, got %s", breaker.state)
	}
}

func TestRecordError(t *testing.T) {
	breaker := NewBreaker(3, 5)

	err := breaker.RecordError("test error")

	if err != nil {
		t.Errorf("RecordError() error = %v, want nil", err)
	}

	if len(breaker.sameErrorHistory) != 1 {
		t.Errorf("RecordError() history length = %d, want 1", len(breaker.sameErrorHistory))
	}

	if breaker.sameErrorHistory[0] != "test error" {
		t.Errorf("RecordError() history[0] = %s, want 'test error'", breaker.sameErrorHistory[0])
	}
}

func TestRepeatedErrorsTriggerHalfOpen(t *testing.T) {
	breaker := NewBreaker(3, 5)

	for i := 0; i < 5; i++ {
		breaker.RecordError("repeated error")
	}

	if breaker.state != StateHalfOpen {
		t.Errorf("RecordError() should trigger HALF_OPEN at 5, got %s", breaker.state)
	}
}

func TestRepeatedErrorsTriggerOpen(t *testing.T) {
	breaker := NewBreaker(3, 5)

	for i := 0; i < 10; i++ {
		breaker.RecordError("repeated error")
	}

	if breaker.state != StateOpen {
		t.Errorf("RecordError() should trigger OPEN at 10, got %s", breaker.state)
	}
}

func TestShouldHalt(t *testing.T) {
	breaker := NewBreaker(3, 5)

	if breaker.ShouldHalt() {
		t.Error("ShouldHalt() should return false when CLOSED")
	}

	for i := 0; i < 10; i++ {
		breaker.RecordError("test")
	}

	if !breaker.ShouldHalt() {
		t.Error("ShouldHalt() should return true when OPEN")
	}
}

func TestIsHalfOpen(t *testing.T) {
	breaker := NewBreaker(3, 5)

	if breaker.IsHalfOpen() {
		t.Error("IsHalfOpen() should return false initially")
	}

	breaker.RecordResult(1, 0, false)
	breaker.RecordResult(2, 0, false)
	breaker.RecordResult(3, 0, false)

	if !breaker.IsHalfOpen() {
		t.Error("IsHalfOpen() should return true after threshold")
	}
}

func TestIsOpen(t *testing.T) {
	breaker := NewBreaker(3, 5)

	if breaker.IsOpen() {
		t.Error("IsOpen() should return false initially")
	}

	for i := 0; i < 10; i++ {
		breaker.RecordError("test")
	}

	if !breaker.IsOpen() {
		t.Error("IsOpen() should return true after threshold")
	}
}

func TestReset(t *testing.T) {
	breaker := NewBreaker(3, 5)

	for i := 0; i < 10; i++ {
		breaker.RecordError("test")
	}

	err := breaker.Reset()

	if err != nil {
		t.Errorf("Reset() error = %v, want nil", err)
	}

	if breaker.state != StateClosed {
		t.Errorf("Reset() state = %s, want CLOSED", breaker.state)
	}

	if breaker.noProgressCount != 0 {
		t.Errorf("Reset() noProgressCount = %d, want 0", breaker.noProgressCount)
	}

	if len(breaker.sameErrorHistory) != 0 {
		t.Errorf("Reset() sameErrorHistory length = %d, want 0", len(breaker.sameErrorHistory))
	}
}

func TestGetStats(t *testing.T) {
	breaker := NewBreaker(3, 5)
	breaker.RecordResult(1, 2, false)
	breaker.RecordError("test error")

	stats := breaker.GetStats()

	if stats["state"] != StateClosed.String() {
		t.Errorf("GetStats() state = %s, want CLOSED", stats["state"])
	}

	if stats["no_progress_count"] != 0 {
		t.Errorf("GetStats() no_progress_count = %d, want 0", stats["no_progress_count"])
	}

	if stats["same_error_count"] != 1 {
		t.Errorf("GetStats() same_error_count = %d, want 1", stats["same_error_count"])
	}

	if _, ok := stats["last_check_time"]; !ok {
		t.Error("GetStats() should have last_check_time")
	}

	if stats["no_progress_threshold"] != 3 {
		t.Errorf("GetStats() no_progress_threshold = %d, want 3", stats["no_progress_threshold"])
	}

	if stats["same_error_threshold"] != 5 {
		t.Errorf("GetStats() same_error_threshold = %d, want 5", stats["same_error_threshold"])
	}
}

func TestLoadState(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to tmpDir BEFORE saving state
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	testState := map[string]interface{}{
		"state":             "HALF_OPEN",
		"no_progress_count": 2,
		"error_history":     []interface{}{"error1", "error1"},
		"last_check_time":   time.Now().Format(time.RFC3339),
	}

	state.SaveCircuitBreakerState(testState)

	loadedBreaker, err := LoadBreakerFromFile()

	if err != nil {
		t.Errorf("LoadBreakerFromFile() error = %v, want nil", err)
	}

	if loadedBreaker.state != StateHalfOpen {
		t.Errorf("LoadBreakerFromFile() state = %s, want HALF_OPEN", loadedBreaker.state)
	}

	if loadedBreaker.noProgressCount != 2 {
		t.Errorf("LoadBreakerFromFile() noProgressCount = %d, want 2", loadedBreaker.noProgressCount)
	}

	if len(loadedBreaker.sameErrorHistory) != 2 {
		t.Errorf("LoadBreakerFromFile() errorHistory length = %d, want 2", len(loadedBreaker.sameErrorHistory))
	}
}

func TestSaveState(t *testing.T) {
	breaker := NewBreaker(3, 5)

	breaker.RecordResult(1, 0, false)
	breaker.RecordError("test")

	err := breaker.SaveState()

	if err != nil {
		t.Errorf("SaveState() error = %v, want nil", err)
	}

	loadedBreaker, _ := LoadBreakerFromFile()

	if loadedBreaker.noProgressCount != breaker.noProgressCount {
		t.Errorf("SaveState() noProgressCount not persisted")
	}

	if len(loadedBreaker.sameErrorHistory) != len(breaker.sameErrorHistory) {
		t.Errorf("SaveState() errorHistory not persisted")
	}
}
