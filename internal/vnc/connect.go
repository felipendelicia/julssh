package vnc

import (
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/store"
)

type MsgVNCLaunched struct{ Err error }

func Connect(c store.Connection) tea.Cmd {
	return func() tea.Msg {
		if _, err := exec.LookPath("vncviewer"); err != nil {
			return MsgVNCLaunched{Err: fmt.Errorf("vncviewer no encontrado — instalá tigervnc-viewer")}
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
