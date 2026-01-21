package project

import (
	"fmt"
	"os"
	"path/filepath"
)

// ProjectMode represents the type of Ralph project
type ProjectMode string

const (
	ProjectModeImplement ProjectMode = "implement"
	ProjectModeRefactor  ProjectMode = "refactor"
	ProjectModeFix       ProjectMode = "fix"
	ProjectModeUnknown   ProjectMode = "unknown"
)

// String returns the string representation of the mode
func (m ProjectMode) String() string {
	return string(m)
}

// ModeConfig holds the file configuration for each mode
type ModeConfig struct {
	Mode       ProjectMode
	InputFile  string   // Primary input file (e.g., PRD.md)
	PlanFile   string   // Plan file (e.g., IMPLEMENTATION_PLAN.md)
	AltInputs  []string // Alternative input files that can trigger this mode
}

// ModeConfigs defines the file configuration for each project mode
var ModeConfigs = []ModeConfig{
	{
		Mode:      ProjectModeImplement,
		InputFile: "PRD.md",
		PlanFile:  "IMPLEMENTATION_PLAN.md",
	},
	{
		Mode:      ProjectModeRefactor,
		InputFile: "REFACTOR.md",
		PlanFile:  "REFACTOR_PLAN.md",
		AltInputs: []string{"REFACTOR_PLAN.md"}, // Can have just the plan file
	},
	{
		Mode:      ProjectModeFix,
		InputFile: "PROMPT.md",
		PlanFile:  "@fix_plan.md",
	},
}

// DetectMode determines the project mode based on files present in the directory
// Priority: Refactor > Fix > Implement (so users can switch to refactor mode easily)
func DetectMode(dir string) ProjectMode {
	if dir == "" {
		dir = "."
	}

	// Refactor mode: REFACTOR_PLAN.md (optionally with REFACTOR.md)
	// Check this FIRST so users can switch to refactor mode even if PRD.md exists
	if fileExistsAt(dir, "REFACTOR_PLAN.md") {
		return ProjectModeRefactor
	}

	// Fix mode: PROMPT.md + @fix_plan.md
	if fileExistsAt(dir, "PROMPT.md") && fileExistsAt(dir, "@fix_plan.md") {
		return ProjectModeFix
	}

	// Implement mode: PRD.md + IMPLEMENTATION_PLAN.md
	if fileExistsAt(dir, "PRD.md") && fileExistsAt(dir, "IMPLEMENTATION_PLAN.md") {
		return ProjectModeImplement
	}

	return ProjectModeUnknown
}

// GetPlanFile returns the plan file path for a given mode
func GetPlanFile(mode ProjectMode) string {
	for _, cfg := range ModeConfigs {
		if cfg.Mode == mode {
			return cfg.PlanFile
		}
	}
	return ""
}

// GetInputFile returns the input file path for a given mode
func GetInputFile(mode ProjectMode) string {
	for _, cfg := range ModeConfigs {
		if cfg.Mode == mode {
			return cfg.InputFile
		}
	}
	return ""
}

// ValidateModeFiles checks if the required files exist for a given mode
func ValidateModeFiles(dir string, mode ProjectMode) error {
	if dir == "" {
		dir = "."
	}

	for _, cfg := range ModeConfigs {
		if cfg.Mode == mode {
			// Check plan file (required)
			if !fileExistsAt(dir, cfg.PlanFile) {
				return fmt.Errorf("missing plan file: %s", cfg.PlanFile)
			}

			// For refactor mode, input file is optional
			if mode != ProjectModeRefactor {
				if !fileExistsAt(dir, cfg.InputFile) {
					return fmt.Errorf("missing input file: %s", cfg.InputFile)
				}
			}
			return nil
		}
	}

	return fmt.Errorf("unknown mode: %s", mode)
}

// FindProjectRoot searches upward from the current directory to find a Ralph project root
func FindProjectRoot(startDir string) (string, ProjectMode, error) {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return "", ProjectModeUnknown, err
		}
	}

	dir := startDir
	for {
		mode := DetectMode(dir)
		if mode != ProjectModeUnknown {
			return dir, mode, nil
		}

		// Also check for .git as a fallback indicator
		if fileExistsAt(dir, ".git") {
			// Found git root but no Ralph project files
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return "", ProjectModeUnknown, fmt.Errorf("could not find Ralph project root (no valid mode configuration found)")
}

// IsValidProject checks if the directory contains a valid Ralph project
func IsValidProject(dir string) bool {
	return DetectMode(dir) != ProjectModeUnknown
}

// ValidateProjectDir checks if the directory is a valid Ralph project and returns an error if not
func ValidateProjectDir(dir string) error {
	mode := DetectMode(dir)
	if mode == ProjectModeUnknown {
		return fmt.Errorf("not a valid Ralph project. Need one of:\n  - PRD.md + IMPLEMENTATION_PLAN.md (implementation mode)\n  - REFACTOR_PLAN.md (refactor mode)\n  - PROMPT.md + @fix_plan.md (fix mode)")
	}
	return nil
}

// fileExistsAt checks if a file exists at the given directory
func fileExistsAt(dir, filename string) bool {
	path := filepath.Join(dir, filename)
	_, err := os.Stat(path)
	return err == nil
}

// ConvertInitMode converts InitMode to ProjectMode for compatibility
func ConvertInitMode(im InitMode) ProjectMode {
	switch im {
	case ModeImplementation:
		return ProjectModeImplement
	case ModeFix:
		return ProjectModeFix
	case ModeRefactor:
		return ProjectModeRefactor
	default:
		return ProjectModeUnknown
	}
}

// ConvertToInitMode converts ProjectMode to InitMode for compatibility
func ConvertToInitMode(pm ProjectMode) InitMode {
	switch pm {
	case ProjectModeImplement:
		return ModeImplementation
	case ProjectModeFix:
		return ModeFix
	case ProjectModeRefactor:
		return ModeRefactor
	default:
		return ""
	}
}
