package model

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/store"
)

type DetailModel struct {
	conn  store.Connection
	store *store.Store
}

func NewDetail(conn store.Connection, s *store.Store) DetailModel {
	return DetailModel{conn: conn, store: s}
}

func (m DetailModel) Init() tea.Cmd                            { return nil }
func (m DetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m DetailModel) View() string                             { return "detail stub" }
