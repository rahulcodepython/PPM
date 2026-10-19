package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rahulcodepython/PPM/internal/project"
	"github.com/rahulcodepython/PPM/internal/ui"
)

func newRunCmd() *cobra.Command {
	var list bool
	cmd := &cobra.Command{
		Use:     "run [script|file.py] [args]...",
		GroupID: "run",
		Short:   "Run a script from ppm.yml (default: start)",
		Long: `Run a script defined under "scripts" in ppm.yml, with the venv
activated and .env loaded. Extra arguments are passed to the script.
A path ending in .py runs that file with the venv's Python.

Scripts named "pre<name>" and "post<name>" run before and after <name>,
e.g. "prestart" runs before "start". Scripts can call other scripts:
  test: python -m pytest
  check: ppm run lint && ppm run test`,
		Example: `  ppm run                 # runs the "start" script
  ppm run test -k login   # runs "test" with extra args
  ppm run -- --port 9000  # runs "start" with extra args
  ppm run tools/seed.py   # run a file with the venv's Python
  ppm run -l              # list scripts`,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveDefault
			}
			p, err := project.Find(".")
			if err != nil {
				return nil, cobra.ShellCompDirectiveDefault
			}
			return p.ScriptNames(), cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			if list {
				printScripts(p)
				return nil
			}
			name := "start"
			if cmd.ArgsLenAtDash() == 0 { // `ppm run -- --flag`: args go to start
			} else if len(args) > 0 {
				name, args = args[0], args[1:]
			}

			script, ok := p.Scripts[name]
			if !ok && !strings.HasSuffix(name, ".py") {
				if len(p.Scripts) == 0 {
					return fmt.Errorf("no scripts defined; add one under \"scripts:\" in %s", project.FileName)
				}
				printScripts(p)
				return fmt.Errorf("script %q not found", name)
			}

			v, err := readyVenv(p)
			if err != nil {
				return err
			}
			if err := activate(p, v); err != nil {
				return err
			}
			if !ok { // a .py file
				return runChild(exec.Command(v.Python(), append([]string{name}, args...)...))
			}
			if pre, ok := p.Scripts["pre"+name]; ok {
				if err := runScript(pre, nil); err != nil {
					return err
				}
			}
			if err := runScript(script, args); err != nil {
				return err
			}
			if post, ok := p.Scripts["post"+name]; ok {
				return runScript(post, nil)
			}
			return nil
		},
	}
	// Flags after the script name belong to the script, not ppm.
	cmd.Flags().SetInterspersed(false)
	cmd.Flags().BoolVarP(&list, "list", "l", false, "list available scripts")
	return cmd
}

func runScript(script string, args []string) error {
	fmt.Fprintln(os.Stderr, ui.Dim(strings.TrimSpace("> "+script+" "+strings.Join(args, " "))))
	return runChild(shellCommand(script, args))
}

func printScripts(p *project.Project) {
	if len(p.Scripts) == 0 {
		fmt.Println("No scripts defined in " + project.FileName)
		return
	}
	rows := make([][]string, 0, len(p.Scripts))
	for _, n := range p.ScriptNames() {
		rows = append(rows, []string{ui.Cyan(n), p.Scripts[n]})
	}
	ui.Table(os.Stdout, []string{"SCRIPT", "COMMAND"}, rows)
}

func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "exec <command> [args]...",
		Aliases: []string{"x"},
		GroupID: "run",
		Short:   "Run any command inside the venv",
		Example: `  ppm exec pytest -q
  ppm exec python -c "import sys; print(sys.prefix)"
  ppm x pip show requests`,
		Args:        needArgs(1, "command to run"),
		Annotations: argsHelp("<command>   program to run, found in .venv first (required)", "[args]...  arguments passed to it"),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			v, err := readyVenv(p)
			if err != nil {
				return err
			}
			if err := activate(p, v); err != nil {
				return err
			}
			// Look up after activation so venv executables win.
			path, err := exec.LookPath(args[0])
			if err != nil {
				return fmt.Errorf("%s: command not found (install it with `ppm add -D %s`?)", args[0], args[0])
			}
			return runChild(exec.Command(path, args[1:]...))
		},
	}
	cmd.Flags().SetInterspersed(false)
	return cmd
}

func newShellCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "shell",
		GroupID: "run",
		Short:   "Open a new shell with the venv activated",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			v, err := readyVenv(p)
			if err != nil {
				return err
			}
			if os.Getenv("VIRTUAL_ENV") == v.Dir {
				return fmt.Errorf("already inside this project's venv")
			}
			if err := activate(p, v); err != nil {
				return err
			}
			sh := os.Getenv("SHELL")
			if runtime.GOOS == "windows" {
				sh = os.Getenv("COMSPEC")
				if sh == "" {
					sh = "cmd.exe"
				}
			} else if sh == "" {
				sh = "/bin/sh"
			}
			ui.Success("Entered %s venv. Type `exit` to leave.", p.Name)
			err = runChild(exec.Command(sh))
			if _, isExit := err.(*exitCodeError); isExit {
				return nil // the shell's last exit status is not ppm's problem
			}
			return err
		},
	}
}
