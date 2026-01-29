package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/brainwhocodes/lisa-loop/internal/circuit"
	"github.com/brainwhocodes/lisa-loop/internal/loop"
)

// Test the loop controller without TUI to see raw output
func main() {
	fmt.Println("=== Lisa Loop Test (No TUI) ===")
	fmt.Println()

	// Change to project directory if provided
	if len(os.Args) > 1 {
		dir := os.Args[1]
		if err := os.Chdir(dir); err != nil {
			fmt.Fprintf(os.Stderr, "Error changing to directory %s: %v\n", dir, err)
			os.Exit(1)
		}
		fmt.Printf("Working directory: %s\n", dir)
	}

	// Create config
	config := loop.Config{
		Backend:      "cli",
		ProjectPath:  ".",
		PromptPath:   "PROMPT.md",
		MaxCalls:     5, // Limit for testing
		Timeout:      120,
		Verbose:      true,
		ResetCircuit: false,
	}

	// Create components
	rateLimiter := loop.NewRateLimiter(config.MaxCalls, 1)
	breaker := circuit.NewBreaker(3, 5)
	controller := loop.NewController(config, rateLimiter, breaker)

	// Set up event callback to print all events
	eventCount := 0
	controller.SetEventCallback(func(event loop.LoopEvent) {
		eventCount++
		timestamp := time.Now().Format("15:04:05")

		switch event.Type {
		case "log":
			// Color code by level
			levelColor := ""
			resetColor := "\033[0m"
			switch event.LogLevel {
			case "INFO":
				levelColor = "\033[34m" // Blue
			case "WARN":
				levelColor = "\033[33m" // Yellow
			case "ERROR":
				levelColor = "\033[31m" // Red
			case "SUCCESS":
				levelColor = "\033[32m" // Green
			}
			fmt.Printf("[%s] %s[%s]%s %s\n", timestamp, levelColor, event.LogLevel, resetColor, event.LogMessage)

		case "loop_update":
			fmt.Printf("[%s] 📊 Loop: %d | Calls: %d | Status: %s | Circuit: %s\n",
				timestamp, event.LoopNumber, event.CallsUsed, event.Status, event.CircuitState)

		default:
			fmt.Printf("[%s] EVENT: %+v\n", timestamp, event)
		}
	})

	// Run the loop
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Println()
	fmt.Println("=== Starting Loop ===")
	fmt.Println()

	err := controller.Run(ctx)

	fmt.Println()
	fmt.Println("=== Loop Finished ===")
	fmt.Printf("Total events: %d\n", eventCount)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Completed successfully")
	}

	// Print stats
	stats := controller.GetStats()
	fmt.Printf("\nStats: %+v\n", stats)
}
