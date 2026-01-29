package runner

import (
	"github.com/brainwhocodes/lisa-loop/internal/codex"
	"github.com/brainwhocodes/lisa-loop/internal/config"
	"github.com/brainwhocodes/lisa-loop/internal/opencode"
)

// Event is a generic event type for streaming output
type Event = map[string]interface{}

// OutputCallback is called for streaming output events
type OutputCallback func(event Event)

// Runner is the interface for executing prompts
type Runner interface {
	// Run executes a prompt and returns the output, session ID, and any error
	Run(prompt string) (output string, sessionID string, err error)

	// SetOutputCallback sets the callback for streaming output events
	SetOutputCallback(cb OutputCallback)

	// Stop cleans up any resources (e.g., managed servers)
	Stop() error
}

// New creates a new runner based on the config backend setting
func New(cfg config.Config) Runner {
	switch cfg.Backend {
	case "opencode":
		return &openCodeWrapper{runner: opencode.NewRunner(cfg)}
	default:
		// Default to codex CLI backend
		return &codexWrapper{runner: codex.NewRunner(codex.Config(cfg))}
	}
}

// codexWrapper wraps codex.Runner to implement the Runner interface
type codexWrapper struct {
	runner *codex.Runner
}

func (w *codexWrapper) Run(prompt string) (string, string, error) {
	return w.runner.Run(prompt)
}

func (w *codexWrapper) SetOutputCallback(cb OutputCallback) {
	w.runner.SetOutputCallback(func(event codex.Event) {
		cb(Event(event))
	})
}

func (w *codexWrapper) Stop() error {
	return nil // Codex CLI doesn't need cleanup
}

// openCodeWrapper wraps opencode.Runner to implement the Runner interface
type openCodeWrapper struct {
	runner *opencode.Runner
}

func (w *openCodeWrapper) Run(prompt string) (string, string, error) {
	return w.runner.Run(prompt)
}

func (w *openCodeWrapper) SetOutputCallback(cb OutputCallback) {
	w.runner.SetOutputCallback(func(event map[string]interface{}) {
		cb(Event(event))
	})
}

func (w *openCodeWrapper) Stop() error {
	return w.runner.Stop()
}
