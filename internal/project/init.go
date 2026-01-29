package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

// InitMode specifies the initialization mode
type InitMode string

const (
	// ModeImplementation generates IMPLEMENTATION_PLAN.md and AGENTS.md from PRD.md
	ModeImplementation InitMode = "implementation"
	// ModeFix generates @fix_plan.md from specs folder
	ModeFix InitMode = "fix"
	// ModeRefactor generates REFACTOR_PLAN.md from REFACTOR.md
	ModeRefactor InitMode = "refactor"
)

// InitOptions holds options for project initialization from PRD
type InitOptions struct {
	PRDPath      string   // Path to PRD.md (default: ./PRD.md)
	SpecsDir     string   // Path to specs directory (default: ./specs)
	RefactorPath string   // Path to REFACTOR.md (default: ./REFACTOR.md)
	OutputDir    string   // Output directory (default: current directory)
	Mode         InitMode // Initialization mode (implementation, fix, or refactor)
	Verbose      bool
}

// InitResult holds the result of project initialization
type InitResult struct {
	PRDPath                string
	SpecsDir               string
	RefactorPath           string
	ImplementationPlanPath string
	AgentsPath             string
	FixPlanPath            string
	RefactorPlanPath       string
	Mode                   InitMode
	Success                bool
}

// InitFromPRD initializes a Lisa project from an existing PRD.md
// It generates IMPLEMENTATION_PLAN.md and AGENTS.md using Codex
func InitFromPRD(opts InitOptions) (*InitResult, error) {
	// Default PRD path
	if opts.PRDPath == "" {
		opts.PRDPath = "PRD.md"
	}

	// Default output directory
	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}

	implPlanPath := filepath.Join(opts.OutputDir, "IMPLEMENTATION_PLAN.md")
	agentsPath := filepath.Join(opts.OutputDir, "AGENTS.md")

	// Check if IMPLEMENTATION_PLAN.md already exists - skip generation
	if _, err := os.Stat(implPlanPath); err == nil {
		log.Info("IMPLEMENTATION_PLAN.md already exists, skipping generation")
		return &InitResult{
			PRDPath:                opts.PRDPath,
			ImplementationPlanPath: implPlanPath,
			AgentsPath:             agentsPath,
			Success:                true,
		}, nil
	}

	// Check if PRD exists
	prdPath := opts.PRDPath
	if !filepath.IsAbs(prdPath) {
		prdPath = filepath.Join(opts.OutputDir, prdPath)
	}

	if _, err := os.Stat(prdPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("PRD.md not found at %s. Create a PRD.md file first or specify path with --prd flag", prdPath)
	}

	// Read PRD content
	prdContent, err := os.ReadFile(prdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PRD.md: %w", err)
	}

	log.Info("Found PRD", "path", prdPath, "size", len(prdContent))

	result := &InitResult{
		PRDPath: prdPath,
		Success: false,
	}

	// Generate IMPLEMENTATION_PLAN.md
	log.Info("Generating IMPLEMENTATION_PLAN.md...")

	implPlanContent, err := generateWithCodex(BuildImplementationPlanPrompt(string(prdContent)), opts.Verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to generate IMPLEMENTATION_PLAN.md: %w", err)
	}

	if err := os.WriteFile(implPlanPath, []byte(implPlanContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write IMPLEMENTATION_PLAN.md: %w", err)
	}
	result.ImplementationPlanPath = implPlanPath
	log.Info("Created", "file", implPlanPath)

	// Generate AGENTS.md
	log.Info("Generating AGENTS.md...")

	agentsContent, err := generateWithCodex(BuildAgentsPrompt(string(prdContent)), opts.Verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AGENTS.md: %w", err)
	}

	if err := os.WriteFile(agentsPath, []byte(agentsContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write AGENTS.md: %w", err)
	}
	result.AgentsPath = agentsPath
	log.Info("Created", "file", agentsPath)

	result.Success = true
	return result, nil
}

// generateWithCodex calls Codex CLI and returns the generated content
// Output is streamed to the console in real-time using the unified helper
func generateWithCodex(prompt string, verbose bool) (string, error) {
	return RunCodexSimple(prompt, verbose)
}

// FindPRD looks for a PRD file in the given directory
func FindPRD(dir string) (string, error) {
	// Common PRD file names to look for
	prdNames := []string{
		"PRD.md",
		"prd.md",
		"PRD.MD",
		"PRODUCT_REQUIREMENTS.md",
		"product_requirements.md",
		"requirements.md",
		"REQUIREMENTS.md",
	}

	for _, name := range prdNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no PRD file found in %s (looked for: %s)", dir, strings.Join(prdNames, ", "))
}

// HasPRD checks if a PRD file exists in the directory
func HasPRD(dir string) bool {
	_, err := FindPRD(dir)
	return err == nil
}

// HasSpecs checks if a specs directory exists with files
func HasSpecs(dir string) bool {
	specsDir := filepath.Join(dir, "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return false
	}
	// Check if there are any markdown files
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			return true
		}
	}
	return false
}

// HasRefactor checks if a REFACTOR.md file exists
func HasRefactor(dir string) bool {
	_, err := FindRefactor(dir)
	return err == nil
}

// FindRefactor looks for a refactor file in the given directory
func FindRefactor(dir string) (string, error) {
	// Common refactor file names to look for
	refactorNames := []string{
		"REFACTOR.md",
		"refactor.md",
		"REFACTORING.md",
		"refactoring.md",
	}

	for _, name := range refactorNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no REFACTOR.md file found in %s", dir)
}

// Init initializes a Lisa project based on the mode
// - Implementation mode: Uses PRD.md to generate IMPLEMENTATION_PLAN.md and AGENTS.md
// - Fix mode: Uses specs folder to generate @fix_plan.md
// - Refactor mode: Analyzes codebase to generate REFACTOR_PLAN.md (no input file needed)
func Init(opts InitOptions) (*InitResult, error) {
	// Auto-detect mode if not specified
	if opts.Mode == "" {
		if HasSpecs(opts.OutputDir) {
			opts.Mode = ModeFix
		} else if HasPRD(opts.OutputDir) {
			opts.Mode = ModeImplementation
		} else {
			// Default to refactor mode if no specific input files found
			// Refactor mode doesn't require input files - Codex analyzes the codebase
			return nil, fmt.Errorf("could not detect project type. Need PRD.md or specs/ folder, or specify --mode refactor")
		}
	}

	switch opts.Mode {
	case ModeRefactor:
		return InitRefactorMode(opts)
	case ModeFix:
		return InitFixMode(opts)
	case ModeImplementation:
		return InitFromPRD(opts)
	default:
		return nil, fmt.Errorf("unknown mode: %s", opts.Mode)
	}
}

// InitFixMode generates @fix_plan.md from specs folder
func InitFixMode(opts InitOptions) (*InitResult, error) {
	// Default specs directory
	if opts.SpecsDir == "" {
		opts.SpecsDir = "specs"
	}

	// Default output directory
	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}

	fixPlanPath := filepath.Join(opts.OutputDir, "@fix_plan.md")

	// Check if @fix_plan.md already exists - skip generation
	if _, err := os.Stat(fixPlanPath); err == nil {
		log.Info("@fix_plan.md already exists, skipping generation")
		return &InitResult{
			FixPlanPath: fixPlanPath,
			Mode:        ModeFix,
			Success:     true,
		}, nil
	}

	// Check if specs directory exists
	specsPath := opts.SpecsDir
	if !filepath.IsAbs(specsPath) {
		specsPath = filepath.Join(opts.OutputDir, specsPath)
	}

	if _, err := os.Stat(specsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("specs directory not found at %s", specsPath)
	}

	// Read all spec files
	specsContent, err := readSpecsFolder(specsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read specs: %w", err)
	}

	log.Info("Found specs", "path", specsPath, "size", len(specsContent))

	result := &InitResult{
		SpecsDir: specsPath,
		Mode:     ModeFix,
		Success:  false,
	}

	// Generate @fix_plan.md
	log.Info("Generating @fix_plan.md...")

	fixPlanContent, err := generateWithCodex(BuildFixPlanPrompt(specsContent), opts.Verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to generate @fix_plan.md: %w", err)
	}

	if err := os.WriteFile(fixPlanPath, []byte(fixPlanContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write @fix_plan.md: %w", err)
	}
	result.FixPlanPath = fixPlanPath
	log.Info("Created", "file", fixPlanPath)

	result.Success = true
	return result, nil
}

// InitRefactorMode generates REFACTOR_PLAN.md by having Codex analyze the codebase
func InitRefactorMode(opts InitOptions) (*InitResult, error) {
	// Default output directory
	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}

	result := &InitResult{
		Mode:    ModeRefactor,
		Success: false,
	}

	refactorPlanPath := filepath.Join(opts.OutputDir, "REFACTOR_PLAN.md")

	// Check if REFACTOR_PLAN.md already exists - skip generation
	if _, err := os.Stat(refactorPlanPath); err == nil {
		log.Info("REFACTOR_PLAN.md already exists, skipping generation")
		result.RefactorPlanPath = refactorPlanPath
		result.Success = true
		return result, nil
	}

	// Generate REFACTOR_PLAN.md using the prompt template
	// Codex will analyze the codebase and generate the plan
	log.Info("Generating REFACTOR_PLAN.md...")
	log.Info("Codex will analyze the codebase and create a refactoring plan")

	// Get the refactor plan prompt template
	prompt, err := GetRefactorPlanPrompt()
	if err != nil {
		return nil, fmt.Errorf("failed to load refactor plan prompt: %w", err)
	}

	refactorPlanContent, err := generateWithCodex(prompt, opts.Verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to generate REFACTOR_PLAN.md: %w", err)
	}

	if err := os.WriteFile(refactorPlanPath, []byte(refactorPlanContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write REFACTOR_PLAN.md: %w", err)
	}
	result.RefactorPlanPath = refactorPlanPath

	log.Info("Created", "file", refactorPlanPath)

	result.Success = true
	return result, nil
}

// readSpecsFolder reads all markdown files from the specs folder and combines them
func readSpecsFolder(specsDir string) (string, error) {
	var content strings.Builder

	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return "", fmt.Errorf("failed to read specs directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process markdown files
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}

		filePath := filepath.Join(specsDir, name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", filePath, err)
		}

		content.WriteString(fmt.Sprintf("\n## File: %s\n\n", name))
		content.WriteString(string(data))
		content.WriteString("\n\n---\n")
	}

	if content.Len() == 0 {
		return "", fmt.Errorf("no markdown files found in specs directory")
	}

	return content.String(), nil
}
