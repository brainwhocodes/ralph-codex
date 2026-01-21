package circuit

import (
	"fmt"
	"time"

	"github.com/brainwhocodes/ralph-codex/internal/state"
)

// State represents circuit breaker state
type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateHalfOpen:
		return "HALF_OPEN"
	case StateOpen:
		return "OPEN"
	default:
		return "UNKNOWN"
	}
}

// Breaker implements circuit breaker pattern
type Breaker struct {
	state               State
	noProgressThreshold int
	noProgressCount     int
	sameErrorThreshold  int
	sameErrorHistory    []string
	lastCheckTime       time.Time
}

// NewBreaker creates a new circuit breaker
func NewBreaker(noProgressThreshold int, sameErrorThreshold int) *Breaker {
	return &Breaker{
		state:               StateClosed,
		noProgressThreshold: noProgressThreshold,
		sameErrorThreshold:  sameErrorThreshold,
		noProgressCount:     0,
		sameErrorHistory:    []string{},
		lastCheckTime:       time.Now(),
	}
}

// RecordResult records a loop result
func (b *Breaker) RecordResult(loopNum int, filesChanged int, hasErrors bool) error {
	// Update check time
	b.lastCheckTime = time.Now()

	// Check for no progress
	if filesChanged == 0 {
		b.noProgressCount++

		// Trigger HALF_OPEN if threshold reached
		if b.noProgressCount >= b.noProgressThreshold && b.state == StateClosed {
			b.state = StateHalfOpen
			return b.SaveState()
		}

		// Trigger OPEN if threshold exceeded
		if b.noProgressCount >= b.noProgressThreshold*2 {
			b.state = StateOpen
			return b.SaveState()
		}
	} else {
		// Reset no-progress counter on progress
		b.noProgressCount = 0
	}

	// Note: Error tracking is handled by RecordError() method
	// hasErrors parameter is used for state tracking only

	return b.SaveState()
}

// RecordError records an error for repeated error detection
func (b *Breaker) RecordError(errorMsg string) error {
	if errorMsg == "" {
		return nil
	}

	b.sameErrorHistory = append(b.sameErrorHistory, errorMsg)
	totalErrors := len(b.sameErrorHistory)

	// Trigger OPEN if threshold exceeded (check before capping)
	if totalErrors >= b.sameErrorThreshold*2 {
		b.state = StateOpen
		// Keep only last N errors for history
		maxHistory := b.sameErrorThreshold * 2
		if len(b.sameErrorHistory) > maxHistory {
			b.sameErrorHistory = b.sameErrorHistory[len(b.sameErrorHistory)-maxHistory:]
		}
		return b.SaveState()
	}

	// Trigger HALF_OPEN if threshold reached
	if totalErrors >= b.sameErrorThreshold && b.state == StateClosed {
		b.state = StateHalfOpen
		return b.SaveState()
	}

	// Keep only last N errors
	maxHistory := b.sameErrorThreshold * 2
	if len(b.sameErrorHistory) > maxHistory {
		b.sameErrorHistory = b.sameErrorHistory[len(b.sameErrorHistory)-maxHistory:]
	}

	return b.SaveState()
}

// GetState returns the current circuit state
func (b *Breaker) GetState() State {
	return b.state
}

// ShouldHalt checks if the circuit should halt execution
func (b *Breaker) ShouldHalt() bool {
	return b.state == StateOpen
}

// IsHalfOpen checks if circuit is in HALF_OPEN state
func (b *Breaker) IsHalfOpen() bool {
	return b.state == StateHalfOpen
}

// IsOpen checks if circuit is in OPEN state
func (b *Breaker) IsOpen() bool {
	return b.state == StateOpen
}

// IsClosed checks if circuit is in CLOSED state
func (b *Breaker) IsClosed() bool {
	return b.state == StateClosed
}

// Reset resets the circuit to CLOSED state
func (b *Breaker) Reset() error {
	b.state = StateClosed
	b.noProgressCount = 0
	b.sameErrorHistory = []string{}
	b.lastCheckTime = time.Now()

	return b.SaveState()
}

// LoadState loads circuit breaker state from file
func (b *Breaker) LoadState() (*Breaker, error) {
	stateMap, err := state.LoadCircuitBreakerState()
	if err != nil {
		return nil, err
	}

	loaded := NewBreaker(3, 5)

	if stateVal, ok := stateMap["state"].(string); ok {
		switch stateVal {
		case "CLOSED":
			loaded.state = StateClosed
		case "HALF_OPEN":
			loaded.state = StateHalfOpen
		case "OPEN":
			loaded.state = StateOpen
		}
	}

	if lastCheck, ok := stateMap["last_check_time"].(string); ok {
		loaded.lastCheckTime, _ = time.Parse(time.RFC3339, lastCheck)
	}

	// Load counters
	if noProg, ok := stateMap["no_progress_count"].(float64); ok {
		loaded.noProgressCount = int(noProg)
	}

	if errHist, ok := stateMap["error_history"].([]interface{}); ok {
		loaded.sameErrorHistory = make([]string, len(errHist))
		for i, err := range errHist {
			if errStr, ok := err.(string); ok {
				loaded.sameErrorHistory[i] = errStr
			}
		}
	}

	return loaded, nil
}

// SaveState saves circuit breaker state to file
func (b *Breaker) SaveState() error {
	stateMap := map[string]interface{}{
		"state":             b.state.String(),
		"no_progress_count": b.noProgressCount,
		"error_history":     b.sameErrorHistory,
		"last_check_time":   b.lastCheckTime.Format(time.RFC3339),
	}

	if err := state.SaveCircuitBreakerState(stateMap); err != nil {
		return fmt.Errorf("failed to save circuit breaker state: %w", err)
	}

	return nil
}

// GetStats returns circuit breaker statistics
func (b *Breaker) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"state":                 b.state.String(),
		"no_progress_count":     b.noProgressCount,
		"same_error_count":      len(b.sameErrorHistory),
		"last_check_time":       b.lastCheckTime.Format(time.RFC3339),
		"no_progress_threshold": b.noProgressThreshold,
		"same_error_threshold":  b.sameErrorThreshold,
	}
}

// CheckNoProgress checks if we've hit the no-progress threshold
func (b *Breaker) CheckNoProgress() bool {
	return b.noProgressCount >= b.noProgressThreshold
}

// CheckRepeatedErrors checks if we've hit the repeated error threshold
func (b *Breaker) CheckRepeatedErrors() bool {
	return len(b.sameErrorHistory) >= b.sameErrorThreshold
}

// GetNoProgressCount returns the current no-progress counter
func (b *Breaker) GetNoProgressCount() int {
	return b.noProgressCount
}

// GetErrorHistory returns the error history
func (b *Breaker) GetErrorHistory() []string {
	return b.sameErrorHistory
}

// LoadBreakerFromFile loads a circuit breaker from the state file
func LoadBreakerFromFile() (*Breaker, error) {
	breaker := NewBreaker(3, 5)
	return breaker.LoadState()
}
