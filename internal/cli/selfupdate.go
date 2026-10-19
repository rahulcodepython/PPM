package cli

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rahulcodepython/PPM/internal/ui"
)

const repo = "rahulcodepython/PPM"

var httpClient = &http.Client{Timeout: 2 * time.Minute}

func newSelfUpdateCmd() *cobra.Command {
	var check, force bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update ppm to the latest release",
		Long: `Download the latest ppm release from GitHub, verify its SHA-256
checksum and replace the running binary. If you installed ppm with a
package manager (apt, dnf, apk), update it with that instead.`,
		Example: `  ppm self update
  ppm self update --check`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			latest, err := latestTag()
			if err != nil {
				return err
			}
			current := resolvedVersion()
			if strings.TrimPrefix(latest, "v") == current && !force {
				ui.Success("ppm %s is the latest version", current)
				return nil
			}
			if check {
				fmt.Printf("Update available: %s → %s. Run `ppm self update`.\n", current, latest)
				return nil
			}
			if current == "dev" && !force {
				return fmt.Errorf("this is a development build; use --force to replace it with %s", latest)
			}
			return selfUpdate(latest)
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "only report whether an update exists")
	cmd.Flags().BoolVar(&force, "force", false, "reinstall even if already up to date")
	return cmd
}

func latestTag() (string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+repo+"/releases/latest", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("checking for updates: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("no ppm release published yet at https://github.com/%s/releases", repo)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checking for updates: GitHub returned %s", resp.Status)
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil || rel.TagName == "" {
		return "", fmt.Errorf("checking for updates: unexpected response from GitHub")
	}
	return rel.TagName, nil
}

func download(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func selfUpdate(tag string) error {
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	archive := fmt.Sprintf("ppm_%s_%s.%s", runtime.GOOS, runtime.GOARCH, ext)
	base := "https://github.com/" + repo + "/releases/download/" + tag + "/"

	ui.Step("Downloading %s %s", archive, tag)
	data, err := download(base + archive)
	if err != nil {
		return err
	}
	sums, err := download(base + "checksums.txt")
	if err != nil {
		return err
	}
	if err := verifyChecksum(data, sums, archive); err != nil {
		return err
	}
	bin, err := extractBinary(data, ext)
	if err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}
	if err := replaceExecutable(exe, bin); err != nil {
		return fmt.Errorf("replacing %s: %w (try again with sudo, or reinstall with the install script)", exe, err)
	}
	ui.Success("Updated ppm to %s", tag)
	return nil
}

func verifyChecksum(data, sums []byte, name string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == name {
			if fields[0] != got {
				return fmt.Errorf("checksum mismatch for %s", name)
			}
			return nil
		}
	}
	return fmt.Errorf("%s not listed in checksums.txt", name)
}

func extractBinary(data []byte, ext string) ([]byte, error) {
	if ext == "zip" {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if filepath.Base(f.Name) == "ppm.exe" {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(rc)
			}
		}
		return nil, errors.New("ppm.exe not found in archive")
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, errors.New("ppm not found in archive")
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag == tar.TypeReg && filepath.Base(h.Name) == "ppm" {
			return io.ReadAll(tr)
		}
	}
}

// replaceExecutable swaps in the new binary. The running file is renamed
// first, which Windows allows even while it is executing.
func replaceExecutable(exe string, bin []byte) error {
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".ppm-new-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	old := exe + ".old"
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), exe); err != nil {
		os.Rename(old, exe) // roll back
		return err
	}
	if runtime.GOOS != "windows" {
		os.Remove(old) // Windows keeps it until the next update
	}
	return nil
}
