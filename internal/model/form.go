package model

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/store"
	"github.com/felipem/julssh/internal/styles"
)

const (
	fieldName = iota
	fieldHost
	fieldPort
	fieldUser
	fieldIdentityFile
	fieldDescription
	fieldTags
	fieldCount
)

var fieldLabels = [fieldCount]string{
	"Nombre", "Host", "Puerto", "Usuario", "Identity File", "Descripción", "Tags",
}

var fieldPlaceholders = [fieldCount]string{
	"Mi servidor", "hostname o IP", "22", "usuario", "~/.ssh/id_ed25519", "opcional", "tag1, tag2",
}

type FormModel struct {
	fields  [fieldCount]textinput.Model
	focused int
	store   *store.Store
	conn    store.Connection
	editing bool
	errMsg  string
	width   int
	height  int
}

func NewForm(s *store.Store, conn *store.Connection) FormModel {
	var fields [fieldCount]textinput.Model
	for i := range fields {
		ti := textinput.New()
		ti.Placeholder = fieldPlaceholders[i]
		ti.CharLimit = 256
		fields[i] = ti
	}
	fields[fieldPort].CharLimit = 5

	m := FormModel{fields: fields, store: s}

	if conn != nil {
		m.conn = *conn
		m.editing = true
		fields[fieldName].SetValue(conn.Name)
		fields[fieldHost].SetValue(conn.Host)
		if conn.Port != 0 {
			fields[fieldPort].SetValue(fmt.Sprintf("%d", conn.Port))
		}
		fields[fieldUser].SetValue(conn.User)
		fields[fieldIdentityFile].SetValue(conn.IdentityFile)
		fields[fieldDescription].SetValue(conn.Description)
		fields[fieldTags].SetValue(strings.Join(conn.Tags, ", "))
		m.fields = fields
	}

	m.fields[fieldName].Focus()
	return m
}

func (m FormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return MsgPopView{} }

		case "tab", "enter":
			if m.focused == fieldCount-1 {
				return m.trySave()
			}
			m.fields[m.focused].Blur()
			m.focused++
			m.fields[m.focused].Focus()
			return m, textinput.Blink

		case "shift+tab":
			if m.focused > 0 {
				m.fields[m.focused].Blur()
				m.focused--
				m.fields[m.focused].Focus()
			}
			return m, textinput.Blink
		}
	}

	var cmd tea.Cmd
	m.fields[m.focused], cmd = m.fields[m.focused].Update(msg)
	return m, cmd
}

func (m FormModel) trySave() (tea.Model, tea.Cmd) {
	host := strings.TrimSpace(m.fields[fieldHost].Value())
	if host == "" {
		m.errMsg = "Host es requerido"
		return m, nil
	}

	portStr := strings.TrimSpace(m.fields[fieldPort].Value())
	port := 22
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil || p < 1 || p > 65535 {
			m.errMsg = "Puerto inválido (1-65535)"
			return m, nil
		}
		port = p
	}

	rawTags := strings.Split(m.fields[fieldTags].Value(), ",")
	var tags []string
	for _, t := range rawTags {
		if t := strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}

	conn := store.Connection{
		ID:           m.conn.ID,
		Name:         strings.TrimSpace(m.fields[fieldName].Value()),
		Host:         host,
		Port:         port,
		User:         strings.TrimSpace(m.fields[fieldUser].Value()),
		IdentityFile: strings.TrimSpace(m.fields[fieldIdentityFile].Value()),
		Description:  strings.TrimSpace(m.fields[fieldDescription].Value()),
		Tags:         tags,
	}

	return m, func() tea.Msg {
		var err error
		if m.editing {
			err = m.store.Update(conn)
		} else {
			err = m.store.Add(conn)
		}
		if err != nil {
			return MsgError{Err: err}
		}
		return MsgPopView{}
	}
}

func (m FormModel) View() string {
	var b strings.Builder

	title := "Nueva conexión"
	if m.editing {
		title = "Editar: " + m.conn.Name
	}
	b.WriteString(styles.Title.Render(title))
	b.WriteString("\n\n")

	for i := range m.fields {
		label := styles.FieldLabel.Render(fieldLabels[i] + ":")
		var input string
		if i == m.focused {
			input = styles.ActiveInput.Render(m.fields[i].View())
		} else {
			input = styles.InactiveInput.Render(m.fields[i].View())
		}
		b.WriteString(label + "  " + input + "\n")
	}

	if m.errMsg != "" {
		b.WriteString("\n")
		b.WriteString(styles.InlineErr.Render("! " + m.errMsg))
		b.WriteString("\n")
	}

	b.WriteString(styles.Footer.Render("[Tab/Enter]siguiente  [Shift+Tab]anterior  [Enter en Tags]guardar  [Esc]cancelar"))

	return b.String()
}
