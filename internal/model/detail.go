package model

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/rdp"
	"github.com/felipem/julssh/internal/ssh"
	"github.com/felipem/julssh/internal/store"
	"github.com/felipem/julssh/internal/styles"
	"github.com/felipem/julssh/internal/vnc"
)

type DetailModel struct {
	connID     string
	store      *store.Store
	confirming bool
	width      int
	height     int
}

func NewDetail(conn store.Connection, s *store.Store) DetailModel {
	return DetailModel{connID: conn.ID, store: s}
}

func (m DetailModel) current() (store.Connection, bool) {
	for _, c := range m.store.Connections {
		if c.ID == m.connID {
			return c, true
		}
	}
	return store.Connection{}, false
}

func (m DetailModel) Init() tea.Cmd { return nil }

func (m DetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.confirming {
			return m.handleConfirm(msg)
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m DetailModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	conn, ok := m.current()
	if !ok {
		return m, func() tea.Msg { return MsgPopView{} }
	}
	switch msg.String() {
	case "esc", "q":
		return m, func() tea.Msg { return MsgPopView{} }
	case "e":
		form := NewForm(m.store, &conn)
		return m, func() tea.Msg { return MsgPushView{View: form} }
	case "d":
		m.confirming = true
		return m, nil
	case "c":
		_ = m.store.RecordConnect(conn.ID)
		return m, connectCmd(conn)
	}
	return m, nil
}

func connectCmd(c store.Connection) tea.Cmd {
	switch c.Type {
	case "rdp":
		return rdp.Connect(c)
	case "vnc":
		return vnc.Connect(c)
	default:
		return ssh.Connect(c)
	}
}

func (m DetailModel) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s", "y":
		if err := m.store.Delete(m.connID); err != nil {
			return m, func() tea.Msg { return MsgError{Err: err} }
		}
		return m, func() tea.Msg { return MsgPopView{} }
	case "n":
		m.confirming = false
	}
	return m, nil
}

func (m DetailModel) View() string {
	conn, ok := m.current()
	if !ok {
		return styles.ErrText.Render("conexión no encontrada")
	}

	var b strings.Builder

	typeLabel := strings.ToUpper(conn.Type)
	b.WriteString(styles.Title.Render(conn.Name + "  " + styles.Tag.Render(typeLabel)))
	b.WriteString("\n")
	b.WriteString(styles.MutedText.Render(strings.Repeat("─", 40)))
	b.WriteString("\n\n")

	field := func(label, value string) {
		if value == "" {
			return
		}
		b.WriteString(styles.FieldLabel.Render(label+":"))
		b.WriteString("  " + value + "\n")
	}

	host := conn.Host
	defaultPort := map[string]int{"ssh": 22, "rdp": 3389, "vnc": 5900}
	if conn.Port != 0 && conn.Port != defaultPort[conn.Type] {
		host = fmt.Sprintf("%s:%d", conn.Host, conn.Port)
	}
	field("Host", host)
	field("Usuario", conn.User)

	switch conn.Type {
	case "ssh":
		field("Identity File", conn.IdentityFile)
	case "rdp":
		field("Dominio", conn.Domain)
	}

	field("Descripción", conn.Description)
	if len(conn.Tags) > 0 {
		tags := ""
		for _, t := range conn.Tags {
			tags += styles.Tag.Render(t) + " "
		}
		b.WriteString(styles.FieldLabel.Render("Tags:"))
		b.WriteString("  " + tags + "\n")
	}
	b.WriteString("\n")
	b.WriteString(styles.MutedText.Render(fmt.Sprintf("Creado: %s", conn.CreatedAt.Format("2006-01-02 15:04"))))
	b.WriteString("\n")

	if m.confirming {
		b.WriteString("\n")
		b.WriteString(styles.ErrText.Render(fmt.Sprintf("Borrar %q? (s/n)", conn.Name)))
	} else {
		b.WriteString(styles.Footer.Render("[c]conectar  [e]editar  [d]borrar  [Esc]volver"))
	}

	return b.String()
}
