// Package python finds interpreters and drives virtual environments and pip.
package python

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const windows = runtime.GOOS == "windows"

var versionRe = regexp.MustCompile(`^\d+(\.\d+){0,2}$`)

// Interpreter is a Python command, e.g. {"py", ["-3.12"]} or {"python3", nil}.
type Interpreter struct {
	Cmd     string
	Args    []string
	Version string
	Exe     string // resolved sys.executable
}

func (i *Interpreter) String() string {
	return strings.TrimSpace(i.Cmd + " " + strings.Join(i.Args, " "))
}

// Command builds an exec.Cmd for the interpreter.
func (i *Interpreter) Command(args ...string) *exec.Cmd {
	return exec.Command(i.Cmd, append(append([]string{}, i.Args...), args...)...)
}

// Find locates a Python 3 interpreter. hint may be empty, a version
// ("3.12"), a command name ("python3.11") or a path.
func Find(hint string) (*Interpreter, error) {
	var candidates [][]string
	want := ""
	switch {
	case hint == "":
		if windows {
			candidates = [][]string{{"py", "-3"}, {"python"}, {"python3"}}
		} else {
			candidates = [][]string{{"python3"}, {"python"}}
		}
	case versionRe.MatchString(hint):
		want = hint
		if windows {
			candidates = [][]string{{"py", "-" + hint}, {"python" + hint}, {"python"}, {"python3"}}
		} else {
			candidates = [][]string{{"python" + hint}, {"python3"}, {"python"}}
		}
	default:
		candidates = [][]string{{hint}}
	}

	for _, c := range candidates {
		in := &Interpreter{Cmd: c[0], Args: c[1:]}
		v, exe, err := in.probe()
		if err != nil || !strings.HasPrefix(v, "3.") {
			continue
		}
		if want != "" && v != want && !strings.HasPrefix(v, want+".") {
			continue
		}
		in.Version, in.Exe = v, exe
		return in, nil
	}
	if hint == "" {
		return nil, fmt.Errorf("python 3 not found on PATH; install it from https://www.python.org/downloads/")
	}
	return nil, fmt.Errorf("python %q not found on PATH", hint)
}

func (i *Interpreter) probe() (version, exe string, err error) {
	if _, err := exec.LookPath(i.Cmd); err != nil {
		return "", "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	args := append(append([]string{}, i.Args...), "-c", "import sys;print('%d.%d.%d' % sys.version_info[:3]);print(sys.executable)")
	out, err := exec.CommandContext(ctx, i.Cmd, args...).Output()
	if err != nil {
		return "", "", err
	}
	version, exe, _ = strings.Cut(strings.TrimSpace(string(out)), "\n")
	return strings.TrimSpace(version), strings.TrimSpace(exe), nil
}
