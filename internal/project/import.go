package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImportOptions holds options for PRD import
type ImportOptions struct {
	SourcePath    string
	ProjectName   string
	OutputDir     string
	Verbose       bool
	ConvertFormat string
}

// ImportResult holds result of PRD import
type ImportResult struct {
	FilesCreated  []string
	ProjectName   string
	Success       bool
	Warnings      []string
	ConvertedFrom string
}

// ImportPRD imports a PRD or specification document and converts to Lisa format
func ImportPRD(opts ImportOptions) (*ImportResult, error) {
	// Validate inputs
	if opts.SourcePath == "" {
		return nil, fmt.Errorf("source path is required")
	}

	// Check if source file exists
	if _, err := os.Stat(opts.SourcePath); err != nil {
		return nil, fmt.Errorf("source file not found: %s", opts.SourcePath)
	}

	// Determine project name
	projectName := opts.ProjectName
	if projectName == "" {
		projectName = extractProjectName(opts.SourcePath)
	}

	result := &ImportResult{
		FilesCreated:  []string{},
		ProjectName:   projectName,
		Success:       false,
		Warnings:      []string{},
		ConvertedFrom: filepath.Ext(opts.SourcePath),
	}

	// Read source content
	sourceContent, err := os.ReadFile(opts.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file: %w", err)
	}

	// Parse source content
	prompt, fixPlan, agent, warnings, err := parseSourceContent(string(sourceContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse source content: %w", err)
	}
	result.Warnings = warnings

	// Determine output directory
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = "."
	}

	// Validate output directory
	if _, err := os.Stat(outputDir); err != nil {
		return nil, fmt.Errorf("output directory not found: %s", outputDir)
	}

	// Write PROMPT.md
	promptPath := filepath.Join(outputDir, "PROMPT.md")
	if err := os.WriteFile(promptPath, []byte(prompt), 0644); err != nil {
		return nil, fmt.Errorf("failed to write PROMPT.md: %w", err)
	}
	result.FilesCreated = append(result.FilesCreated, promptPath)
	if opts.Verbose {
		fmt.Printf("Created: %s\n", promptPath)
	}

	// Write @fix_plan.md
	fixPlanPath := filepath.Join(outputDir, "@fix_plan.md")
	if err := os.WriteFile(fixPlanPath, []byte(fixPlan), 0644); err != nil {
		return nil, fmt.Errorf("failed to write @fix_plan.md: %w", err)
	}
	result.FilesCreated = append(result.FilesCreated, fixPlanPath)
	if opts.Verbose {
		fmt.Printf("Created: %s\n", fixPlanPath)
	}

	// Write @AGENT.md
	agentPath := filepath.Join(outputDir, "@AGENT.md")
	if err := os.WriteFile(agentPath, []byte(agent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write @AGENT.md: %w", err)
	}
	result.FilesCreated = append(result.FilesCreated, agentPath)
	if opts.Verbose {
		fmt.Printf("Created: %s\n", agentPath)
	}

	result.Success = true

	return result, nil
}

// parseSourceContent parses source content and extracts PROMPT, fix plan, and agent sections
// When encountering repeated headings for the same section, content is appended rather than reset
func parseSourceContent(content string) (string, string, string, []string, error) {
	var promptBuilder, fixPlanBuilder, agentBuilder strings.Builder
	warnings := []string{}

	lines := strings.Split(content, "\n")
	currentSection := "prompt"

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect section headers and switch
		if strings.HasPrefix(trimmed, "#") {
			lowerLine := strings.ToLower(trimmed)
			foundSection := false

			// Check for prompt section (highest priority)
			if strings.Contains(lowerLine, "prompt") {
				currentSection = "prompt"
				// Append the heading, don't reset - preserves content from repeated headings
				promptBuilder.WriteString(line)
				promptBuilder.WriteString("\n")
				foundSection = true
			}
			// Check for fixplan section (only if prompt not found)
			if !foundSection && (strings.Contains(lowerLine, "task") || strings.Contains(lowerLine, "plan") || strings.Contains(lowerLine, "todo")) {
				currentSection = "fixplan"
				// Append the heading, don't reset - preserves content from repeated headings
				fixPlanBuilder.WriteString(line)
				fixPlanBuilder.WriteString("\n")
				foundSection = true
			}
			// Check for agent section (only if others not found)
			if !foundSection && strings.Contains(lowerLine, "agent") {
				currentSection = "agent"
				// Append the heading, don't reset - preserves content from repeated headings
				agentBuilder.WriteString(line)
				agentBuilder.WriteString("\n")
			}
			continue
		}

		// Append to appropriate section
		switch currentSection {
		case "prompt":
			promptBuilder.WriteString(line)
			promptBuilder.WriteString("\n")
		case "fixplan":
			fixPlanBuilder.WriteString(line)
			fixPlanBuilder.WriteString("\n")
		case "agent":
			agentBuilder.WriteString(line)
			agentBuilder.WriteString("\n")
		}
	}

	prompt := strings.TrimSpace(promptBuilder.String())
	fixPlan := strings.TrimSpace(fixPlanBuilder.String())
	agent := strings.TrimSpace(agentBuilder.String())

	// Provide defaults for missing sections
	if prompt == "" {
		prompt = "# Development Instructions\n\nPlease specify development goals and rules."
		warnings = append(warnings, "No prompt section found, using default")
	}

	if fixPlan == "" {
		fixPlan = defaultFixPlanTemplate()
		warnings = append(warnings, "No fix plan section found, using default")
	}

	if agent == "" {
		agent = defaultAgentTemplate()
		warnings = append(warnings, "No agent section found, using default")
	}

	return prompt, fixPlan, agent, warnings, nil
}

// extractProjectName extracts project name from source file path
func extractProjectName(sourcePath string) string {
	// Get filename without extension
	filename := filepath.Base(sourcePath)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Remove common prefixes
	prefixes := []string{"PRD_", "prd_", "spec_", "SPEC_", "requirements_", "REQUIREMENTS_"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			name = name[len(prefix):]
			break
		}
	}

	// Remove common suffixes
	suffixes := []string{"_prd", "_PRD", "_spec", "_SPEC", "_requirements", "_REQUIREMENTS"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			name = name[:len(name)-len(suffix)]
			break
		}
	}

	// Clean up remaining special characters
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ToLower(name)

	if name == "" {
		name = "my-project"
	}

	return name
}

// SupportedFormats returns list of supported import formats
func SupportedFormats() []string {
	return []string{
		".md",
		".txt",
		".json",
		".yaml",
		".yml",
	}
}

// IsSupportedFormat checks if file format is supported
func IsSupportedFormat(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))

	for _, supported := range SupportedFormats() {
		if ext == supported {
			return true
		}
	}

	return false
}

// GetConversionSummary returns a summary of what was converted
func (r *ImportResult) GetConversionSummary() string {
	builder := &strings.Builder{}
	fmt.Fprintf(builder, "Project: %s\n", r.ProjectName)
	fmt.Fprintf(builder, "Source: %s\n", r.ConvertedFrom)
	fmt.Fprintf(builder, "Files created: %d\n", len(r.FilesCreated))

	if len(r.Warnings) > 0 {
		builder.WriteString("\nWarnings:\n")
		for _, warning := range r.Warnings {
			fmt.Fprintf(builder, "  - %s\n", warning)
		}
	}

	return builder.String()
}
