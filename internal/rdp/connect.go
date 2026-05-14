package rdp

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/logger"
	"github.com/felipem/julssh/internal/store"
)

const launchTimeout = 2 * time.Second

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
		args := buildArgs(c)
		logger.Log("rdp: exec xfreerdp %v", args)
		var stderr bytes.Buffer
		cmd := exec.Command("xfreerdp", args...)
		cmd.Stderr = &stderr
		if err := cmd.Start(); err != nil {
			logger.Log("rdp: Start() error: %v", err)
			return MsgRDPLaunched{Err: err}
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			errOut := stderr.String()
			logger.Log("rdp: exited quickly — err=%v stderr=%q", err, errOut)
			if err != nil {
				msg := firstErrLine(errOut)
				if msg == "" {
					msg = err.Error()
				}
				return MsgRDPLaunched{Err: fmt.Errorf("xfreerdp: %s", msg)}
			}
			return MsgRDPLaunched{}
		case <-time.After(launchTimeout):
			logger.Log("rdp: process still running after 1s — assuming connected")
			return MsgRDPLaunched{}
		}
	}
}

func firstErrLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "[ERROR]") {
			if i := strings.LastIndex(line, "] - "); i >= 0 {
				return strings.TrimSpace(line[i+4:])
			}
		}
	}
	return ""
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
	if c.Password != "" {
		args = append(args, "/p:"+c.Password)
	}
	return append(args, "/dynamic-resolution", "/cert-ignore")
}
