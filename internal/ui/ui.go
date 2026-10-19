// Package ui holds small helpers for terminal output and interactive prompts.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var (
	stdin    = bufio.NewReader(os.Stdin)
	useColor = colorEnabled()
)

func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func paint(code, s string) string {
	if !useColor {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

func Bold(s string) string   { return paint("1", s) }
func Dim(s string) string    { return paint("2", s) }
func Green(s string) string  { return paint("32", s) }
func Yellow(s string) string { return paint("33", s) }
func Red(s string) string    { return paint("31", s) }
func Cyan(s string) string   { return paint("36", s) }

// Step prints the start of a unit of work.
func Step(format string, a ...any) {
	fmt.Println(Cyan("›") + " " + fmt.Sprintf(format, a...))
}

// Success prints a completed unit of work.
func Success(format string, a ...any) {
	fmt.Println(Green("✓") + " " + fmt.Sprintf(format, a...))
}

// Warn prints a non-fatal problem to stderr.
func Warn(format string, a ...any) {
	fmt.Fprintln(os.Stderr, Yellow("!")+" "+fmt.Sprintf(format, a...))
}

// Error prints a fatal problem to stderr.
func Error(format string, a ...any) {
	fmt.Fprintln(os.Stderr, Red("✗")+" "+fmt.Sprintf(format, a...))
}

func readLine() (string, bool) {
	line, err := stdin.ReadString('\n')
	line = strings.TrimSpace(line)
	return line, err == nil || line != ""
}

// Ask prompts for a value. Empty input or EOF returns def.
func Ask(label, def string) string {
	if def != "" {
		fmt.Printf("%s %s: ", label, Dim("("+def+")"))
	} else {
		fmt.Printf("%s: ", label)
	}
	line, ok := readLine()
	if !ok {
		fmt.Println()
	}
	if line == "" {
		return def
	}
	return line
}

// Confirm asks a yes/no question. Empty input or EOF returns def.
func Confirm(label string, def bool) bool {
	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	for {
		fmt.Printf("%s %s: ", label, Dim("("+hint+")"))
		line, ok := readLine()
		if !ok {
			fmt.Println()
			return def
		}
		switch strings.ToLower(line) {
		case "":
			return def
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}
	}
}
