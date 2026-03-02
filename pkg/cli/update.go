package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"aead.dev/minisign"
	"github.com/spf13/cobra"

	"github.com/Chocapikk/pik/pkg/output"
)

const (
	githubRepo       = "Chocapikk/pik"
	signingPublicKey = "RWTGq/JU6UbpIDjfAHsW9l6SYetGGY+O5bjYKpo4tjzTRnaCVTVnjvSA"
)

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update pik to the latest release",
		Run: func(_ *cobra.Command, _ []string) {
			if err := autoUpdate(Version); err != nil {
				output.Error("%v", err)
				os.Exit(1)
			}
		},
	}
}

func autoUpdate(currentVersion string) error {
	output.Status("Checking for updates...")

	latest, err := latestRelease()
	if err != nil {
		return err
	}

	if currentVersion == latest {
		output.Success("Already up-to-date (%s)", currentVersion)
		return nil
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	binaryName := fmt.Sprintf("pik_%s_%s_%s%s", latest, runtime.GOOS, runtime.GOARCH, ext)
	baseURL := fmt.Sprintf("https://github.com/%s/releases/download/%s", githubRepo, latest)

	// Strip leading v for checksums filename (goreleaser uses version without v prefix).
	version := strings.TrimPrefix(latest, "v")
	checksumsName := fmt.Sprintf("pik_%s_checksums.txt", version)

	output.Status("Downloading %s", latest)

	binary, err := httpGet(baseURL + "/" + binaryName)
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}

	checksums, err := httpGet(baseURL + "/" + checksumsName)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}

	signature, err := httpGet(baseURL + "/" + checksumsName + ".minisig")
	if err != nil {
		return fmt.Errorf("download signature: %w", err)
	}

	output.Status("Verifying signature...")

	var pubKey minisign.PublicKey
	if err := pubKey.UnmarshalText([]byte(signingPublicKey)); err != nil {
		return fmt.Errorf("invalid embedded public key: %w", err)
	}

	if !minisign.Verify(pubKey, checksums, signature) {
		return fmt.Errorf("signature verification failed - binary may have been tampered with")
	}

	output.Status("Verifying checksum...")

	hash := sha256.Sum256(binary)
	got := hex.EncodeToString(hash[:])

	expected, err := findChecksum(checksums, binaryName)
	if err != nil {
		return err
	}

	if got != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, got)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	tmp := exe + ".tmp"
	if err := os.WriteFile(tmp, binary, 0o755); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	if runtime.GOOS == "windows" {
		os.Remove(exe)
	}

	if err := os.Rename(tmp, exe); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("replace failed: %w", err)
	}

	output.Success("Updated to %s (signature verified). Restart pik to use the new version.", latest)
	return nil
}

func httpGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}

func findChecksum(checksums []byte, filename string) (string, error) {
	for _, line := range strings.Split(string(checksums), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == filename {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("checksum not found for %s", filename)
}

func latestRelease() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to check releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}

	var result struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("invalid response: %w", err)
	}
	if result.TagName == "" {
		return "", fmt.Errorf("no release found")
	}
	return result.TagName, nil
}
