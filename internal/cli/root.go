// Package cli wires the ppm commands together with cobra.
package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rahulcodepython/PPM/internal/dotenv"
	"github.com/rahulcodepython/PPM/internal/project"
	"github.com/rahulcodepython/PPM/internal/python"
	"github.com/rahulcodepython/PPM/internal/ui"
)

// Set at build time: -ldflags "-X github.com/rahulcodepython/PPM/internal/cli.version=1.2.3"
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var pseudoVersion = regexp.MustCompile(`\d{14}-[0-9a-f]{12}`)

// resolvedVersion prefers the ldflags version, then the module version
// recorded by `go install ...@vX.Y.Z`.
func resolvedVersion() string {
	if version != "dev" {
		return version
	}
	// Local builds get VCS pseudo-versions like v0.0.0-20250415205020-e551e83a46e5; skip those.
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" &&
		!pseudoVersion.MatchString(bi.Main.Version) {
		return strings.TrimPrefix(bi.Main.Version, "v")
	}
	return version
}

// exitCodeError carries a child process exit code without printing anything.
type exitCodeError struct{ code int }

func (e *exitCodeError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

// NewRootCmd builds the ppm command tree.
func NewRootCmd() *cobra.Command {
	var chdir string
	root := &cobra.Command{
		Use:   "ppm",
		Short: "Python Project Manager: scaffold projects, manage venvs and dependencies, run scripts",
		Long: `ppm is a single-binary project manager for Python.

It creates projects, keeps a virtual environment in .venv, records your
dependencies in ppm.yml and runs project scripts with the venv and .env loaded.`,
		Example: `  ppm init myapp           # new project in ./myapp
  ppm add requests         # install and save a dependency
  ppm add -D pytest        # install and save a dev dependency
  ppm run                  # run the "start" script
  ppm exec pytest -q       # run any command inside the venv
  ppm -C ../other list     # act on another project

Docs: https://github.com/rahulcodepython/PPM#readme`,
		Version:       resolvedVersion(),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if chdir != "" {
				return os.Chdir(chdir)
			}
			return nil
		},
	}
	root.PersistentFlags().StringVarP(&chdir, "cwd", "C", "", "run as if ppm was started in this directory")
	root.SetVersionTemplate("ppm {{.Version}}\n")
	root.SetUsageTemplate(usageTemplate)

	root.AddGroup(
		&cobra.Group{ID: "project", Title: "Project:"},
		&cobra.Group{ID: "deps", Title: "Dependencies:"},
		&cobra.Group{ID: "run", Title: "Running:"},
		&cobra.Group{ID: "tool", Title: "ppm itself:"},
	)
	root.AddCommand(
		newInitCmd(), newDoctorCmd(), newUseCmd(), newCleanCmd(), newEnvCmd(),
		newAddCmd(), newInstallCmd(), newRemoveCmd(), newUpdateCmd(),
		newListCmd(), newOutdatedCmd(), newFreezeCmd(),
		newRunCmd(), newExecCmd(), newShellCmd(), newScriptCmd(),
		newVersionCmd(), newSelfCmd(), newSelfUpdateAlias(),
	)
	return root
}

// Execute runs the CLI and returns the process exit code.
func Execute() int {
	err := NewRootCmd().Execute()
	if err == nil {
		return 0
	}
	var ec *exitCodeError
	if errors.As(err, &ec) {
		return ec.code
	}
	ui.Error("%v", err)
	return 1
}

// loadProject finds ppm.yml and moves into the project root so relative
// paths in scripts behave the same from any subdirectory.
func loadProject() (*project.Project, error) {
	p, err := project.Find(".")
	if err != nil {
		return nil, err
	}
	if p.Migrated {
		ui.Warn("%s uses the old ppm3 format; it will be rewritten in the new format on next save", project.FileName)
		ui.Warn("old files listed every installed package; review `dependencies` and keep only the ones you import")
	}
	return p, os.Chdir(p.Root)
}

// projectVenv returns the project venv handle without creating it.
func projectVenv(p *project.Project) (python.Venv, error) {
	return python.NewVenv(p.VenvDir(), p.Installer)
}

// ensureVenv returns the project venv, creating it if needed.
func ensureVenv(p *project.Project) (v python.Venv, created bool, err error) {
	if v, err = projectVenv(p); err != nil || v.Exists() {
		return v, false, err
	}
	in, err := python.Find(p.Python)
	if err != nil {
		return v, false, err
	}
	return v, true, createVenv(p, v, in)
}

func createVenv(p *project.Project, v python.Venv, in *python.Interpreter) error {
	ui.Step("Creating virtual environment in %s (Python %s, %s)", p.Venv, in.Version, v.Installer())
	if err := v.Create(in); err != nil {
		return err
	}
	ui.Success("Virtual environment ready")
	return nil
}

// readyVenv is ensureVenv plus installing ppm.yml dependencies when the
// venv is new, so commands work right after a fresh clone.
func readyVenv(p *project.Project) (python.Venv, error) {
	v, created, err := ensureVenv(p)
	if err != nil || !created {
		return v, err
	}
	return v, installAll(p, v, true)
}

func installAll(p *project.Project, v python.Venv, dev bool) error {
	deps := p.AllDependencies(dev)
	if len(deps) == 0 {
		return nil
	}
	ui.Step("Installing dependencies (%d)", len(deps))
	if err := v.Pip(append([]string{"install"}, deps...)...); err != nil {
		return err
	}
	ui.Success("Dependencies installed")
	return nil
}

// activate loads .env (without overriding variables already set) and puts
// the venv on PATH.
func activate(p *project.Project, v python.Venv) error {
	vars, err := dotenv.Load(p.EnvFilePath())
	if err != nil {
		return fmt.Errorf("reading %s: %w", p.EnvFile, err)
	}
	for k, val := range vars {
		if _, set := os.LookupEnv(k); !set {
			os.Setenv(k, val)
		}
	}
	v.Activate()
	return nil
}

// runChild runs an interactive child process. Ctrl+C goes to the child
// only, and its exit code becomes ppm's exit code.
func runChild(c *exec.Cmd) error {
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)

	err := c.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code := ee.ExitCode()
		if code < 0 { // killed by a signal
			code = 130
		}
		return &exitCodeError{code: code}
	}
	return err
}
