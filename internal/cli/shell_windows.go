//go:build windows

package cli

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// shellCommand runs script with cmd.exe. The command line is set directly
// because cmd.exe does not follow the usual Windows quoting rules.
func shellCommand(script string, args []string) *exec.Cmd {
	line := script
	for _, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\"&|<>^") {
			a = `"` + strings.ReplaceAll(a, `"`, `""`) + `"`
		}
		line += " " + a
	}
	comspec := os.Getenv("COMSPEC")
	if comspec == "" {
		comspec = "cmd.exe"
	}
	cmd := exec.Command(comspec)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `/d /s /c "` + line + `"`}
	return cmd
}
