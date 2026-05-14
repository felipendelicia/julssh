package model

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/store"
	"github.com/felipem/julssh/internal/styles"
)

type inputMode int

const (
	inputNone   inputMode = iota
	inputExport
	inputImport
)

type connGroup struct {
	label string
	conns []store.Connection
}

type ListModel struct {
	store       *store.Store
	cursor      int
	filterMode  bool
	filterQuery string
	mode        inputMode
	pathInput   textinput.Model
	width       int
	height      int
}

func NewList(s *store.Store) ListModel {
	pi := textinput.New()
	pi.CharLimit = 512
	pi.Placeholder = "ruta/al/archivo.json"
	return ListModel{store: s, pathInput: pi}
}

func (m ListModel) Init() tea.Cmd { return nil }

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.mode != inputNone {
			return m.handlePathInput(msg)
		}
		conns := m.displayConns()
		if m.filterMode {
			return m.handleFilterKey(msg, conns)
		}
		return m.handleKey(msg, conns)
	}
	return m, nil
}

func (m ListModel) handleKey(msg tea.KeyMsg, conns []store.Connection) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(conns)-1 {
			m.cursor++
		}
	case "enter":
		if len(conns) > 0 && m.cursor < len(conns) {
			detail := NewDetail(conns[m.cursor], m.store)
			return m, func() tea.Msg { return MsgPushView{View: detail} }
		}
	case "n":
		form := NewForm(m.store, nil)
		return m, func() tea.Msg { return MsgPushView{View: form} }
	case "e":
		if len(conns) > 0 && m.cursor < len(conns) {
			conn := conns[m.cursor]
			form := NewForm(m.store, &conn)
			return m, func() tea.Msg { return MsgPushView{View: form} }
		}
	case "c":
		if len(conns) > 0 && m.cursor < len(conns) {
			conn := conns[m.cursor]
			_ = m.store.RecordConnect(conn.ID)
			return m, connectCmd(conn)
		}
	case "/":
		m.filterMode = true
		m.filterQuery = ""
		m.cursor = 0
	case "X":
		m.mode = inputExport
		m.pathInput.SetValue("")
		m.pathInput.Focus()
		return m, textinput.Blink
	case "I":
		m.mode = inputImport
		m.pathInput.SetValue("")
		m.pathInput.Focus()
		return m, textinput.Blink
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m ListModel) handlePathInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = inputNone
		m.pathInput.Blur()
		return m, nil
	case "enter":
		path := strings.TrimSpace(m.pathInput.Value())
		currentMode := m.mode
		m.mode = inputNone
		m.pathInput.Blur()
		if path == "" {
			return m, nil
		}
		switch currentMode {
		case inputExport:
			count := len(m.store.Connections)
			return m, func() tea.Msg {
				if err := m.store.ExportAll(path); err != nil {
					return MsgError{Err: err}
				}
				return MsgExportDone{Path: path, Count: count}
			}
		case inputImport:
			return m, func() tea.Msg {
				added, err := m.store.ImportMerge(path)
				if err != nil {
					return MsgError{Err: err}
				}
				return MsgImportDone{Added: added}
			}
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

func (m ListModel) handleFilterKey(msg tea.KeyMsg, conns []store.Connection) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filterMode = false
		m.filterQuery = ""
		m.cursor = 0
	case "enter":
		m.filterMode = false
	case "backspace":
		if len(m.filterQuery) > 0 {
			m.filterQuery = m.filterQuery[:len(m.filterQuery)-1]
			m.cursor = 0
		}
	default:
		if len(msg.String()) == 1 {
			m.filterQuery += msg.String()
			m.cursor = 0
		}
	}
	return m, nil
}

func (m ListModel) displayConns() []store.Connection {
	conns := m.store.Filter(m.filterQuery)
	if m.filterQuery != "" {
		return conns
	}
	groups := groupConnections(conns)
	var result []store.Connection
	for _, g := range groups {
		result = append(result, g.conns...)
	}
	return result
}

func groupConnections(conns []store.Connection) []connGroup {
	groups := map[string]*connGroup{}
	var order []string
	for _, c := range conns {
		label := "sin tag"
		if len(c.Tags) > 0 {
			label = c.Tags[0]
		}
		if _, ok := groups[label]; !ok {
			groups[label] = &connGroup{label: label}
			order = append(order, label)
		}
		groups[label].conns = append(groups[label].conns, c)
	}
	sort.Slice(order, func(i, j int) bool {
		if order[i] == "sin tag" {
			return false
		}
		if order[j] == "sin tag" {
			return true
		}
		return order[i] < order[j]
	})
	result := make([]connGroup, len(order))
	for i, label := range order {
		result[i] = *groups[label]
		sort.Slice(result[i].conns, func(a, b int) bool {
			return result[i].conns[a].Name < result[i].conns[b].Name
		})
	}
	return result
}

func timeAgo(t *time.Time) string {
	if t == nil {
		return "nunca"
	}
	d := time.Since(*t)
	switch {
	case d < time.Minute:
		return "ahora"
	case d < time.Hour:
		return fmt.Sprintf("hace %dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("hace %dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("hace %dd", int(d.Hours()/24))
	default:
		return t.Format("02/01/06")
	}
}

func (m ListModel) View() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("julssh :: jd"))
	b.WriteString("\n\n")

	if len(m.store.Connections) == 0 && m.filterQuery == "" {
		b.WriteString(styles.MutedText.Render("  Sin conexiones. Presioná [n] para agregar la primera."))
		b.WriteString("\n")
	} else if m.filterQuery != "" {
		conns := m.store.Filter(m.filterQuery)
		if len(conns) == 0 {
			b.WriteString(styles.MutedText.Render(fmt.Sprintf("  Sin resultados para %q", m.filterQuery)))
			b.WriteString("\n")
		} else {
			for i, c := range conns {
				row := formatRow(c, m.width)
				if i == m.cursor {
					b.WriteString(styles.SelectedRow.Render("> " + row))
				} else {
					b.WriteString(styles.NormalRow.Render("  " + row))
				}
				b.WriteString("\n")
			}
		}
	} else {
		groups := groupConnections(m.store.Filter(""))
		idx := 0
		for gi, g := range groups {
			b.WriteString(styles.MutedText.Render("[" + g.label + "]"))
			b.WriteString("\n")
			for _, c := range g.conns {
				row := formatRow(c, m.width)
				if idx == m.cursor {
					b.WriteString(styles.SelectedRow.Render("> " + row))
				} else {
					b.WriteString(styles.NormalRow.Render("  " + row))
				}
				b.WriteString("\n")
				idx++
			}
			if gi < len(groups)-1 {
				b.WriteString("\n")
			}
		}
	}

	switch m.mode {
	case inputExport:
		b.WriteString(styles.Footer.Render("Export a: " + m.pathInput.View() + "  [Enter]confirmar  [Esc]cancelar"))
	case inputImport:
		b.WriteString(styles.Footer.Render("Import desde: " + m.pathInput.View() + "  [Enter]confirmar  [Esc]cancelar"))
	default:
		footer := "[n]ueva  [e]editar  [c]conectar  [/]filtrar  [X]exportar  [I]importar  [q]salir"
		if m.filterMode {
			footer = fmt.Sprintf("Filtro: %s_  [Esc]cancelar", m.filterQuery)
		} else if m.filterQuery != "" {
			footer = fmt.Sprintf("Filtro: %q activo  [/]cambiar  [q]salir", m.filterQuery)
		}
		b.WriteString(styles.Footer.Render(footer))
	}

	return b.String()
}

func formatRow(c store.Connection, width int) string {
	host := c.Host
	defaultPort := map[string]int{"ssh": 22, "rdp": 3389, "vnc": 5900}
	if c.Port != 0 && c.Port != defaultPort[c.Type] {
		host = fmt.Sprintf("%s:%d", c.Host, c.Port)
	}
	if c.User != "" {
		host = c.User + "@" + host
	}

	typeBadge := styles.Tag.Render(strings.ToUpper(c.Type))
	tags := ""
	for _, t := range c.Tags {
		tags += styles.Tag.Render(t) + " "
	}
	name := fmt.Sprintf("%-20s", c.Name)
	addr := fmt.Sprintf("%-30s", host)
	ago := styles.MutedText.Render(timeAgo(c.LastConnectedAt))
	return typeBadge + " " + name + "  " + addr + "  " + tags + ago
}
