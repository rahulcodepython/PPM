//go:build !windows

package cli

import "os/exec"

// shellCommand runs script with sh. Extra args are passed as "$@", so
// they reach the script unchanged, spaces and quotes included.
func shellCommand(script string, args []string) *exec.Cmd {
	if len(args) > 0 {
		script += ` "$@"`
	}
	return exec.Command("/bin/sh", append([]string{"-c", script, "ppm"}, args...)...)
}
