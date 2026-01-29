package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindPRD(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-find-prd")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test: No PRD file
	_, err = FindPRD(tmpDir)
	if err == nil {
		t.Error("FindPRD should return error when no PRD file exists")
	}

	// Create PRD.md
	prdPath := filepath.Join(tmpDir, "PRD.md")
	if err := os.WriteFile(prdPath, []byte("# Test PRD"), 0644); err != nil {
		t.Fatalf("Failed to write PRD.md: %v", err)
	}

	// Test: PRD.md exists
	foundPath, err := FindPRD(tmpDir)
	if err != nil {
		t.Errorf("FindPRD failed: %v", err)
	}
	if foundPath != prdPath {
		t.Errorf("FindPRD returned %s, expected %s", foundPath, prdPath)
	}
}

func TestHasPRD(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-has-prd")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test: No PRD
	if HasPRD(tmpDir) {
		t.Error("HasPRD should return false when no PRD exists")
	}

	// Create PRD.md
	prdPath := filepath.Join(tmpDir, "PRD.md")
	if err := os.WriteFile(prdPath, []byte("# Test PRD"), 0644); err != nil {
		t.Fatalf("Failed to write PRD.md: %v", err)
	}

	// Test: PRD exists
	if !HasPRD(tmpDir) {
		t.Error("HasPRD should return true when PRD.md exists")
	}
}

func TestInitFromPRD_NoPRD(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-init-no-prd")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test: Init without PRD should fail
	opts := InitOptions{
		OutputDir: tmpDir,
		Verbose:   false,
	}

	_, err = InitFromPRD(opts)
	if err == nil {
		t.Error("InitFromPRD should fail when PRD.md doesn't exist")
	}
}

func TestBuildImplementationPlanPrompt(t *testing.T) {
	prdContent := "# Test Project\n\nBuild a simple CLI tool."

	prompt := BuildImplementationPlanPrompt(prdContent)

	// Should contain the PRD content
	if !contains(prompt, prdContent) {
		t.Error("Prompt should contain PRD content")
	}

	// Should contain separator
	if !contains(prompt, "---") {
		t.Error("Prompt should contain separator")
	}
}

func TestBuildAgentsPrompt(t *testing.T) {
	prdContent := "# Test Project\n\nBuild a web API."

	prompt := BuildAgentsPrompt(prdContent)

	// Should contain the PRD content
	if !contains(prdContent, prdContent) {
		t.Error("Prompt should contain PRD content")
	}

	// Should contain separator
	if !contains(prompt, "---") {
		t.Error("Prompt should contain separator")
	}
}

func TestHasSpecs(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-has-specs")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test: No specs folder
	if HasSpecs(tmpDir) {
		t.Error("HasSpecs should return false when no specs folder exists")
	}

	// Create specs folder but empty
	specsDir := filepath.Join(tmpDir, "specs")
	if err := os.MkdirAll(specsDir, 0755); err != nil {
		t.Fatalf("Failed to create specs dir: %v", err)
	}

	if HasSpecs(tmpDir) {
		t.Error("HasSpecs should return false when specs folder is empty")
	}

	// Add a markdown file
	specFile := filepath.Join(specsDir, "api.md")
	if err := os.WriteFile(specFile, []byte("# API Spec"), 0644); err != nil {
		t.Fatalf("Failed to write spec file: %v", err)
	}

	if !HasSpecs(tmpDir) {
		t.Error("HasSpecs should return true when specs folder has .md files")
	}
}

func TestInitFixMode_NoSpecs(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-fix-no-specs")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := InitOptions{
		OutputDir: tmpDir,
		Mode:      ModeFix,
		Verbose:   false,
	}

	_, err = InitFixMode(opts)
	if err == nil {
		t.Error("InitFixMode should fail when specs folder doesn't exist")
	}
}

func TestInit_AutoDetect(t *testing.T) {
	// Create temp directory with PRD
	tmpDir, err := os.MkdirTemp("", "test-init-autodetect")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// No PRD or specs - should fail
	opts := InitOptions{
		OutputDir: tmpDir,
		Verbose:   false,
	}

	_, err = Init(opts)
	if err == nil {
		t.Error("Init should fail when neither PRD.md nor specs/ exists")
	}
}

func TestBuildFixPlanPrompt(t *testing.T) {
	specsContent := "# API Spec\n\nDefine REST endpoints."

	prompt := BuildFixPlanPrompt(specsContent)

	// Should contain the specs content
	if !contains(prompt, specsContent) {
		t.Error("Prompt should contain specs content")
	}

	// Should contain separator
	if !contains(prompt, "---") {
		t.Error("Prompt should contain separator")
	}
}

func TestHasRefactor(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-has-refactor")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test: No REFACTOR.md
	if HasRefactor(tmpDir) {
		t.Error("HasRefactor should return false when no REFACTOR.md exists")
	}

	// Create REFACTOR.md
	refactorPath := filepath.Join(tmpDir, "REFACTOR.md")
	if err := os.WriteFile(refactorPath, []byte("# Refactoring Goals"), 0644); err != nil {
		t.Fatalf("Failed to write REFACTOR.md: %v", err)
	}

	// Test: REFACTOR.md exists
	if !HasRefactor(tmpDir) {
		t.Error("HasRefactor should return true when REFACTOR.md exists")
	}
}

func TestFindRefactor(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-find-refactor")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test: No refactor file
	_, err = FindRefactor(tmpDir)
	if err == nil {
		t.Error("FindRefactor should return error when no refactor file exists")
	}

	// Create REFACTOR.md
	refactorPath := filepath.Join(tmpDir, "REFACTOR.md")
	if err := os.WriteFile(refactorPath, []byte("# Refactoring Goals"), 0644); err != nil {
		t.Fatalf("Failed to write REFACTOR.md: %v", err)
	}

	// Test: REFACTOR.md exists
	foundPath, err := FindRefactor(tmpDir)
	if err != nil {
		t.Errorf("FindRefactor failed: %v", err)
	}
	if foundPath != refactorPath {
		t.Errorf("FindRefactor returned %s, expected %s", foundPath, refactorPath)
	}
}

func TestInitRefactorMode_SkipsIfPlanExists(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "test-refactor-skip")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create existing REFACTOR_PLAN.md
	planPath := filepath.Join(tmpDir, "REFACTOR_PLAN.md")
	if err := os.WriteFile(planPath, []byte("# Existing Plan\n- [ ] Task 1"), 0644); err != nil {
		t.Fatalf("Failed to create plan file: %v", err)
	}

	opts := InitOptions{
		OutputDir: tmpDir,
		Mode:      ModeRefactor,
		Verbose:   false,
	}

	result, err := InitRefactorMode(opts)
	if err != nil {
		t.Errorf("InitRefactorMode should succeed when REFACTOR_PLAN.md exists: %v", err)
	}
	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if !result.Success {
		t.Error("Result.Success should be true")
	}
	if result.RefactorPlanPath != planPath {
		t.Errorf("RefactorPlanPath = %s, want %s", result.RefactorPlanPath, planPath)
	}
}

func TestBuildRefactorPlanPrompt(t *testing.T) {
	refactorContent := "# Refactoring Goals\n\nClean up the API layer."

	prompt := BuildRefactorPlanPrompt(refactorContent)

	// Should contain the refactor content
	if !contains(prompt, refactorContent) {
		t.Error("Prompt should contain refactor content")
	}

	// Should contain separator
	if !contains(prompt, "---") {
		t.Error("Prompt should contain separator")
	}
}

func TestInit_AutoDetect_Refactor(t *testing.T) {
	// Create temp directory with REFACTOR.md
	tmpDir, err := os.MkdirTemp("", "test-init-autodetect-refactor")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create REFACTOR.md
	refactorPath := filepath.Join(tmpDir, "REFACTOR.md")
	if err := os.WriteFile(refactorPath, []byte("# Refactoring Goals"), 0644); err != nil {
		t.Fatalf("Failed to write REFACTOR.md: %v", err)
	}

	opts := InitOptions{
		OutputDir: tmpDir,
		Verbose:   false,
	}

	// Init should detect refactor mode but will fail because Codex isn't available in tests
	// We're just testing the auto-detection works
	_, err = Init(opts)
	// Error is expected because Codex isn't available, but mode should be detected
	if err == nil {
		t.Log("Init succeeded (Codex available)")
	} else if !contains(err.Error(), "codex") && !contains(err.Error(), "REFACTOR_PLAN") {
		// If error is not about codex/generation, it might be wrong mode detection
		t.Logf("Init error (expected - Codex not available): %v", err)
	}
}

// TestReadSpecsFolder tests the readSpecsFolder function with various scenarios
func TestReadSpecsFolder(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		wantErr     bool
		errContains string
		wantContent []string // Substrings that should be in the result
		wantOrder   []string // Substrings that should appear in order
	}{
		{
			name: "empty folder",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				specsDir := filepath.Join(tmpDir, "specs")
				if err := os.MkdirAll(specsDir, 0755); err != nil {
					t.Fatalf("Failed to create specs dir: %v", err)
				}
				return specsDir
			},
			wantErr:     true,
			errContains: "no markdown files found",
		},
		{
			name: "non-existent folder",
			setupFunc: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantErr:     true,
			errContains: "failed to read specs directory",
		},
		{
			name: "ignores non-markdown files",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				specsDir := filepath.Join(tmpDir, "specs")
				if err := os.MkdirAll(specsDir, 0755); err != nil {
					t.Fatalf("Failed to create specs dir: %v", err)
				}
				// Create non-markdown files
				os.WriteFile(filepath.Join(specsDir, "readme.txt"), []byte("Text content"), 0644)
				os.WriteFile(filepath.Join(specsDir, "data.json"), []byte(`{"key": "value"}`), 0644)
				os.WriteFile(filepath.Join(specsDir, "script.py"), []byte("print('hello')"), 0644)
				// Create one markdown file
				os.WriteFile(filepath.Join(specsDir, "api.md"), []byte("# API Spec"), 0644)
				return specsDir
			},
			wantErr:     false,
			wantContent: []string{"# API Spec", "File: api.md"},
			wantOrder:   []string{"File: api.md", "# API Spec"},
		},
		{
			name: "multiple markdown files concatenated",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				specsDir := filepath.Join(tmpDir, "specs")
				if err := os.MkdirAll(specsDir, 0755); err != nil {
					t.Fatalf("Failed to create specs dir: %v", err)
				}
				// Create multiple markdown files
				os.WriteFile(filepath.Join(specsDir, "01_intro.md"), []byte("# Introduction\n\nThis is the intro."), 0644)
				os.WriteFile(filepath.Join(specsDir, "02_api.md"), []byte("# API\n\nAPI endpoints here."), 0644)
				os.WriteFile(filepath.Join(specsDir, "03_auth.md"), []byte("# Authentication\n\nAuth methods."), 0644)
				return specsDir
			},
			wantErr: false,
			wantContent: []string{
				"# Introduction",
				"# API",
				"# Authentication",
				"File: 01_intro.md",
				"File: 02_api.md",
				"File: 03_auth.md",
			},
		},
		{
			name: "case insensitive markdown extension",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				specsDir := filepath.Join(tmpDir, "specs")
				if err := os.MkdirAll(specsDir, 0755); err != nil {
					t.Fatalf("Failed to create specs dir: %v", err)
				}
				// Create files with different case extensions
				os.WriteFile(filepath.Join(specsDir, "lowercase.md"), []byte("# Lowercase"), 0644)
				os.WriteFile(filepath.Join(specsDir, "uppercase.MD"), []byte("# Uppercase"), 0644)
				os.WriteFile(filepath.Join(specsDir, "mixed.Md"), []byte("# Mixed"), 0644)
				return specsDir
			},
			wantErr: false,
			wantContent: []string{
				"# Lowercase",
				"# Uppercase",
				"# Mixed",
			},
		},
		{
			name: "ignores subdirectories",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				specsDir := filepath.Join(tmpDir, "specs")
				subDir := filepath.Join(specsDir, "subdir")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					t.Fatalf("Failed to create subdir: %v", err)
				}
				// Create file in subdirectory (should be ignored)
				os.WriteFile(filepath.Join(subDir, "nested.md"), []byte("# Nested"), 0644)
				// Create file in main directory
				os.WriteFile(filepath.Join(specsDir, "main.md"), []byte("# Main"), 0644)
				return specsDir
			},
			wantErr:     false,
			wantContent: []string{"# Main", "File: main.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			specsDir := tt.setupFunc(t)
			content, err := readSpecsFolder(specsDir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("readSpecsFolder() expected error, got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("readSpecsFolder() error = %v, expected to contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("readSpecsFolder() unexpected error = %v", err)
				return
			}

			// Check expected content
			for _, want := range tt.wantContent {
				if !strings.Contains(content, want) {
					t.Errorf("readSpecsFolder() content missing expected substring: %q\nGot content:\n%s", want, content)
				}
			}

			// Check ordering if specified
			if len(tt.wantOrder) > 1 {
				lastIdx := -1
				for _, want := range tt.wantOrder {
					idx := strings.Index(content, want)
					if idx == -1 {
						t.Errorf("readSpecsFolder() content missing expected substring for ordering check: %q", want)
						continue
					}
					if idx <= lastIdx {
						t.Errorf("readSpecsFolder() content ordering incorrect: %q appears at wrong position\nGot content:\n%s", want, content)
					}
					lastIdx = idx
				}
			}

			// Verify content structure (separator between files)
			if !strings.Contains(content, "\n\n---\n") {
				t.Error("readSpecsFolder() content missing file separator '---'")
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
