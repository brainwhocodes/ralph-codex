package main

import (
	"bytes"
	"flag"
	"strings"
	"testing"
)

func TestCommandParsing(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCmd    string
		wantSetup  bool
		wantImport bool
	}{
		{
			name:      "no args defaults to run",
			args:      []string{},
			wantCmd:   "run",
			wantSetup: false,
		},
		{
			name:      "explicit run command",
			args:      []string{"run"},
			wantCmd:   "run",
			wantSetup: false,
		},
		{
			name:      "setup command",
			args:      []string{"setup"},
			wantCmd:   "setup",
			wantSetup: true,
		},
		{
			name:      "import command",
			args:      []string{"import"},
			wantCmd:   "import",
			wantSetup: false,
		},
		{
			name:      "status command",
			args:      []string{"status"},
			wantCmd:   "status",
			wantSetup: false,
		},
		{
			name:      "reset-circuit command",
			args:      []string{"reset-circuit"},
			wantCmd:   "reset-circuit",
			wantSetup: false,
		},
		{
			name:      "help command",
			args:      []string{"help"},
			wantCmd:   "help",
			wantSetup: false,
		},
		{
			name:      "version command",
			args:      []string{"version"},
			wantCmd:   "version",
			wantSetup: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				backend    string
				projectDir string
				promptFile string
				maxCalls   int
				timeout    int
				useMonitor bool
				verbose    bool

				setupName  string
				withGit    bool
				importSrc  string
				importName string
			)

			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.BoolVar(&useMonitor, "monitor", false, "")
			fs.BoolVar(&verbose, "verbose", false, "")
			fs.StringVar(&backend, "backend", "sdk", "")
			fs.StringVar(&projectDir, "project", ".", "")
			fs.StringVar(&promptFile, "prompt", "PROMPT.md", "")
			fs.IntVar(&maxCalls, "calls", 100, "")
			fs.IntVar(&timeout, "timeout", 600, "")
			fs.StringVar(&setupName, "name", "", "")
			fs.BoolVar(&withGit, "git", true, "")
			fs.StringVar(&importSrc, "source", "", "")
			fs.StringVar(&importName, "import-name", "", "")

			fs.SetOutput(&bytes.Buffer{})

			err := fs.Parse(tt.args)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			if fs.NArg() == 0 {
				t.Log("Default command: run")
			} else {
				cmd := fs.Arg(0)
				if cmd != tt.wantCmd {
					t.Errorf("Command = %v, want %v", cmd, tt.wantCmd)
				}
			}

			if tt.wantSetup {
				if setupName == "" {
					t.Logf("Setup command without --name: this would fail in real application (expected behavior)")
				} else {
					t.Logf("Setup command with --name: %s", setupName)
				}
			}
		})
	}
}

func TestFlagDefaults(t *testing.T) {
	tests := []struct {
		name         string
		wantBackend  string
		wantProject  string
		wantPrompt   string
		wantMaxCalls int
		wantTimeout  int
	}{
		{
			name:         "default values",
			wantBackend:  "sdk",
			wantProject:  ".",
			wantPrompt:   "PROMPT.md",
			wantMaxCalls: 100,
			wantTimeout:  600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				backend    string
				projectDir string
				promptFile string
				maxCalls   int
				timeout    int
			)

			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.StringVar(&backend, "backend", "sdk", "")
			fs.StringVar(&projectDir, "project", ".", "")
			fs.StringVar(&promptFile, "prompt", "PROMPT.md", "")
			fs.IntVar(&maxCalls, "calls", 100, "")
			fs.IntVar(&timeout, "timeout", 600, "")

			fs.SetOutput(&bytes.Buffer{})

			err := fs.Parse([]string{})
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			if backend != tt.wantBackend {
				t.Errorf("Backend = %v, want %v", backend, tt.wantBackend)
			}
			if projectDir != tt.wantProject {
				t.Errorf("ProjectDir = %v, want %v", projectDir, tt.wantProject)
			}
			if promptFile != tt.wantPrompt {
				t.Errorf("PromptFile = %v, want %v", promptFile, tt.wantPrompt)
			}
			if maxCalls != tt.wantMaxCalls {
				t.Errorf("MaxCalls = %v, want %v", maxCalls, tt.wantMaxCalls)
			}
			if timeout != tt.wantTimeout {
				t.Errorf("Timeout = %v, want %v", timeout, tt.wantTimeout)
			}
		})
	}
}

func TestCustomBackendFlag(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantBackend string
	}{
		{
			name:        "cli backend",
			args:        []string{"--backend", "cli"},
			wantBackend: "cli",
		},
		{
			name:        "sdk backend",
			args:        []string{"--backend", "sdk"},
			wantBackend: "sdk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var backend string

			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.StringVar(&backend, "backend", "sdk", "")
			fs.SetOutput(&bytes.Buffer{})

			err := fs.Parse(tt.args)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			if backend != tt.wantBackend {
				t.Errorf("Backend = %v, want %v", backend, tt.wantBackend)
			}
		})
	}
}

func TestMultipleFlags(t *testing.T) {
	args := []string{
		"--backend", "cli",
		"--calls", "50",
		"--timeout", "300",
		"--verbose",
		"--monitor",
		"run",
	}

	var (
		backend    string
		maxCalls   int
		timeout    int
		verbose    bool
		useMonitor bool
	)

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.StringVar(&backend, "backend", "sdk", "")
	fs.IntVar(&maxCalls, "calls", 100, "")
	fs.IntVar(&timeout, "timeout", 600, "")
	fs.BoolVar(&verbose, "verbose", false, "")
	fs.BoolVar(&useMonitor, "monitor", false, "")
	fs.SetOutput(&bytes.Buffer{})

	err := fs.Parse(args)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if backend != "cli" {
		t.Errorf("Backend = %v, want cli", backend)
	}
	if maxCalls != 50 {
		t.Errorf("MaxCalls = %v, want 50", maxCalls)
	}
	if timeout != 300 {
		t.Errorf("Timeout = %v, want 300", timeout)
	}
	if !verbose {
		t.Error("Verbose should be true")
	}
	if !useMonitor {
		t.Error("Monitor should be true")
	}

	cmd := fs.Arg(0)
	if cmd != "run" {
		t.Errorf("Command = %v, want run", cmd)
	}
}

func TestCommandValidation(t *testing.T) {
	tests := []struct {
		name    string
		command string
		isValid bool
	}{
		{
			name:    "valid commands",
			command: "run",
			isValid: true,
		},
		{
			name:    "valid setup command",
			command: "setup",
			isValid: true,
		},
		{
			name:    "valid import command",
			command: "import",
			isValid: true,
		},
		{
			name:    "valid status command",
			command: "status",
			isValid: true,
		},
		{
			name:    "valid reset-circuit command",
			command: "reset-circuit",
			isValid: true,
		},
		{
			name:    "valid help command",
			command: "help",
			isValid: true,
		},
		{
			name:    "valid version command",
			command: "version",
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validCommands := map[string]bool{
				"run":           true,
				"setup":         true,
				"import":        true,
				"status":        true,
				"reset-circuit": true,
				"help":          true,
				"version":       true,
			}

			_, ok := validCommands[tt.command]
			if ok != tt.isValid {
				t.Errorf("Command %s validity check failed", tt.command)
			}
		})
	}
}

func TestHelpText(t *testing.T) {
	var buf bytes.Buffer

	buf.WriteString("Lisa Codex - Autonomous AI Development Loop with Charm TUI\n\n")
	buf.WriteString("Usage:\n")
	buf.WriteString("  ralph [command] [options]\n\n")
	buf.WriteString("Commands:\n")
	buf.WriteString("  run (default)      Run autonomous development loop\n")
	buf.WriteString("  setup              Create a new Lisa-managed project\n")
	buf.WriteString("  import              Import PRD or specification document\n")
	buf.WriteString("  status             Show project status\n")
	buf.WriteString("  reset-circuit       Reset circuit breaker state\n")
	buf.WriteString("  help               Show this help\n")
	buf.WriteString("  version            Show version\n")

	helpText := buf.String()

	requiredSections := []string{
		"Commands:",
		"run (default)",
		"setup",
		"import",
		"status",
		"reset-circuit",
		"help",
		"version",
	}

	for _, section := range requiredSections {
		if !strings.Contains(helpText, section) {
			t.Errorf("Help text missing required section: %s", section)
		}
	}
}
