package runner

import (
	"testing"

	"github.com/brainwhocodes/lisa-loop/internal/config"
)

func TestNew_DefaultsToCodex(t *testing.T) {
	cfg := config.Config{
		Backend: "",
	}

	r := New(cfg)

	// Verify it returns a runner (codexWrapper)
	if r == nil {
		t.Error("expected runner to be created")
	}

	// Verify it's the codex wrapper by checking type
	_, ok := r.(*codexWrapper)
	if !ok {
		t.Error("expected codexWrapper for empty backend")
	}
}

func TestNew_CodexBackend(t *testing.T) {
	cfg := config.Config{
		Backend: "cli",
	}

	r := New(cfg)

	_, ok := r.(*codexWrapper)
	if !ok {
		t.Error("expected codexWrapper for cli backend")
	}
}

func TestNew_OpenCodeBackend(t *testing.T) {
	cfg := config.Config{
		Backend:           "opencode",
		OpenCodeServerURL: "http://localhost:8080",
		OpenCodeUsername:  "opencode",
		OpenCodePassword:  "secret",
		OpenCodeModelID:   "glm-4.7",
	}

	r := New(cfg)

	_, ok := r.(*openCodeWrapper)
	if !ok {
		t.Error("expected openCodeWrapper for opencode backend")
	}
}

func TestRunner_SetOutputCallback(t *testing.T) {
	cfg := config.Config{
		Backend: "cli",
	}

	r := New(cfg)

	r.SetOutputCallback(func(event Event) {
		// Callback set successfully
	})

	// Just verify we can set the callback without error
	if r == nil {
		t.Error("expected runner to be created")
	}
}

func TestRunner_SetOutputCallback_OpenCode(t *testing.T) {
	cfg := config.Config{
		Backend:           "opencode",
		OpenCodeServerURL: "http://localhost:8080",
	}

	r := New(cfg)

	r.SetOutputCallback(func(event Event) {
		// Callback set successfully
	})

	// Just verify we can set the callback without error
	if r == nil {
		t.Error("expected runner to be created")
	}
}
