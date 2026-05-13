package model

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/store"
)

type ListModel struct{ store *store.Store }

func NewList(s *store.Store) ListModel                        { return ListModel{store: s} }
func (m ListModel) Init() tea.Cmd                             { return nil }
func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)  { return m, nil }
func (m ListModel) View() string                              { return "list stub" }
