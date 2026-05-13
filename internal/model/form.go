package model

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/store"
)

type FormModel struct {
	store *store.Store
}

func NewForm(s *store.Store, conn *store.Connection) FormModel {
	return FormModel{store: s}
}

func (m FormModel) Init() tea.Cmd                            { return nil }
func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m FormModel) View() string                             { return "form stub" }
