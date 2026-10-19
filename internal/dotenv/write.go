package dotenv

import (
	"errors"
	"io/fs"
	"os"
	"sort"
	"strings"
)

// Set writes vars into the .env file at path, replacing existing keys in
// place and appending new ones. Comments and other lines are kept.
func Set(path string, vars map[string]string) error {
	lines, err := readLines(path)
	if err != nil {
		return err
	}
	done := map[string]bool{}
	for i, line := range lines {
		if k, ok := lineKey(line); ok {
			if v, set := vars[k]; set {
				lines[i] = k + "=" + quote(v)
				done[k] = true
			}
		}
	}
	keys := make([]string, 0, len(vars))
	for k := range vars {
		if !done[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		lines = append(lines, k+"="+quote(vars[k]))
	}
	return writeLines(path, lines)
}

// Unset removes keys from the .env file and returns the ones it found.
func Unset(path string, keys []string) ([]string, error) {
	lines, err := readLines(path)
	if err != nil {
		return nil, err
	}
	drop := map[string]bool{}
	for _, k := range keys {
		drop[k] = true
	}
	var removed []string
	out := lines[:0]
	for _, line := range lines {
		if k, ok := lineKey(line); ok && drop[k] {
			removed = append(removed, k)
			continue
		}
		out = append(out, line)
	}
	return removed, writeLines(path, out)
}

func lineKey(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", false
	}
	k, _, ok := strings.Cut(strings.TrimPrefix(line, "export "), "=")
	return strings.TrimSpace(k), ok
}

// quote wraps values that Parse would otherwise change.
func quote(v string) string {
	if v == "" || !strings.ContainsAny(v, " \t\n#'\"\\") {
		return v
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`).Replace(v) + `"`
}

func readLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	text := strings.TrimRight(string(data), "\n")
	if text == "" {
		return nil, nil
	}
	return strings.Split(text, "\n"), nil
}

func writeLines(path string, lines []string) error {
	data := strings.Join(lines, "\n")
	if data != "" {
		data += "\n"
	}
	return os.WriteFile(path, []byte(data), 0o600)
}
