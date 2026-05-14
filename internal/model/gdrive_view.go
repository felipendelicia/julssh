package model

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/gdrive"
	"github.com/felipem/julssh/internal/store"
	"github.com/felipem/julssh/internal/styles"
)

type driveViewState int

const (
	driveStateLoading      driveViewState = iota
	driveStateConnected
	driveStateDisconnected
)

// msgDriveUserInfo carries the result of an async user-info load.
type msgDriveUserInfo struct {
	info *gdrive.UserInfo
	err  error
}

// msgDriveLogoutDone carries the result of an async logout.
type msgDriveLogoutDone struct{ err error }

// msgDriveLoginDone carries the result of an async login.
type msgDriveLoginDone struct{ err error }

// DriveViewModel is the Google Drive management screen.
type DriveViewModel struct {
	store   *store.Store
	state   driveViewState
	info    *gdrive.UserInfo
	infoErr string
}

func NewDriveView(s *store.Store) DriveViewModel {
	return DriveViewModel{store: s, state: driveStateLoading}
}

// Init fires an async cmd to check login state and load user info.
func (m DriveViewModel) Init() tea.Cmd {
	return func() tea.Msg {
		if !gdrive.IsLoggedIn() {
			return msgDriveUserInfo{}
		}
		info, err := gdrive.GetUserInfo()
		return msgDriveUserInfo{info: info, err: err}
	}
}

func (m DriveViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case msgDriveUserInfo:
		if msg.info == nil && msg.err == nil {
			m.state = driveStateDisconnected
			return m, nil
		}
		m.state = driveStateConnected
		m.info = msg.info
		if msg.err != nil {
			m.infoErr = msg.err.Error()
		}
		return m, nil

	case msgDriveLogoutDone:
		if msg.err != nil {
			return m, func() tea.Msg { return MsgError{Err: msg.err} }
		}
		m.state = driveStateDisconnected
		m.info = nil
		m.infoErr = ""
		return m, nil

	case msgDriveLoginDone:
		if msg.err != nil {
			return m, func() tea.Msg { return MsgError{Err: msg.err} }
		}
		m.state = driveStateLoading
		return m, m.Init()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m DriveViewModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		return m, func() tea.Msg { return MsgPopView{} }

	case "S":
		if m.state != driveStateConnected {
			return m, nil
		}
		s := m.store
		return m, func() tea.Msg {
			if err := gdrive.Push(s); err != nil {
				return MsgDrivePushDone{Err: err}
			}
			return MsgDrivePushDone{}
		}

	case "L":
		switch m.state {
		case driveStateDisconnected:
			return m, func() tea.Msg {
				if err := gdrive.Login(); err != nil {
					return msgDriveLoginDone{err: err}
				}
				return msgDriveLoginDone{}
			}
		case driveStateConnected:
			s := m.store
			return m, func() tea.Msg {
				added, err := gdrive.Pull(s)
				return MsgDrivePullDone{Added: added, Err: err}
			}
		}

	case "O":
		if m.state != driveStateConnected {
			return m, nil
		}
		return m, func() tea.Msg {
			if err := gdrive.Logout(); err != nil {
				return msgDriveLogoutDone{err: err}
			}
			return msgDriveLogoutDone{}
		}
	}
	return m, nil
}

func (m DriveViewModel) View() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Google Drive"))
	b.WriteString("\n\n")

	switch m.state {
	case driveStateLoading:
		b.WriteString(styles.MutedText.Render("  Cargando...") + "\n")

	case driveStateDisconnected:
		b.WriteString(styles.FieldLabel.Render("Estado:"))
		b.WriteString("  " + styles.ErrText.Render("Desconectado") + "\n\n")
		b.WriteString(styles.Footer.Render("[L] Iniciar sesión con Google  [Esc] volver"))

	case driveStateConnected:
		b.WriteString(styles.FieldLabel.Render("Estado:"))
		b.WriteString("  " + styles.MutedText.Render("Conectado") + "\n")
		if m.info != nil {
			b.WriteString(styles.FieldLabel.Render("Cuenta:"))
			b.WriteString("  " + m.info.Email + "\n")
			b.WriteString(styles.FieldLabel.Render("Nombre:"))
			b.WriteString("  " + m.info.Name + "\n")
		} else if m.infoErr != "" {
			b.WriteString(styles.ErrText.Render("  Error al cargar info: "+m.infoErr) + "\n")
		}
		b.WriteString("\n")
		b.WriteString(styles.Footer.Render("[S] Subir conexiones  [L] Bajar conexiones  [O] Desloguear  [Esc] volver"))
	}

	return b.String()
}
