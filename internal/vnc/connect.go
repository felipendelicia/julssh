package vnc

import (
	"bytes"
	"crypto/des"
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/felipem/julssh/internal/logger"
	"github.com/felipem/julssh/internal/store"
)

const launchTimeout = 2 * time.Second

type MsgVNCLaunched struct {
	Err      error
	Bin      string
	Pkg      string
	RetryCmd tea.Cmd
}

func Connect(c store.Connection) tea.Cmd {
	return func() tea.Msg {
		if _, err := exec.LookPath("vncviewer"); err != nil {
			return MsgVNCLaunched{
				Err:      fmt.Errorf("vncviewer no encontrado"),
				Bin:      "vncviewer",
				Pkg:      "tigervnc-viewer",
				RetryCmd: Connect(c),
			}
		}
		port := 5900
		if c.Port != 0 {
			port = c.Port
		}

		args := []string{fmt.Sprintf("%s::%d", c.Host, port)}

		var tmpPassFile string
		if c.Password != "" {
			f, err := writeTempPassFile(c.Password)
			if err != nil {
				logger.Log("vnc: error writing password file: %v", err)
				return MsgVNCLaunched{Err: fmt.Errorf("vnc: no se pudo crear archivo de contraseña: %w", err)}
			}
			tmpPassFile = f
			args = append(args, "-PasswordFile="+f)
		}

		logger.Log("vnc: exec vncviewer %v", args)
		var stderr bytes.Buffer
		cmd := exec.Command("vncviewer", args...)
		cmd.Stderr = &stderr

		if err := cmd.Start(); err != nil {
			logger.Log("vnc: Start() error: %v", err)
			if tmpPassFile != "" {
				os.Remove(tmpPassFile)
			}
			return MsgVNCLaunched{Err: err}
		}

		done := make(chan error, 1)
		go func() {
			err := cmd.Wait()
			if tmpPassFile != "" {
				os.Remove(tmpPassFile)
			}
			done <- err
		}()

		select {
		case err := <-done:
			errOut := stderr.String()
			logger.Log("vnc: exited quickly — err=%v stderr=%q", err, errOut)
			if err != nil {
				return MsgVNCLaunched{Err: fmt.Errorf("vncviewer: %w", err)}
			}
			return MsgVNCLaunched{}
		case <-time.After(launchTimeout):
			logger.Log("vnc: process still running after 1s — assuming connected")
			return MsgVNCLaunched{}
		}
	}
}

// writeTempPassFile escribe la contraseña en formato VNC (DES obfuscado) en un archivo temporal.
func writeTempPassFile(password string) (string, error) {
	encrypted, err := vncEncryptPassword(password)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "julssh-vnc-*.passwd")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(encrypted); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// vncEncryptPassword genera los 8 bytes del archivo de contraseña VNC.
// VNC usa DES-ECB con una clave fija; los bytes de la clave están bit-invertidos
// respecto al formato estándar.
func vncEncryptPassword(password string) ([]byte, error) {
	fixedKey := []byte{23, 82, 107, 6, 35, 78, 88, 7}

	key := make([]byte, 8)
	for i, b := range fixedKey {
		var x byte
		for j := 0; j < 8; j++ {
			x |= ((b >> uint(j)) & 1) << uint(7-j)
		}
		key[i] = x
	}

	plain := make([]byte, 8)
	copy(plain, []byte(password))

	c, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	encrypted := make([]byte, 8)
	c.Encrypt(encrypted, plain)
	return encrypted, nil
}
