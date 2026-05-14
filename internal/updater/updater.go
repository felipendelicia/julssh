package updater

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var apiURL = "https://api.github.com/repos/felipendelicia/julssh/releases/latest"

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Check looks up the latest GitHub release. If newer than currentVersion,
// it prompts the user, downloads, replaces the binary, and relaunches.
// Returns immediately if currentVersion == "dev".
func Check(currentVersion string) {
	if currentVersion == "dev" {
		return
	}

	rel, err := fetchLatest(apiURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "No se pudo verificar actualizaciones: %v\n", err)
		return
	}

	newer, err := isNewer(rel.TagName, currentVersion)
	if err != nil || !newer {
		return
	}

	url := findAssetURL(rel, assetFilename())
	if url == "" {
		return
	}

	fmt.Printf("Nueva versión %s disponible. ¿Actualizar? [s/N] ", rel.TagName)
	var answer string
	fmt.Scanln(&answer)
	if strings.ToLower(strings.TrimSpace(answer)) != "s" {
		return
	}

	fmt.Println("Descargando actualización...")
	if err := doUpdate(url); err != nil {
		fmt.Fprintf(os.Stderr, "Error al actualizar: %v\n", err)
		return
	}

	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error al relanzar: %v\n", err)
		return
	}
	if err := syscall.Exec(execPath, os.Args, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "Error al relanzar: %v\n", err)
	}
}

func fetchLatest(url string) (*release, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func parseSemver(v string) ([3]int, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return [3]int{}, fmt.Errorf("invalid semver: %q", v)
	}
	var nums [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, fmt.Errorf("invalid semver part %q: %w", p, err)
		}
		nums[i] = n
	}
	return nums, nil
}

func isNewer(latest, current string) (bool, error) {
	l, err := parseSemver(latest)
	if err != nil {
		return false, err
	}
	c, err := parseSemver(current)
	if err != nil {
		return false, err
	}
	for i := range l {
		if l[i] > c[i] {
			return true, nil
		}
		if l[i] < c[i] {
			return false, nil
		}
	}
	return false, nil
}

func assetFilename() string {
	return fmt.Sprintf("julssh_linux_%s.tar.gz", runtime.GOARCH)
}

func findAssetURL(rel *release, name string) string {
	for _, a := range rel.Assets {
		if a.Name == name {
			return a.BrowserDownloadURL
		}
	}
	return ""
}

func doUpdate(url string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("obtener ruta ejecutable: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("descarga: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("descarga HTTP %d", resp.StatusCode)
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var binaryData []byte
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		if hdr.Name == "julssh" || strings.HasSuffix(hdr.Name, "/julssh") {
			binaryData, err = io.ReadAll(tr)
			if err != nil {
				return fmt.Errorf("leer binario: %w", err)
			}
			break
		}
	}
	if binaryData == nil {
		return fmt.Errorf("binario julssh no encontrado en el archivo")
	}

	dir := filepath.Dir(execPath)
	tmp, err := os.CreateTemp(dir, "julssh-update-*")
	if err != nil {
		return fmt.Errorf("crear temp: %w", err)
	}
	tmpName := tmp.Name()

	if err := tmp.Chmod(0755); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("chmod: %w", err)
	}
	if _, err := tmp.Write(binaryData); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("escribir temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("cerrar temp: %w", err)
	}

	if err := os.Rename(tmpName, execPath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("reemplazar ejecutable: %w", err)
	}

	return nil
}
