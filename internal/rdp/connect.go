package rdp

import (
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/store"
)

type MsgRDPLaunched struct {
	Err      error
	Bin      string   // set when client binary not found
	Pkg      string   // apt package to install
	RetryCmd tea.Cmd  // retry after install
}

func Connect(c store.Connection) tea.Cmd {
	return func() tea.Msg {
		if _, err := exec.LookPath("xfreerdp"); err != nil {
			return MsgRDPLaunched{
				Err:      fmt.Errorf("xfreerdp no encontrado"),
				Bin:      "xfreerdp",
				Pkg:      "freerdp2-x11",
				RetryCmd: Connect(c),
			}
		}
		cmd := exec.Command("xfreerdp", buildArgs(c)...)
		if err := cmd.Start(); err != nil {
			return MsgRDPLaunched{Err: err}
		}
		return MsgRDPLaunched{}
	}
}

func buildArgs(c store.Connection) []string {
	port := 3389
	if c.Port != 0 {
		port = c.Port
	}
	args := []string{fmt.Sprintf("/v:%s:%d", c.Host, port)}
	if c.User != "" {
		args = append(args, "/u:"+c.User)
	}
	if c.Domain != "" {
		args = append(args, "/d:"+c.Domain)
	}
	return append(args, "/dynamic-resolution")
}
