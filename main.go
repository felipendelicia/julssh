package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/logger"
	"github.com/felipem/julssh/internal/model"
	"github.com/felipem/julssh/internal/store"
)

// version is set at build time by GoReleaser via -X main.version={{.Version}}
var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("julssh", version)
		os.Exit(0)
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("no se puede determinar config dir: %v", err)
	}
	storePath := filepath.Join(configDir, "julssh", "connections.json")
	logPath := filepath.Join(configDir, "julssh", "julssh.log")
	_ = logger.Init(logPath)

	s, err := store.Load(storePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error al cargar conexiones: %v\n", err)
		os.Exit(1)
	}

	app := model.NewApp(s)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
