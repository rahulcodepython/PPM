package ui

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

// Width is the number of terminal columns s takes, ignoring color codes.
func Width(s string) int {
	return utf8.RuneCountInString(ansiRe.ReplaceAllString(s, ""))
}

// Pad right-pads s with spaces to width columns, ignoring color codes.
func Pad(s string, width int) string {
	if n := width - Width(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

// Table writes rows as a bordered table whose columns fit their widest
// cell. Cells may contain color codes.
func Table(w io.Writer, headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = Width(h)
	}
	for _, r := range rows {
		for i, c := range r {
			if i < len(widths) && Width(c) > widths[i] {
				widths[i] = Width(c)
			}
		}
	}

	border := func(left, mid, right string) {
		parts := make([]string, len(widths))
		for i, n := range widths {
			parts[i] = strings.Repeat("─", n+2)
		}
		fmt.Fprintln(w, Dim(left+strings.Join(parts, mid)+right))
	}
	line := func(cells []string, style func(string) string) {
		var b strings.Builder
		b.WriteString(Dim("│"))
		for i := range widths {
			c := ""
			if i < len(cells) {
				c = cells[i]
			}
			b.WriteString(" " + Pad(style(c), widths[i]) + " " + Dim("│"))
		}
		fmt.Fprintln(w, b.String())
	}

	border("┌", "┬", "┐")
	line(headers, Bold)
	border("├", "┼", "┤")
	for _, r := range rows {
		line(r, func(s string) string { return s })
	}
	border("└", "┴", "┘")
}
