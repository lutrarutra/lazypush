package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lutrarutra/lazypush/internal/config"
	"github.com/lutrarutra/lazypush/internal/llm"
	"github.com/lutrarutra/lazypush/internal/tui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "auth":
			runAuth()
			return
		case "reset":
			runReset()
			return
		}
	}

	forceLogin := len(os.Args) > 1 && os.Args[1] == "login"

	model := tui.NewModel(forceLogin)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runAuth() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.APIKey == "" {
		fmt.Println("No API provider configured.")
		fmt.Println("Run 'lazypush login' to set one up.")
		os.Exit(1)
	}

	fmt.Printf("Testing: %s / model=%s ... ", cfg.APIURL, cfg.Model)

	client := llm.New(cfg.APIURL, cfg.APIKey, cfg.Model)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		fmt.Printf("FAILED\n  %v\n", err)
		os.Exit(1)
	}

	fmt.Println("OK")
}

func runReset() {
	if !config.Exists() {
		fmt.Println("No config file found.")
		return
	}

	fmt.Printf("Delete API config at %s?\n", config.ConfigPath())
	fmt.Print("y/N: ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" && answer != "yes" {
		fmt.Println("Cancelled.")
		return
	}

	if err := config.Delete(); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Config deleted.")
}
