// Package toolchain manages the Go compiler for building standalone exploits.
// Auto-downloads Go if not available on the system.
package toolchain

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Chocapikk/pik/pkg/log"
)

const goVersion = "1.24.4"

// --- Types ---

// BuildOpts configures a compilation.
type BuildOpts struct {
	Dir    string // source directory (contains main.go + go.mod)
	Output string // output binary path
	OS     string // GOOS (empty = native)
	Arch   string // GOARCH (empty = native)
}

// --- Public API ---

// Build compiles a Go program. Downloads the toolchain if needed.
func Build(opts BuildOpts) error {
	goBin, err := resolve()
	if err != nil {
		return fmt.Errorf("toolchain: %w", err)
	}

	buildEnv := env(opts)

	// Tidy deps first
	tidy := exec.Command(goBin, "mod", "tidy")
	tidy.Dir = opts.Dir
	tidy.Env = buildEnv
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	if err := tidy.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	// Build
	build := exec.Command(goBin, "build", "-o", opts.Output, "-trimpath", "-ldflags=-s -w", ".")
	build.Dir = opts.Dir
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	build.Env = buildEnv

	return build.Run()
}

// --- Resolution ---

func resolve() (string, error) {
	if path, err := exec.LookPath("go"); err == nil {
		return path, nil
	}

	managed := filepath.Join(pikDir(), "go", "bin", "go")
	if _, err := os.Stat(managed); err == nil {
		return managed, nil
	}

	if err := download(); err != nil {
		return "", err
	}
	return managed, nil
}

// --- Download ---

func download() error {
	url := fmt.Sprintf("https://go.dev/dl/go%s.%s-%s.tar.gz", goVersion, runtime.GOOS, runtime.GOARCH)
	dest := pikDir()

	log.Status("Go not found, downloading go%s...", goVersion)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	if err := extractTarGz(resp.Body, dest); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	goBin := filepath.Join(dest, "go", "bin", "go")
	if _, err := os.Stat(goBin); err != nil {
		return fmt.Errorf("extraction failed: %s not found", goBin)
	}

	log.Success("Installed go%s to %s", goVersion, filepath.Join(dest, "go"))
	return nil
}

// --- Build environment ---

func env(opts BuildOpts) []string {
	e := os.Environ()
	e = set(e, "CGO_ENABLED", "0")

	managed := filepath.Join(pikDir(), "go")
	if _, err := os.Stat(managed); err == nil {
		e = set(e, "GOROOT", managed)
		e = set(e, "PATH", filepath.Join(managed, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	if opts.OS != "" {
		e = set(e, "GOOS", opts.OS)
	}
	if opts.Arch != "" {
		e = set(e, "GOARCH", opts.Arch)
	}
	return e
}

func set(env []string, key, val string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + val
			return env
		}
	}
	return append(env, prefix+val)
}

// --- Tar extraction ---

func extractTarGz(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, hdr.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(f, tr)
			f.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// --- Scaffold ---

// ScaffoldOpts configures standalone scaffold generation.
type ScaffoldOpts struct {
	ImportPath string // Go import path of the module package
	ModRoot    string // Local repo root (empty = fetch from proxy)
	Version    string // Module version tag (used when ModRoot is empty)
}

// Scaffold creates a temp directory with main.go and go.mod for a standalone build.
// Returns the dir path and a cleanup function.
func Scaffold(opts ScaffoldOpts) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "pik-build-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(tmpDir) }

	if err := RenderToFile(filepath.Join(tmpDir, "main.go"), "standalone_main.go.tpl", map[string]string{
		"ImportPath": opts.ImportPath,
	}); err != nil {
		cleanup()
		return "", nil, err
	}

	version := opts.Version
	if version == "" {
		version = "v0.0.0"
	}

	if err := RenderToFile(filepath.Join(tmpDir, "go.mod"), "standalone_gomod.tpl", map[string]string{
		"ModRoot": opts.ModRoot,
		"Version": version,
	}); err != nil {
		cleanup()
		return "", nil, err
	}

	return tmpDir, cleanup, nil
}

// --- Paths ---

func pikDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".pik")
	}
	return filepath.Join(".", ".pik")
}
