package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestTableAlignsColoredCells(t *testing.T) {
	useColor = true
	defer func() { useColor = colorEnabled() }()

	var buf bytes.Buffer
	Table(&buf, []string{"PACKAGE", "TYPE"}, [][]string{
		{"uvicorn[standard]", Green("prod")},
		{"six", Red("missing")},
	})
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 6 {
		t.Fatalf("got %d lines:\n%s", len(lines), buf.String())
	}
	for _, l := range lines {
		if w := Width(l); w != Width(lines[0]) {
			t.Errorf("line width %d != %d: %q", w, Width(lines[0]), l)
		}
	}
}
