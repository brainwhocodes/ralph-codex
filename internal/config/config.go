package config

// Config holds unified configuration for Ralph Codex
type Config struct {
	Backend      string
	ProjectPath  string
	PromptPath   string
	MaxCalls     int
	Timeout      int
	Verbose      bool
	ResetCircuit bool
}
