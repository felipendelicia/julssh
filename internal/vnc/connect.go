package vnc

import (
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/store"
)

type MsgVNCLaunched struct {
	Err      error
	Bin      string   // set when client binary not found
	Pkg      string   // apt package to install
	RetryCmd tea.Cmd  // retry after install
}

func Connect(c store.Connection) tea.Cmd {
	return func() tea.Msg {
		if _, err := exec.LookPath("vncviewer"); err != nil {
			return MsgVNCLaunched{
				Err:      fmt.Errorf("vncviewer no encontrado"),
				Bin:      "vncviewer",
				Pkg:      "tigervnc-viewer",
				RetryCmd: Connect(c),
			}
		}
		port := 5900
		if c.Port != 0 {
			port = c.Port
		}
		cmd := exec.Command("vncviewer", fmt.Sprintf("%s:%d", c.Host, port))
		if err := cmd.Start(); err != nil {
			return MsgVNCLaunched{Err: err}
		}
		return MsgVNCLaunched{}
	}
}
