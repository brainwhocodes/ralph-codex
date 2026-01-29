package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadStateFile(t *testing.T) {
	// Setup: create a temporary state file
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "test_state.json")
	testData := []byte(`{"test": "data"}`)

	err := os.WriteFile(statePath, testData, 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Change to tmpDir for the test
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Test: Read the state file
	data, err := ReadStateFile("test_state.json")

	// Assert
	if err != nil {
		t.Errorf("ReadStateFile() error = %v, want nil", err)
	}

	if string(data) != string(testData) {
		t.Errorf("ReadStateFile() data = %s, want %s", string(data), string(testData))
	}
}

func TestWriteStateFileAtomic(t *testing.T) {
	// Setup: create temp directory
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	testPath := "test_atomic.json"
	testData := []byte(`{"atomic": "write"}`)

	// Test: Write state file atomically
	err := WriteStateFile(testPath, testData)

	// Assert
	if err != nil {
		t.Errorf("WriteStateFile() error = %v, want nil", err)
	}

	// Verify file exists (not .tmp)
	if _, err := os.Stat(testPath); err != nil {
		t.Errorf("WriteStateFile() did not create file: %v", err)
	}

	// Verify content
	data, _ := os.ReadFile(testPath)
	if string(data) != string(testData) {
		t.Errorf("WriteStateFile() content mismatch")
	}
}

func TestLoadCallCount(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Test: Load non-existent call count
	count, err := LoadCallCount()

	if err != nil {
		t.Errorf("LoadCallCount() error = %v, want nil", err)
	}

	if count != 0 {
		t.Errorf("LoadCallCount() count = %d, want 0", count)
	}
}

func TestSaveCallCount(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Test: Save call count
	err := SaveCallCount(42)

	if err != nil {
		t.Errorf("SaveCallCount() error = %v, want nil", err)
	}

	// Verify
	count, _ := LoadCallCount()
	if count != 42 {
		t.Errorf("SaveCallCount() count = %d, want 42", count)
	}
}

func TestLoadLastReset(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Test: Load non-existent last reset
	r, err := LoadLastReset()

	if err != nil {
		t.Errorf("LoadLastReset() error = %v, want nil", err)
	}

	if r.IsZero() {
		t.Errorf("LoadLastReset() time is zero, want now")
	}
}

func TestSaveLastReset(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Test: Save last reset with a specific time
	testTime := time.Now().Truncate(time.Second) // Truncate to avoid nanosecond precision issues
	err := SaveLastReset(testTime)

	if err != nil {
		t.Errorf("SaveLastReset() error = %v, want nil", err)
	}

	// Verify
	loaded, _ := LoadLastReset()
	// Compare at second precision due to JSON serialization
	if !loaded.Truncate(time.Second).Equal(testTime) {
		t.Errorf("SaveLastReset() time mismatch: got %v, want %v", loaded, testTime)
	}
}

func TestLoadCodexSession(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Test: Load non-existent session
	id, err := LoadCodexSession()

	if err != nil {
		t.Errorf("LoadCodexSession() error = %v, want nil", err)
	}

	if id != "" {
		t.Errorf("LoadCodexSession() id = %s, want empty", id)
	}
}

func TestSaveCodexSession(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Test: Save Codex session
	testID := "thread-abc-123"
	err := SaveCodexSession(testID)

	if err != nil {
		t.Errorf("SaveCodexSession() error = %v, want nil", err)
	}

	// Verify
	loaded, _ := LoadCodexSession()
	if loaded != testID {
		t.Errorf("SaveCodexSession() id = %s, want %s", loaded, testID)
	}
}

func TestLoadLisaSession(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Test: Load non-existent session
	sess, err := LoadLisaSession()

	if err != nil {
		t.Errorf("LoadLisaSession() error = %v, want nil", err)
	}

	if len(sess) != 0 {
		t.Errorf("LoadLisaSession() session = %v, want empty", sess)
	}
}

func TestSaveLisaSession(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Test: Save Lisa session
	testSess := map[string]interface{}{
		"key":    "value",
		"number": 42,
	}
	err := SaveLisaSession(testSess)

	if err != nil {
		t.Errorf("SaveLisaSession() error = %v, want nil", err)
	}

	// Verify
	loaded, _ := LoadLisaSession()
	if len(loaded) != len(testSess) {
		t.Errorf("SaveLisaSession() session mismatch")
	}
}

func TestLoadExitSignals(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Test: Load non-existent signals
	signals, err := LoadExitSignals()

	if err != nil {
		t.Errorf("LoadExitSignals() error = %v, want nil", err)
	}

	if len(signals) != 0 {
		t.Errorf("LoadExitSignals() signals = %v, want empty", signals)
	}
}

func TestSaveExitSignals(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Test: Save exit signals
	testSignals := []string{"signal1", "signal2", "signal3"}
	err := SaveExitSignals(testSignals)

	if err != nil {
		t.Errorf("SaveExitSignals() error = %v, want nil", err)
	}

	// Verify
	loaded, _ := LoadExitSignals()
	if len(loaded) != len(testSignals) {
		t.Errorf("SaveExitSignals() signals mismatch")
	}
}

func TestLoadCircuitBreakerState(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Test: Load non-existent circuit breaker state
	st, err := LoadCircuitBreakerState()

	if err != nil {
		t.Errorf("LoadCircuitBreakerState() error = %v, want nil", err)
	}

	// Should default to CLOSED state
	if st["state"] != "CLOSED" {
		t.Errorf("LoadCircuitBreakerState() state = %s, want CLOSED", st["state"])
	}
}

func TestSaveCircuitBreakerState(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Test: Save circuit breaker state
	testState := map[string]interface{}{
		"state":    "CLOSED",
		"counters": map[string]int{"errors": 0, "no_progress": 0},
	}
	err := SaveCircuitBreakerState(testState)

	if err != nil {
		t.Errorf("SaveCircuitBreakerState() error = %v, want nil", err)
	}

	// Verify
	loaded, _ := LoadCircuitBreakerState()
	if loaded["state"] != testState["state"] {
		t.Errorf("SaveCircuitBreakerState() state mismatch")
	}
}

func TestCleanupOldFiles(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Create some .tmp and .json files
	os.WriteFile("test1.tmp", []byte("data"), 0644)
	os.WriteFile("test2.tmp", []byte("data"), 0644)
	os.WriteFile("test3.json", []byte("data"), 0644)

	// Test: Cleanup
	err := CleanupOldFiles()

	if err != nil {
		t.Errorf("CleanupOldFiles() error = %v, want nil", err)
	}

	// Verify .tmp files removed, .json files kept
	if _, err := os.Stat("test1.tmp"); err == nil {
		t.Error("test1.tmp should have been removed")
	}

	if _, err := os.Stat("test2.tmp"); err == nil {
		t.Error("test2.tmp should have been removed")
	}

	if _, err := os.Stat("test3.json"); err != nil {
		t.Error("test3.json should have been kept")
	}
}

func TestAtomicWrite(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	// Test: AtomicWrite
	testPath := "atomic.json"
	testData := []byte(`{"test": "atomic"}`)
	err := AtomicWrite(testPath, testData)

	if err != nil {
		t.Errorf("AtomicWrite() error = %v, want nil", err)
	}

	// Verify
	data, _ := os.ReadFile(testPath)
	if string(data) != string(testData) {
		t.Errorf("AtomicWrite() content mismatch")
	}
}
