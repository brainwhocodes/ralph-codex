package project

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/brainwhocodes/ralph-codex/internal/codex"
	"github.com/charmbracelet/log"
)

// CodexOptions configures how Codex is invoked
type CodexOptions struct {
	Prompt      string // The prompt to send to Codex
	WorkingDir  string // Working directory for execution
	Verbose     bool   // Enable verbose logging
	StreamToTTY bool   // Whether to stream output to TTY
	PassAsArg   bool   // Whether to pass prompt as argument instead of stdin
}

// CodexResult holds the result of a Codex invocation
type CodexResult struct {
	Content     string            // Extracted message content
	RawOutput   string            // Full raw output
	Events      []codex.Event     // Parsed events
	ToolCalls   []CodexToolCall   // Tool calls made
}

// CodexToolCall represents a tool call made by Codex
type CodexToolCall struct {
	Name   string
	Target string
	Status string
}

// RunCodex executes Codex CLI with the given options and returns the result
// This is the unified helper for all Codex invocations in the project package
func RunCodex(opts CodexOptions) (*CodexResult, error) {
	args := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
		"--sandbox", "danger-full-access",
	}

	// If passing prompt as argument, append it
	if opts.PassAsArg && opts.Prompt != "" {
		args = append(args, opts.Prompt)
	}

	cmd := exec.Command("codex", args...)

	// Set working directory if specified
	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	}

	// Pass prompt via stdin unless using argument mode
	if !opts.PassAsArg && opts.Prompt != "" {
		cmd.Stdin = strings.NewReader(opts.Prompt)
	}

	if opts.Verbose {
		log.Debug("Calling Codex...", "args", strings.Join(args, " "))
	}

	// Get stdout pipe for streaming
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start codex: %w", err)
	}

	// Read stderr in background
	var stderrBuf strings.Builder
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				break
			}
			if n > 0 {
				stderrBuf.WriteString(string(buf[:n]))
				if opts.StreamToTTY {
					fmt.Fprint(os.Stderr, string(buf[:n]))
				}
			}
		}
	}()

	// Process JSONL output
	result := &CodexResult{
		Events:    make([]codex.Event, 0),
		ToolCalls: make([]CodexToolCall, 0),
	}

	var rawOutput strings.Builder
	var content strings.Builder

	// Use large buffer for scanner to handle big JSONL lines
	const maxScannerBuffer = 1024 * 1024 // 1MB
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, maxScannerBuffer), maxScannerBuffer)

	for scanner.Scan() {
		line := scanner.Text()
		rawOutput.WriteString(line)
		rawOutput.WriteString("\n")

		if line == "" {
			continue
		}

		// Parse JSONL
		var event codex.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Not JSON, might be raw output
			content.WriteString(line)
			continue
		}

		result.Events = append(result.Events, event)

		// Use unified event parser
		parsed := codex.ParseEvent(event)
		if parsed == nil {
			continue
		}

		switch parsed.Type {
		case "reasoning":
			if opts.Verbose && parsed.Text != "" {
				log.Debug(parsed.Text)
			}
		case "message", "delta":
			if parsed.Text != "" {
				if opts.StreamToTTY {
					fmt.Print(parsed.Text)
				}
				content.WriteString(parsed.Text)
			}
		case "tool_call", "tool_result":
			if parsed.ToolName != "" {
				result.ToolCalls = append(result.ToolCalls, CodexToolCall{
					Name:   parsed.ToolName,
					Target: parsed.ToolTarget,
					Status: parsed.ToolStatus,
				})
				if opts.Verbose {
					log.Debug("Tool", "name", parsed.ToolName, "target", parsed.ToolTarget, "status", parsed.ToolStatus)
				}
			}
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading codex output: %w", err)
	}

	if opts.StreamToTTY {
		fmt.Println() // Final newline after streaming
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		errMsg := stderrBuf.String()
		if errMsg == "" {
			errMsg = rawOutput.String()
		}
		return nil, fmt.Errorf("codex execution failed: %w\nOutput: %s", err, errMsg)
	}

	result.Content = content.String()
	result.RawOutput = rawOutput.String()

	return result, nil
}

// RunCodexSimple is a convenience wrapper that runs Codex and returns just the content
func RunCodexSimple(prompt string, verbose bool) (string, error) {
	result, err := RunCodex(CodexOptions{
		Prompt:      prompt,
		Verbose:     verbose,
		StreamToTTY: true,
	})
	if err != nil {
		return "", err
	}

	if result.Content == "" {
		return "", fmt.Errorf("no content generated from Codex")
	}

	return result.Content, nil
}

// RunCodexInDir runs Codex in a specific directory with TTY streaming
func RunCodexInDir(prompt, dir string) error {
	_, err := RunCodex(CodexOptions{
		Prompt:      prompt,
		WorkingDir:  dir,
		PassAsArg:   true,
		StreamToTTY: true,
	})
	return err
}

// RunCodexWithDirectStream runs Codex with direct IO streaming (no parsing)
// Used when raw streaming to TTY is needed without JSONL processing
func RunCodexWithDirectStream(prompt, workingDir string) error {
	args := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
		"--sandbox", "danger-full-access",
		prompt,
	}

	cmd := exec.Command("codex", args...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start codex: %w", err)
	}

	// Stream directly to TTY
	go func() {
		if _, err := io.Copy(os.Stdout, stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error copying stdout: %v\n", err)
		}
	}()
	go func() {
		if _, err := io.Copy(os.Stderr, stderr); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error copying stderr: %v\n", err)
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("codex execution failed: %w", err)
	}

	return nil
}
