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
	fieldDomain
	fieldPassword
	fieldDescription
	fieldTags
	fieldCount
)

var fieldLabels = [fieldCount]string{
	"Nombre", "Host", "Puerto", "Usuario", "Identity File (SSH)", "Dominio (RDP)", "Contraseña", "Descripción", "Tags",
}

var fieldPlaceholders = [fieldCount]string{
	"Mi servidor", "hostname o IP", "", "usuario", "~/.ssh/id_ed25519", "CORP", "", "opcional", "tag1, tag2",
}

var portPlaceholders = map[string]string{
	"ssh": "22",
	"rdp": "3389",
	"vnc": "5900",
}

var defaultPorts = map[string]int{
	"ssh": 22,
	"rdp": 3389,
	"vnc": 5900,
}

var typeOptions = []string{"ssh", "rdp", "vnc"}

type FormModel struct {
	fields         [fieldCount]textinput.Model
	focused        int
	onTypeSelector bool
	typeIdx        int
	store          *store.Store
	conn           store.Connection
	editing        bool
	errMsg         string
	width          int
	height         int
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
	fields[fieldPassword].EchoMode = textinput.EchoPassword
	fields[fieldPassword].EchoCharacter = '•'

	m := FormModel{
		fields:         fields,
		store:          s,
		onTypeSelector: true,
	}
	m.updatePortPlaceholder()

	if conn != nil {
		m.conn = *conn
		m.editing = true

		for i, t := range typeOptions {
			if t == conn.Type {
				m.typeIdx = i
				break
			}
		}

		fields[fieldName].SetValue(conn.Name)
		fields[fieldHost].SetValue(conn.Host)
		if conn.Port != 0 {
			fields[fieldPort].SetValue(fmt.Sprintf("%d", conn.Port))
		}
		fields[fieldUser].SetValue(conn.User)
		fields[fieldIdentityFile].SetValue(conn.IdentityFile)
		fields[fieldDomain].SetValue(conn.Domain)
		fields[fieldPassword].SetValue(conn.Password)
		fields[fieldDescription].SetValue(conn.Description)
		fields[fieldTags].SetValue(strings.Join(conn.Tags, ", "))
		m.fields = fields
		m.onTypeSelector = false
		m.focused = fieldName
		m.fields[fieldName].Focus()
	}

	return m
}

func (m *FormModel) currentType() string {
	return typeOptions[m.typeIdx]
}

func (m *FormModel) updatePortPlaceholder() {
	m.fields[fieldPort].Placeholder = portPlaceholders[m.currentType()]
}

func (m FormModel) visibleFields() []int {
	switch m.currentType() {
	case "rdp":
		return []int{fieldName, fieldHost, fieldPort, fieldUser, fieldDomain, fieldPassword, fieldDescription, fieldTags}
	case "vnc":
		return []int{fieldName, fieldHost, fieldPort, fieldUser, fieldPassword, fieldDescription, fieldTags}
	default:
		return []int{fieldName, fieldHost, fieldPort, fieldUser, fieldIdentityFile, fieldDescription, fieldTags}
	}
}

func (m FormModel) nextVisible(current int) int {
	visible := m.visibleFields()
	for i, f := range visible {
		if f == current && i+1 < len(visible) {
			return visible[i+1]
		}
	}
	return current
}

func (m FormModel) prevVisible(current int) int {
	visible := m.visibleFields()
	for i, f := range visible {
		if f == current && i > 0 {
			return visible[i-1]
		}
	}
	return current
}

func (m FormModel) isLastVisible() bool {
	visible := m.visibleFields()
	return len(visible) > 0 && visible[len(visible)-1] == m.focused
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
		if m.onTypeSelector {
			return m.handleTypeSelector(msg)
		}
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return MsgPopView{} }

		case "tab", "enter":
			if m.isLastVisible() {
				return m.trySave()
			}
			m.fields[m.focused].Blur()
			m.focused = m.nextVisible(m.focused)
			m.fields[m.focused].Focus()
			return m, textinput.Blink

		case "shift+tab":
			prev := m.prevVisible(m.focused)
			if prev == m.focused {
				m.fields[m.focused].Blur()
				m.onTypeSelector = true
				return m, nil
			}
			m.fields[m.focused].Blur()
			m.focused = prev
			m.fields[m.focused].Focus()
			return m, textinput.Blink
		}
	}

	var cmd tea.Cmd
	m.fields[m.focused], cmd = m.fields[m.focused].Update(msg)
	return m, cmd
}

func (m FormModel) handleTypeSelector(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return MsgPopView{} }
	case "space", "right", "l":
		m.typeIdx = (m.typeIdx + 1) % len(typeOptions)
		m.updatePortPlaceholder()
	case "left", "h":
		m.typeIdx = (m.typeIdx - 1 + len(typeOptions)) % len(typeOptions)
		m.updatePortPlaceholder()
	case "tab", "enter":
		m.onTypeSelector = false
		visible := m.visibleFields()
		m.focused = visible[0]
		m.fields[m.focused].Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m FormModel) trySave() (tea.Model, tea.Cmd) {
	host := strings.TrimSpace(m.fields[fieldHost].Value())
	if host == "" {
		m.errMsg = "Host es requerido"
		return m, nil
	}

	connType := m.currentType()
	portStr := strings.TrimSpace(m.fields[fieldPort].Value())
	port := defaultPorts[connType]
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
		Type:         connType,
		Name:         strings.TrimSpace(m.fields[fieldName].Value()),
		Host:         host,
		Port:         port,
		User:         strings.TrimSpace(m.fields[fieldUser].Value()),
		IdentityFile: strings.TrimSpace(m.fields[fieldIdentityFile].Value()),
		Domain:       strings.TrimSpace(m.fields[fieldDomain].Value()),
		Password:     m.fields[fieldPassword].Value(),
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

	typeLine := ""
	for i, t := range typeOptions {
		label := strings.ToUpper(t)
		if i == m.typeIdx {
			if m.onTypeSelector {
				label = styles.ActiveInput.Render("[ " + label + " ]")
			} else {
				label = styles.Tag.Render(label)
			}
		} else {
			label = styles.InactiveInput.Render("  " + label + "  ")
		}
		typeLine += label + " "
	}
	b.WriteString(styles.FieldLabel.Render("Tipo:") + "  " + typeLine + "\n\n")

	visible := m.visibleFields()
	visibleSet := make(map[int]bool, len(visible))
	for _, f := range visible {
		visibleSet[f] = true
	}

	for i := range m.fields {
		if !visibleSet[i] {
			continue
		}
		label := styles.FieldLabel.Render(fieldLabels[i] + ":")
		var input string
		if !m.onTypeSelector && i == m.focused {
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

	b.WriteString(styles.Footer.Render("[Space/←/→]tipo  [Tab/Enter]siguiente  [Shift+Tab]anterior  [Esc]cancelar"))

	return b.String()
}
