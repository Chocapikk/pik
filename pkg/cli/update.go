package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/Chocapikk/pik/pkg/output"
)

const githubRepo = "Chocapikk/pik"

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
	url := fmt.Sprintf(
		"https://github.com/%s/releases/download/%s/pik_%s_%s_%s%s",
		githubRepo, latest, latest, runtime.GOOS, runtime.GOARCH, ext,
	)

	output.Status("Downloading %s", latest)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("release not found: %s", url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	tmp := exe + ".tmp"
	if err := os.WriteFile(tmp, body, 0o755); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	if runtime.GOOS == "windows" {
		os.Remove(exe)
	}

	if err := os.Rename(tmp, exe); err != nil {
		return fmt.Errorf("replace failed: %w", err)
	}

	output.Success("Updated to %s. Restart pik to use the new version.", latest)
	return nil
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
