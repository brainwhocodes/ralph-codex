package loop

import (
	"context"
	"fmt"
	"time"

	"github.com/brainwhocodes/lisa-loop/internal/state"
)

// RateLimiter tracks API calls per hour
type RateLimiter struct {
	maxCalls     int
	resetHours   int
	currentCalls int
	lastReset    time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxCalls int, resetHours int) *RateLimiter {
	return &RateLimiter{
		maxCalls:     maxCalls,
		resetHours:   resetHours,
		currentCalls: 0,
		lastReset:    time.Now(),
	}
}

// CanMakeCall checks if another call can be made
func (r *RateLimiter) CanMakeCall() bool {
	return r.currentCalls < r.maxCalls
}

// RecordCall records a call and updates tracking
func (r *RateLimiter) RecordCall() error {
	// Check if we need to reset
	if r.ShouldReset() {
		if err := r.Reset(); err != nil {
			return fmt.Errorf("failed to reset rate limiter: %w", err)
		}
	}

	r.currentCalls++

	// Update last reset time to track when we made this call
	if r.currentCalls == 1 {
		r.lastReset = time.Now()
	}

	// Persist state
	if err := r.SaveState(); err != nil {
		return fmt.Errorf("failed to save rate limiter state: %w", err)
	}

	return nil
}

// CallsMade returns the number of calls made in current hour
func (r *RateLimiter) CallsMade() int {
	return r.currentCalls
}

// CallsRemaining returns the number of calls remaining in current hour
func (r *RateLimiter) CallsRemaining() int {
	return r.maxCalls - r.currentCalls
}

// LastResetTime returns the time of the last reset
func (r *RateLimiter) LastResetTime() time.Time {
	return r.lastReset
}

// ShouldReset checks if the rate limit should be reset
func (r *RateLimiter) ShouldReset() bool {
	return time.Since(r.lastReset).Hours() >= float64(r.resetHours)
}

// Reset resets the call counter
func (r *RateLimiter) Reset() error {
	r.currentCalls = 0
	r.lastReset = time.Now()

	return r.SaveState()
}

// TimeUntilReset returns time remaining until next reset
func (r *RateLimiter) TimeUntilReset() time.Duration {
	elapsed := time.Since(r.lastReset)
	resetWindow := time.Duration(r.resetHours) * time.Hour
	remaining := resetWindow - elapsed

	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

// LoadState loads rate limiter state from files while preserving configured limits
func (r *RateLimiter) LoadState() (*RateLimiter, error) {
	count, err := state.LoadCallCount()
	if err != nil {
		return nil, err
	}

	reset, err := state.LoadLastReset()
	if err != nil {
		return nil, err
	}

	// Preserve configured maxCalls and resetHours, only update runtime state
	return &RateLimiter{
		maxCalls:     r.maxCalls,   // Preserve configured value
		resetHours:   r.resetHours, // Preserve configured value
		currentCalls: count,
		lastReset:    reset,
	}, nil
}

// LoadStateInto loads persisted state into the current rate limiter instance
func (r *RateLimiter) LoadStateInto() error {
	count, err := state.LoadCallCount()
	if err != nil {
		return err
	}

	reset, err := state.LoadLastReset()
	if err != nil {
		return err
	}

	// Only update runtime state, preserve configured limits
	r.currentCalls = count
	r.lastReset = reset

	return nil
}

// SaveState saves rate limiter state to files
func (r *RateLimiter) SaveState() error {
	if err := state.SaveCallCount(r.currentCalls); err != nil {
		return fmt.Errorf("failed to save call count: %w", err)
	}

	if err := state.SaveLastReset(r.lastReset); err != nil {
		return fmt.Errorf("failed to save last reset: %w", err)
	}

	return nil
}

// WaitForReset waits for the next hour with countdown
func (r *RateLimiter) WaitForReset(ctx context.Context) error {
	remaining := r.TimeUntilReset()
	if remaining == 0 {
		// Already reset, just wait a bit
		time.Sleep(1 * time.Second)
		return nil
	}

	fmt.Printf("\nRate limit reached. Waiting %v until next hour...\n", remaining.Round(time.Second))

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			remaining = r.TimeUntilReset()
			if remaining <= 0 {
				fmt.Println("Rate limit reset!")
				return nil
			}
			remainingSec := int(remaining.Seconds()) + 1
			fmt.Printf("\r%2d seconds remaining...", remainingSec)

		case <-ctx.Done():
			return fmt.Errorf("wait for reset cancelled")
		}
	}
}

// ResetWithCountdown shows progress while waiting for reset
func (r *RateLimiter) ResetWithCountdown(ctx context.Context) error {
	fmt.Println("\nRate limit reached.")
	return r.WaitForReset(ctx)
}

// SetMaxCalls updates the maximum calls per hour
func (r *RateLimiter) SetMaxCalls(maxCalls int) error {
	r.maxCalls = maxCalls
	return r.SaveState()
}

// SetResetHours updates the reset interval in hours
func (r *RateLimiter) SetResetHours(hours int) error {
	r.resetHours = hours
	return r.SaveState()
}

// GetStats returns current rate limiter statistics
func (r *RateLimiter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"max_calls":        r.maxCalls,
		"current_calls":    r.currentCalls,
		"calls_remaining":  r.CallsRemaining(),
		"last_reset":       r.lastReset.Format(time.RFC3339),
		"reset_hours":      r.resetHours,
		"time_until_reset": r.TimeUntilReset().String(),
	}
}
