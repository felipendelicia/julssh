package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/model"
	"github.com/felipem/julssh/internal/store"
)

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("no se puede determinar config dir: %v", err)
	}
	storePath := filepath.Join(configDir, "julssh", "connections.json")

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
