package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/brainwhocodes/ralph-codex/internal/circuit"
	"github.com/brainwhocodes/ralph-codex/internal/codex"
	"github.com/brainwhocodes/ralph-codex/internal/loop"
	"github.com/brainwhocodes/ralph-codex/internal/project"
	"github.com/brainwhocodes/ralph-codex/internal/tui"
)

func main() {
	// Find command and separate from flags
	args := os.Args[1:]
	command, flagArgs := extractCommand(args)

	var (
		projectDir string
		promptFile string
		maxCalls   int
		timeout    int
		useMonitor bool
		verbose    bool

		setupName   string
		setupPrompt string
		setupInit   bool
		withGit     bool
		importSrc   string
		importName  string
	)

	fs := flag.NewFlagSet("ralph", flag.ExitOnError)
	fs.BoolVar(&useMonitor, "monitor", false, "Enable integrated monitoring")
	fs.BoolVar(&verbose, "verbose", false, "Verbose output")

	fs.StringVar(&projectDir, "project", ".", "Project directory")
	fs.StringVar(&promptFile, "prompt", "PROMPT.md", "Prompt file")
	fs.IntVar(&maxCalls, "calls", 3, "Max loop iterations (default: 3)")
	fs.IntVar(&timeout, "timeout", 600, "Codex timeout (seconds)")

	fs.StringVar(&setupName, "name", "", "Project name (for setup command)")
	fs.StringVar(&setupPrompt, "description", "", "Project description for Codex to generate customized templates")
	fs.BoolVar(&setupInit, "init", false, "Initialize in current directory (for existing projects)")
	fs.BoolVar(&withGit, "git", true, "Initialize git repository (for setup command)")

	fs.StringVar(&importSrc, "source", "", "Source file to import (for import command)")
	fs.StringVar(&importName, "import-name", "", "Project name (for import command, auto-detect if empty)")

	fs.Usage = printHelp

	if err := fs.Parse(flagArgs); err != nil {
		os.Exit(1)
	}

	switch command {
	case "setup":
		handleSetupCommand(setupName, setupPrompt, setupInit, withGit, verbose)
	case "import":
		handleImportCommand(importSrc, importName, projectDir, verbose)
	case "status":
		handleStatusCommand(projectDir)
	case "reset-circuit":
		handleResetCircuitCommand(projectDir)
	case "run", "help", "version":
		handleSubcommands(command, projectDir, promptFile, maxCalls, timeout, useMonitor, verbose)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n\n", command)
		printHelp()
		os.Exit(1)
	}
}

func handleSubcommands(command, projectDir, promptFile string, maxCalls, timeout int, useMonitor, verbose bool) {
	switch command {
	case "help", "--help", "-h":
		printHelp()
		os.Exit(0)
	case "version", "--version":
		fmt.Println("Ralph Codex v1.0.0")
		fmt.Println("Charm TUI scaffold - Complete")
		os.Exit(0)
	default:
		handleRunCommand(projectDir, promptFile, maxCalls, timeout, useMonitor, verbose)
	}
}

func handleSetupCommand(projectName string, prompt string, init bool, withGit bool, verbose bool) {
	if projectName == "" && !init {
		fmt.Fprintln(os.Stderr, "Error: --name is required for setup command (or use --init for current directory)")
		os.Exit(1)
	}

	// If --init is used without --name, use current directory name
	if init && projectName == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not get current directory: %v\n", err)
			os.Exit(1)
		}
		projectName = wd
	}

	opts := project.SetupOptions{
		ProjectName: projectName,
		TemplateDir: "",
		WithGit:     withGit,
		Verbose:     verbose,
		Prompt:      prompt,
		Init:        init,
	}

	result, err := project.Setup(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up project: %v\n", err)
		os.Exit(1)
	}

	if !result.Success {
		fmt.Fprintf(os.Stderr, "Project setup failed\n")
		os.Exit(1)
	}

	fmt.Printf("✅ Project created successfully!\n")
	fmt.Printf("   Location: %s\n", result.ProjectPath)
	fmt.Printf("   Files created: %d\n", len(result.FilesCreated))
	if result.GitInitialized {
		fmt.Printf("   Git repository initialized\n")
	}
	fmt.Println("\nNext steps:")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Println("  ralph --monitor")
}

func handleImportCommand(sourcePath string, projectName string, outputDir string, verbose bool) {
	if sourcePath == "" {
		fmt.Fprintln(os.Stderr, "Error: --source is required for import command")
		os.Exit(1)
	}

	if !project.IsSupportedFormat(sourcePath) {
		fmt.Fprintf(os.Stderr, "Error: unsupported file format: %s\n", sourcePath)
		fmt.Fprintln(os.Stderr, "Supported formats:", project.SupportedFormats())
		os.Exit(1)
	}

	opts := project.ImportOptions{
		SourcePath:    sourcePath,
		ProjectName:   projectName,
		OutputDir:     outputDir,
		Verbose:       verbose,
		ConvertFormat: "",
	}

	result, err := project.ImportPRD(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error importing PRD: %v\n", err)
		os.Exit(1)
	}

	if !result.Success {
		fmt.Fprintf(os.Stderr, "Import failed\n")
		os.Exit(1)
	}

	fmt.Printf("✅ Import completed successfully!\n")
	fmt.Printf("   Project: %s\n", result.ProjectName)
	fmt.Printf("   Files created: %d\n", len(result.FilesCreated))
	fmt.Printf("   Converted from: %s\n", result.ConvertedFrom)

	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	fmt.Println(result.GetConversionSummary())
	fmt.Println("\nNext steps:")
	fmt.Println("  ralph --monitor")
}

func handleStatusCommand(projectPath string) {
	if err := os.Chdir(projectPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing to project directory: %v\n", err)
		os.Exit(1)
	}

	if err := project.ValidateProject(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'ralph setup' to create a new project\n")
		os.Exit(1)
	}

	fmt.Println("✅ Valid Ralph Codex project")

	projectRoot, err := project.GetProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding project root: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("   Project root: %s\n", projectRoot)

	tasks, err := loop.LoadFixPlan()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load @fix_plan.md: %v\n", err)
	} else {
		completed := 0
		for _, task := range tasks {
			if len(task) > 0 && task[0] == '[' {
				completed++
			}
		}
		fmt.Printf("   Tasks: %d/%d completed\n", completed, len(tasks))
	}
}

func handleResetCircuitCommand(projectPath string) {
	if err := os.Chdir(projectPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing to project directory: %v\n", err)
		os.Exit(1)
	}

	breaker := circuit.NewBreaker(3, 5)
	if err := breaker.Reset(); err != nil {
		fmt.Fprintf(os.Stderr, "Error resetting circuit breaker: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Circuit breaker reset successfully")
	fmt.Println("   State: CLOSED")
	fmt.Println("   Ready to resume loop")
	fmt.Println("\nNext step:")
	fmt.Println("  ralph --monitor")
}

func handleRunCommand(projectPath string, promptFile string, maxCalls int, timeout int, useMonitor bool, verbose bool) {
	if err := os.Chdir(projectPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing to project directory: %v\n", err)
		os.Exit(1)
	}

	if err := project.ValidateProject(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'ralph setup' to create a new project\n")
		os.Exit(1)
	}

	config := loop.Config{
		Backend:      "cli",
		ProjectPath:  projectPath,
		PromptPath:   promptFile,
		MaxCalls:     maxCalls,
		Timeout:      timeout,
		Verbose:      verbose,
		ResetCircuit: false,
	}

	rateLimiter := loop.NewRateLimiter(config.MaxCalls, 1)
	breaker := circuit.NewBreaker(3, 5)
	controller := loop.NewController(config, rateLimiter, breaker)

	ctx, cancel := context.WithCancel(context.Background())
	setupGracefulShutdown(cancel, controller)

	if useMonitor {
		runWithMonitor(ctx, controller, config, verbose)
	} else {
		runHeadless(ctx, controller, config, verbose)
	}
}

func runWithMonitor(ctx context.Context, controller *loop.Controller, config loop.Config, verbose bool) {
	fmt.Printf("🚀 Starting Ralph Codex with TUI monitoring (max %d calls)...\n", config.MaxCalls)

	tuiConfig := codex.Config{
		Backend:      config.Backend,
		ProjectPath:  config.ProjectPath,
		PromptPath:   config.PromptPath,
		MaxCalls:     config.MaxCalls,
		Timeout:      config.Timeout,
		Verbose:      config.Verbose,
		ResetCircuit: false,
	}

	program := tui.NewProgram(tuiConfig, controller)
	if err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func runHeadless(ctx context.Context, controller *loop.Controller, config loop.Config, verbose bool) {
	fmt.Println("🚀 Starting Ralph Codex in headless mode...")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Set up event callback to print logs in headless mode
	controller.SetEventCallback(func(event loop.LoopEvent) {
		switch event.Type {
		case "log":
			levelEmoji := ""
			switch event.LogLevel {
			case "INFO":
				levelEmoji = "ℹ️ "
			case "WARN":
				levelEmoji = "⚠️ "
			case "ERROR":
				levelEmoji = "❌"
			case "SUCCESS":
				levelEmoji = "✅"
			}
			fmt.Printf("%s %s\n", levelEmoji, event.LogMessage)
		case "loop_update":
			if verbose {
				fmt.Printf("📊 Loop %d | Calls: %d | Status: %s | Circuit: %s\n",
					event.LoopNumber, event.CallsUsed, event.Status, event.CircuitState)
			}
		}
	})

	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		if err := controller.Run(ctx); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Loop error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n✅ Ralph Codex loop completed successfully")
	case <-ctx.Done():
		fmt.Println("\n🛑 Ralph Codex stopped by user")
		os.Exit(0)
	}
}

func setupGracefulShutdown(cancel context.CancelFunc, controller *loop.Controller) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\n\n⚠️  Received signal: %v\n", sig)
		fmt.Println("Performing graceful shutdown...")

		cancel()

		if err := controller.GracefulExit(); err != nil {
			fmt.Fprintf(os.Stderr, "Error during graceful exit: %v\n", err)
		}

		os.Exit(0)
	}()
}

func isFlag(arg string) bool {
	return len(arg) > 0 && arg[0] == '-'
}

func isCommand(arg string) bool {
	validCommands := map[string]bool{
		"run":           true,
		"setup":         true,
		"import":        true,
		"status":        true,
		"reset-circuit": true,
		"help":          true,
		"version":       true,
	}
	return validCommands[arg]
}

func extractCommand(args []string) (string, []string) {
	command := "run"
	flagArgs := []string{}

	for i, arg := range args {
		if isCommand(arg) {
			command = arg
			// Collect flags before and after the command
			flagArgs = append(flagArgs, args[:i]...)
			flagArgs = append(flagArgs, args[i+1:]...)
			return command, flagArgs
		}
	}

	// No command found, use default and all args are flags
	return command, args
}

func printHelp() {
	fmt.Println("Ralph Codex - Autonomous AI Development Loop with Charm TUI")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  ralph [command] [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  run (default)      Run autonomous development loop")
	fmt.Println("  setup              Create a new Ralph-managed project")
	fmt.Println("  import              Import PRD or specification document")
	fmt.Println("  status             Show project status")
	fmt.Println("  reset-circuit       Reset circuit breaker state")
	fmt.Println("  help               Show this help")
	fmt.Println("  version            Show version")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --project <path>        Project directory (default: .)")
	fmt.Println("  --prompt <file>         Prompt file (default: PROMPT.md)")
	fmt.Println("  --calls <number>        Max loop iterations (default: 3)")
	fmt.Println("  --timeout <seconds>      Codex timeout (default: 600)")
	fmt.Println("  --monitor               Enable integrated TUI monitoring")
	fmt.Println("  --verbose              Verbose output")
	fmt.Println("")
	fmt.Println("Setup command options:")
	fmt.Println("  --name <project-name>   Project name (required unless --init)")
	fmt.Println("  --description <text>    Project description for Codex to generate templates")
	fmt.Println("  --init                  Initialize in current directory (existing project)")
	fmt.Println("  --git                   Initialize git (default: true)")
	fmt.Println("")
	fmt.Println("Import command options:")
	fmt.Println("  --source <file>         Source file to import (required)")
	fmt.Println("  --import-name <name>    Project name (auto-detect if empty)")
	fmt.Println("")
	fmt.Println("TUI Keybindings:")
	fmt.Println("  q / Ctrl+c   Quit")
	fmt.Println("  r             Run/restart loop")
	fmt.Println("  p             Pause/resume")
	fmt.Println("  l             Toggle log view")
	fmt.Println("  ?             Show help")
}
