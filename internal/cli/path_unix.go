//go:build !windows

package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	blockStart = "# >>> ppm >>>"
	blockEnd   = "# <<< ppm <<<"
)

// shellFiles returns the startup files that should put dir on PATH.
// ~/.profile is always included; the others when they exist or belong to
// the user's shell (zsh is the macOS default).
func shellFiles(home string) (posix []string, fish string) {
	shell := filepath.Base(os.Getenv("SHELL"))
	posix = []string{filepath.Join(home, ".profile")}
	for _, f := range []struct {
		name  string
		force bool
	}{
		{".bashrc", shell == "bash"},
		{".bash_profile", false},
		{".zshrc", shell == "zsh" || runtime.GOOS == "darwin"},
	} {
		path := filepath.Join(home, f.name)
		if _, err := os.Stat(path); err == nil || f.force {
			posix = append(posix, path)
		}
	}
	fishDir := filepath.Join(home, ".config", "fish")
	if _, err := os.Stat(fishDir); err == nil || shell == "fish" {
		fish = filepath.Join(fishDir, "conf.d", "ppm.fish")
	}
	return posix, fish
}

// addToPath appends a marked PATH block to the shell startup files.
func addToPath(dir string) (bool, error) {
	if onPath(dir) {
		return false, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	shown := dir
	if rel, err := filepath.Rel(home, dir); err == nil && !strings.HasPrefix(rel, "..") {
		shown = "$HOME/" + filepath.ToSlash(rel)
	}
	block := "\n" + blockStart + "\nexport PATH=\"" + shown + ":$PATH\"\n" + blockEnd + "\n"

	posix, fish := shellFiles(home)
	changed := false
	for _, f := range posix {
		data, err := os.ReadFile(f)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return changed, err
		}
		if strings.Contains(string(data), blockStart) {
			continue
		}
		if err := appendFile(f, block); err != nil {
			return changed, err
		}
		changed = true
	}
	if fish != "" {
		if _, err := os.Stat(fish); errors.Is(err, fs.ErrNotExist) {
			if err := os.MkdirAll(filepath.Dir(fish), 0o755); err != nil {
				return changed, err
			}
			if err := os.WriteFile(fish, []byte("# Added by ppm\nfish_add_path -g "+dir+"\n"), 0o644); err != nil {
				return changed, err
			}
			changed = true
		}
	}
	return changed, nil
}

// removeFromPath deletes the blocks written by addToPath.
func removeFromPath(string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	posix, fish := shellFiles(home)
	for _, f := range posix {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		text := string(data)
		start := strings.Index(text, "\n"+blockStart)
		if start < 0 {
			continue
		}
		end := strings.Index(text[start:], blockEnd)
		if end < 0 {
			continue
		}
		end += start + len(blockEnd)
		if end < len(text) && text[end] == '\n' {
			end++
		}
		if err := os.WriteFile(f, []byte(text[:start]+text[end:]), 0o644); err != nil {
			return err
		}
	}
	if fish != "" {
		os.Remove(fish)
	}
	return nil
}

func appendFile(path, text string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
