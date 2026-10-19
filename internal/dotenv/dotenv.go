// Package dotenv reads KEY=VALUE files such as .env.
package dotenv

import (
	"bufio"
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
)

// Load reads a .env file. A missing file returns an empty map.
func Load(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse supports comments, blank lines, an optional "export " prefix,
// single quotes (literal), double quotes (with \n, \t, \", \\ escapes)
// and trailing " # comments" on unquoted values.
func Parse(r io.Reader) (map[string]string, error) {
	vars := map[string]string{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		vars[key] = parseValue(strings.TrimSpace(val))
	}
	return vars, sc.Err()
}

func parseValue(v string) string {
	if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
		return v[1 : len(v)-1]
	}
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return strings.NewReplacer(`\n`, "\n", `\t`, "\t", `\"`, `"`, `\\`, `\`).Replace(v[1 : len(v)-1])
	}
	if i := strings.Index(v, " #"); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	return v
}
