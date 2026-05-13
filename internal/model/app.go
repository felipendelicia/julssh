package model

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/rdp"
	"github.com/felipem/julssh/internal/ssh"
	"github.com/felipem/julssh/internal/store"
	"github.com/felipem/julssh/internal/styles"
	"github.com/felipem/julssh/internal/vnc"
)

// Router messages — child views return these as tea.Cmd to navigate.
type MsgPushView struct{ View tea.Model }
type MsgPopView struct{}
type MsgError struct{ Err error }

type msgClearStatus struct{}
type msgStatusOK struct{ text string }

func clearStatusCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return msgClearStatus{}
	})
}

type AppModel struct {
	stack     []tea.Model
	store     *store.Store
	statusMsg string
	statusOK  bool
}

func NewApp(s *store.Store) AppModel {
	return AppModel{
		stack: []tea.Model{NewList(s)},
		store: s,
	}
}

func (a AppModel) Init() tea.Cmd {
	return nil
}

func (a AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case MsgPushView:
		a.stack = append(a.stack, msg.View)
		return a, nil

	case MsgPopView:
		if len(a.stack) > 1 {
			a.stack = a.stack[:len(a.stack)-1]
		}
		return a, nil

	case MsgError:
		a.statusMsg = msg.Err.Error()
		a.statusOK = false
		return a, clearStatusCmd()

	case msgClearStatus:
		a.statusMsg = ""
		return a, nil

	case msgStatusOK:
		a.statusMsg = msg.text
		a.statusOK = true
		return a, clearStatusCmd()

	case ssh.MsgSSHDone:
		if msg.Err != nil {
			a.statusMsg = "ssh: " + msg.Err.Error()
			a.statusOK = false
			return a, clearStatusCmd()
		}
		return a, nil

	case rdp.MsgRDPLaunched:
		if msg.Err != nil {
			if msg.Bin != "" {
				a.stack = append(a.stack, NewInstallView(msg.Bin, msg.Pkg, msg.RetryCmd))
				return a, nil
			}
			a.statusMsg = "rdp: " + msg.Err.Error()
			a.statusOK = false
			return a, clearStatusCmd()
		}
		return a, func() tea.Msg { return msgStatusOK{text: "RDP abierto"} }

	case vnc.MsgVNCLaunched:
		if msg.Err != nil {
			if msg.Bin != "" {
				a.stack = append(a.stack, NewInstallView(msg.Bin, msg.Pkg, msg.RetryCmd))
				return a, nil
			}
			a.statusMsg = "vnc: " + msg.Err.Error()
			a.statusOK = false
			return a, clearStatusCmd()
		}
		return a, func() tea.Msg { return msgStatusOK{text: "VNC abierto"} }

	case tea.WindowSizeMsg:
		var cmds []tea.Cmd
		for i, v := range a.stack {
			updated, cmd := v.Update(msg)
			a.stack[i] = updated
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)
	}

	top := len(a.stack) - 1
	updated, cmd := a.stack[top].Update(msg)
	a.stack[top] = updated
	return a, cmd
}

func (a AppModel) View() string {
	view := a.stack[len(a.stack)-1].View()
	if a.statusMsg != "" {
		if a.statusOK {
			view += "\n" + styles.MutedText.Render("✓ "+a.statusMsg)
		} else {
			view += "\n" + styles.ErrText.Render("! "+a.statusMsg)
		}
	}
	return view
}
