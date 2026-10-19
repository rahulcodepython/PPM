package python

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rahulcodepython/PPM/internal/project"
)

// Venv is a virtual environment directory plus the installer used to
// manage its packages.
type Venv struct {
	Dir string
	uv  string // path to uv; empty means pip
}

// NewVenv returns the venv at dir. installer is "pip", "uv" or ""/"auto"
// (uv when it is on PATH, otherwise pip). PPM_INSTALLER overrides it.
func NewVenv(dir, installer string) (Venv, error) {
	if env := os.Getenv("PPM_INSTALLER"); env != "" {
		installer = env
	}
	v := Venv{Dir: dir}
	switch installer {
	case "", "auto":
		v.uv, _ = exec.LookPath("uv")
	case "uv":
		path, err := exec.LookPath("uv")
		if err != nil {
			return v, fmt.Errorf("installer is uv but uv is not on PATH; install it from https://docs.astral.sh/uv/ or set installer: pip")
		}
		v.uv = path
	case "pip":
	default:
		return v, fmt.Errorf("unknown installer %q (use pip, uv or auto)", installer)
	}
	return v, nil
}

// Installer names the tool managing packages: "uv" or "pip".
func (v Venv) Installer() string {
	if v.uv != "" {
		return "uv"
	}
	return "pip"
}

// BinDir is where the venv keeps python, pip and console scripts.
func (v Venv) BinDir() string {
	if windows {
		return filepath.Join(v.Dir, "Scripts")
	}
	return filepath.Join(v.Dir, "bin")
}

// Python is the venv's interpreter.
func (v Venv) Python() string {
	if windows {
		return filepath.Join(v.BinDir(), "python.exe")
	}
	return filepath.Join(v.BinDir(), "python")
}

// Exists reports whether the venv has a usable interpreter.
func (v Venv) Exists() bool {
	_, err := os.Stat(v.Python())
	return err == nil
}

// Version returns the venv's Python version.
func (v Venv) Version() (string, error) {
	ver, _, err := (&Interpreter{Cmd: v.Python()}).probe()
	return ver, err
}

// Create builds the venv with the given interpreter. With pip it also
// upgrades pip; with uv it seeds pip so `ppm exec pip` still works.
func (v Venv) Create(in *Interpreter) error {
	var cmd *exec.Cmd
	if v.uv != "" {
		cmd = exec.Command(v.uv, "venv", "--quiet", "--seed", "--allow-existing", "--python", in.Exe, v.Dir)
	} else {
		cmd = in.Command("-m", "venv", v.Dir)
	}
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "ensurepip") {
			msg += "\nhint: on Debian/Ubuntu install the venv module: sudo apt install python3-venv"
		}
		return fmt.Errorf("creating virtual environment failed: %s", msg)
	}
	if v.uv == "" {
		// A stale pip is the most common cause of failed installs; best effort.
		_ = v.command("install", "--quiet", "--upgrade", "pip").Run()
	}
	return nil
}

// command builds a pip (or `uv pip`) invocation for this venv.
func (v Venv) command(args ...string) *exec.Cmd {
	if v.uv != "" {
		filtered := args[:0:0]
		for _, a := range args {
			if a != "--yes" { // uv pip uninstall never prompts and rejects --yes
				filtered = append(filtered, a)
			}
		}
		cmd := exec.Command(v.uv, append(append([]string{"pip"}, filtered...), "--python", v.Python())...)
		return cmd
	}
	cmd := exec.Command(v.Python(), append([]string{"-m", "pip"}, args...)...)
	cmd.Env = append(os.Environ(), "PIP_DISABLE_PIP_VERSION_CHECK=1")
	return cmd
}

// Pip runs pip inside the venv with output streamed to the terminal.
func (v Venv) Pip(args ...string) error {
	cmd := v.command(args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s failed", v.Installer(), args[0])
	}
	return nil
}

// PipOutput runs pip and returns its stdout.
func (v Venv) PipOutput(args ...string) ([]byte, error) {
	cmd := v.command(args...)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

// Package is one installed distribution.
type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Latest  string `json:"latest_version,omitempty"`
}

// Packages lists everything installed in the venv, sorted by name.
func (v Venv) Packages() ([]Package, error) {
	return v.listJSON("list", "--format=json")
}

// Outdated lists installed packages that have a newer release.
func (v Venv) Outdated() ([]Package, error) {
	return v.listJSON("list", "--outdated", "--format=json")
}

func (v Venv) listJSON(args ...string) ([]Package, error) {
	out, err := v.PipOutput(args...)
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w", v.Installer(), strings.Join(args, " "), err)
	}
	var pkgs []Package
	if err := json.Unmarshal(out, &pkgs); err != nil {
		return nil, fmt.Errorf("%s list returned bad JSON: %w", v.Installer(), err)
	}
	sort.Slice(pkgs, func(i, j int) bool { return project.Normalize(pkgs[i].Name) < project.Normalize(pkgs[j].Name) })
	return pkgs, nil
}

// Installed maps normalized package names to installed versions.
func (v Venv) Installed() (map[string]string, error) {
	pkgs, err := v.Packages()
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(pkgs))
	for _, p := range pkgs {
		m[project.Normalize(p.Name)] = p.Version
	}
	return m, nil
}

// Activate puts the venv on PATH for this process and its children,
// the same as `source .venv/bin/activate`.
func (v Venv) Activate() {
	os.Setenv("VIRTUAL_ENV", v.Dir)
	os.Setenv("PATH", v.BinDir()+string(os.PathListSeparator)+os.Getenv("PATH"))
	os.Unsetenv("PYTHONHOME")
}
