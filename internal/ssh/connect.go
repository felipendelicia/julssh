package ssh

import (
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/store"
)

type MsgSSHDone struct{ Err error }

func buildArgs(conn store.Connection) []string {
	var args []string
	if conn.User != "" {
		args = append(args, fmt.Sprintf("%s@%s", conn.User, conn.Host))
	} else {
		args = append(args, conn.Host)
	}
	if conn.Port != 0 && conn.Port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", conn.Port))
	}
	if conn.IdentityFile != "" {
		args = append(args, "-i", conn.IdentityFile)
	}
	return args
}

func Connect(conn store.Connection) tea.Cmd {
	args := buildArgs(conn)
	c := exec.Command("ssh", args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return MsgSSHDone{Err: err}
	})
}
