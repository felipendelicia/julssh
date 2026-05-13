package model

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/styles"
)

type msgInstallDone struct{ err error }

type InstallView struct {
	bin      string
	pkg      string
	retryCmd tea.Cmd
	hasApt   bool
	errMsg   string
	waiting  bool
}

func NewInstallView(bin, pkg string, retryCmd tea.Cmd) InstallView {
	_, err := exec.LookPath("apt-get")
	return InstallView{
		bin:      bin,
		pkg:      pkg,
		retryCmd: retryCmd,
		hasApt:   err == nil,
	}
}

func (m InstallView) Init() tea.Cmd { return nil }

func (m InstallView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgInstallDone:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.waiting = false
			return m, nil
		}
		return m, tea.Batch(
			func() tea.Msg { return MsgPopView{} },
			m.retryCmd,
		)

	case tea.KeyMsg:
		if m.waiting {
			return m, nil
		}
		switch msg.String() {
		case "s", "y":
			if m.hasApt {
				m.waiting = true
				cmd := exec.Command("sudo", "apt-get", "install", "-y", m.pkg)
				return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
					return msgInstallDone{err: err}
				})
			}
		case "esc", "n", "q":
			return m, func() tea.Msg { return MsgPopView{} }
		}
	}
	return m, nil
}

func (m InstallView) View() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Cliente no encontrado"))
	b.WriteString("\n\n")
	b.WriteString(styles.ErrText.Render("! " + m.bin + " no está instalado"))
	b.WriteString("\n\n")

	if m.hasApt {
		b.WriteString(styles.MutedText.Render("Comando de instalación:"))
		b.WriteString("\n")
		b.WriteString("  sudo apt-get install -y " + m.pkg)
		b.WriteString("\n\n")
		if m.waiting {
			b.WriteString(styles.MutedText.Render("Instalando..."))
		} else if m.errMsg != "" {
			b.WriteString(styles.ErrText.Render("! instalación falló: " + m.errMsg))
			b.WriteString("\n")
			b.WriteString(styles.Footer.Render("[Esc]volver"))
		} else {
			b.WriteString(styles.Footer.Render("[s]instalar ahora  [n/Esc]cancelar"))
		}
	} else {
		b.WriteString(styles.MutedText.Render("Instalá manualmente según tu distro:"))
		b.WriteString("\n")
		b.WriteString("  sudo apt-get install -y " + m.pkg + "   # Debian/Ubuntu")
		b.WriteString("\n")
		b.WriteString("  sudo pacman -S " + m.pkg + "")
		b.WriteString("   # Arch\n")
		b.WriteString("  sudo dnf install -y " + m.pkg + "")
		b.WriteString("   # Fedora\n")
		b.WriteString("\n")
		b.WriteString(styles.Footer.Render("[Esc]volver"))
	}

	return b.String()
}
