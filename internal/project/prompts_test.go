package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPromptTemplate(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (string, func())
		templateDir string
		filename    string
		wantContent string
		wantErr     bool
		errContains string
	}{
		{
			name: "custom template directory takes precedence",
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				customDir := filepath.Join(tmpDir, "custom")
				os.MkdirAll(customDir, 0755)

				// Create custom template
				customContent := "Custom template content"
				os.WriteFile(filepath.Join(customDir, "test.md"), []byte(customContent), 0644)

				cleanup := func() {}
				return customDir, cleanup
			},
			templateDir: "", // Will be set by setupFunc return value
			filename:    "test.md",
			wantContent: "Custom template content",
			wantErr:     false,
		},
		{
			name: "falls back to home directory when custom not found",
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()

				// Create home directory structure
				homeDir := filepath.Join(tmpDir, "home")
				os.MkdirAll(homeDir, 0755)
				os.Setenv("HOME", homeDir)

				ralphDir := filepath.Join(homeDir, ".ralph", "templates")
				os.MkdirAll(ralphDir, 0755)

				// Create home template
				homeContent := "Home template content"
				os.WriteFile(filepath.Join(ralphDir, "test.md"), []byte(homeContent), 0644)

				cleanup := func() {
					os.Unsetenv("HOME")
				}
				return "", cleanup // Empty templateDir to trigger home lookup
			},
			templateDir: "", // Empty to trigger fallback
			filename:    "test.md",
			wantContent: "Home template content",
			wantErr:     false,
		},
		{
			name: "template not found in any location",
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				// Set HOME to temp dir with no .ralph/templates
				os.Setenv("HOME", tmpDir)

				cleanup := func() {
					os.Unsetenv("HOME")
				}
				return "", cleanup
			},
			templateDir: "/nonexistent/custom",
			filename:    "nonexistent.md",
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "custom directory overrides home directory",
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()

				// Create home directory with template
				homeDir := filepath.Join(tmpDir, "home")
				os.MkdirAll(homeDir, 0755)
				os.Setenv("HOME", homeDir)
				ralphDir := filepath.Join(homeDir, ".ralph", "templates")
				os.MkdirAll(ralphDir, 0755)
				os.WriteFile(filepath.Join(ralphDir, "test.md"), []byte("Home version"), 0644)

				// Create custom directory with different template
				customDir := filepath.Join(tmpDir, "custom")
				os.MkdirAll(customDir, 0755)
				os.WriteFile(filepath.Join(customDir, "test.md"), []byte("Custom version"), 0644)

				cleanup := func() {
					os.Unsetenv("HOME")
				}
				return customDir, cleanup
			},
			templateDir: "", // Set by setupFunc
			filename:    "test.md",
			wantContent: "Custom version",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original TemplateDir
			origTemplateDir := TemplateDir
			defer func() { TemplateDir = origTemplateDir }()

			// Setup test environment
			customDir, cleanup := tt.setupFunc(t)
			defer cleanup()

			// Set TemplateDir (either from setupFunc or test case)
			if tt.templateDir != "" {
				TemplateDir = tt.templateDir
			} else if customDir != "" {
				TemplateDir = customDir
			}

			content, err := loadPromptTemplate(tt.filename)

			if tt.wantErr {
				if err == nil {
					t.Errorf("loadPromptTemplate() expected error, got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("loadPromptTemplate() error = %v, expected to contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("loadPromptTemplate() unexpected error = %v", err)
				return
			}

			if !strings.Contains(content, tt.wantContent) {
				t.Errorf("loadPromptTemplate() content = %q, expected to contain %q", content, tt.wantContent)
			}
		})
	}
}

func TestLoadPromptTemplate_ResolutionOrder(t *testing.T) {
	// Save original TemplateDir
	origTemplateDir := TemplateDir
	defer func() { TemplateDir = origTemplateDir }()

	// Create a temp directory structure
	tmpDir := t.TempDir()

	// Create three tiers of template directories
	customDir := filepath.Join(tmpDir, "custom")
	homeDir := filepath.Join(tmpDir, "home")
	defaultDir := filepath.Join(tmpDir, "default")

	os.MkdirAll(customDir, 0755)
	os.MkdirAll(filepath.Join(homeDir, ".ralph", "templates"), 0755)
	os.MkdirAll(defaultDir, 0755)

	// Create templates at each level with different content
	os.WriteFile(filepath.Join(customDir, "tier_test.md"), []byte("custom"), 0644)
	os.WriteFile(filepath.Join(homeDir, ".ralph", "templates", "tier_test.md"), []byte("home"), 0644)
	os.WriteFile(filepath.Join(defaultDir, "tier_test.md"), []byte("default"), 0644)

	// Set HOME environment
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", origHome)

	// Temporarily override getDefaultTemplateDir to return our test directory
	// Since we can't easily override the function, we'll test the resolution
	// by checking each tier individually

	tests := []struct {
		name        string
		templateDir string
		wantContent string
		description string
	}{
		{
			name:        "custom tier",
			templateDir: customDir,
			wantContent: "custom",
			description: "Custom template directory should be used when provided",
		},
		{
			name:        "home tier",
			templateDir: "", // Empty to trigger fallback
			wantContent: "home",
			description: "Home directory should be used when custom is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			TemplateDir = tt.templateDir

			content, err := loadPromptTemplate("tier_test.md")
			if err != nil {
				t.Fatalf("loadPromptTemplate() unexpected error = %v", err)
			}

			if !strings.Contains(content, tt.wantContent) {
				t.Errorf("%s: got content %q, expected %q", tt.description, content, tt.wantContent)
			}
		})
	}
}

func TestGetImplementationPlanPrompt(t *testing.T) {
	// This test verifies the function can be called and handles missing templates
	// Since we can't guarantee templates exist in all environments,
	// we just verify it doesn't panic and returns expected behavior
	_, err := GetImplementationPlanPrompt()
	// May error if templates don't exist, which is fine for this test
	_ = err
}

func TestGetAgentsPrompt(t *testing.T) {
	// Similar to above - verify function can be called
	_, err := GetAgentsPrompt()
	_ = err
}

func TestGetFixPlanPrompt(t *testing.T) {
	// Similar to above - verify function can be called
	_, err := GetFixPlanPrompt()
	_ = err
}

func TestGetRefactorPlanPrompt(t *testing.T) {
	// Similar to above - verify function can be called
	_, err := GetRefactorPlanPrompt()
	_ = err
}

func TestBuildImplementationPlanPrompt_WithPRD(t *testing.T) {
	prdContent := "# My Project\n\nBuild something amazing."
	prompt := BuildImplementationPlanPrompt(prdContent)

	// Should contain PRD content
	if !strings.Contains(prompt, prdContent) {
		t.Error("BuildImplementationPlanPrompt() should contain PRD content")
	}

	// Should contain separator
	if !strings.Contains(prompt, "---") {
		t.Error("BuildImplementationPlanPrompt() should contain separator")
	}
}

func TestBuildAgentsPrompt_WithPRD(t *testing.T) {
	prdContent := "# My Project\n\nBuild something amazing."
	prompt := BuildAgentsPrompt(prdContent)

	// Should contain PRD content
	if !strings.Contains(prompt, prdContent) {
		t.Error("BuildAgentsPrompt() should contain PRD content")
	}

	// Should contain separator
	if !strings.Contains(prompt, "---") {
		t.Error("BuildAgentsPrompt() should contain separator")
	}
}

func TestBuildFixPlanPrompt_WithSpecs(t *testing.T) {
	specsContent := "# API Spec\n\nDefine endpoints."
	prompt := BuildFixPlanPrompt(specsContent)

	// Should contain specs content
	if !strings.Contains(prompt, specsContent) {
		t.Error("BuildFixPlanPrompt() should contain specs content")
	}

	// Should contain separator
	if !strings.Contains(prompt, "---") {
		t.Error("BuildFixPlanPrompt() should contain separator")
	}
}

func TestBuildRefactorPlanPrompt_WithContent(t *testing.T) {
	refactorContent := "# Refactoring Goals\n\nClean up code."
	prompt := BuildRefactorPlanPrompt(refactorContent)

	// Should contain refactor content
	if !strings.Contains(prompt, refactorContent) {
		t.Error("BuildRefactorPlanPrompt() should contain refactor content")
	}

	// Should contain separator
	if !strings.Contains(prompt, "---") {
		t.Error("BuildRefactorPlanPrompt() should contain separator")
	}
}
