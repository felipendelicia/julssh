package model

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/store"
	"github.com/felipem/julssh/internal/styles"
)

type ListModel struct {
	store       *store.Store
	cursor      int
	filterMode  bool
	filterQuery string
	width       int
	height      int
}

func NewList(s *store.Store) ListModel {
	return ListModel{store: s}
}

func (m ListModel) Init() tea.Cmd { return nil }

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	conns := m.store.Filter(m.filterQuery)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
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
	case "/":
		m.filterMode = true
		m.filterQuery = ""
		m.cursor = 0
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
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

func (m ListModel) View() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("julssh ♥ Julieta"))
	b.WriteString("\n\n")

	conns := m.store.Filter(m.filterQuery)

	if len(conns) == 0 {
		if m.filterQuery != "" {
			b.WriteString(styles.MutedText.Render(fmt.Sprintf("  Sin resultados para %q", m.filterQuery)))
		} else {
			b.WriteString(styles.MutedText.Render("  Sin conexiones. Presioná [n] para agregar la primera."))
		}
		b.WriteString("\n")
	}

	for i, c := range conns {
		row := formatRow(c, m.width)
		if i == m.cursor {
			b.WriteString(styles.SelectedRow.Render("> " + row))
		} else {
			b.WriteString(styles.NormalRow.Render("  " + row))
		}
		b.WriteString("\n")
	}

	footer := "[n]ueva  [/]filtrar  [q]salir"
	if m.filterMode {
		footer = fmt.Sprintf("Filtro: %s_  [Esc]cancelar", m.filterQuery)
	} else if m.filterQuery != "" {
		footer = fmt.Sprintf("Filtro: %q activo  [/]cambiar  [q]salir", m.filterQuery)
	}
	b.WriteString(styles.Footer.Render(footer))

	return b.String()
}

func formatRow(c store.Connection, width int) string {
	host := c.Host
	if c.Port != 0 && c.Port != 22 {
		host = fmt.Sprintf("%s:%d", c.Host, c.Port)
	}
	if c.User != "" {
		host = c.User + "@" + host
	}

	tags := ""
	for _, t := range c.Tags {
		tags += styles.Tag.Render(t) + " "
	}

	name := fmt.Sprintf("%-20s", c.Name)
	addr := fmt.Sprintf("%-30s", host)
	return name + "  " + addr + "  " + tags
}
