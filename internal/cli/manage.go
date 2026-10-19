package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rahulcodepython/PPM/internal/dotenv"
	"github.com/rahulcodepython/PPM/internal/python"
	"github.com/rahulcodepython/PPM/internal/ui"
)

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "use <python>",
		GroupID: "project",
		Short:   "Switch the project to another Python version",
		Long: `Save the Python version in ppm.yml, rebuild .venv with it and
reinstall all dependencies.`,
		Example: `  ppm use 3.12
  ppm use python3.11
  ppm use /opt/python/bin/python3`,
		Args:        exactArgs(1, "Python version or path"),
		Annotations: argsHelp("<python>   version (3.12), command (python3.11) or path (required)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			in, err := python.Find(args[0])
			if err != nil {
				return err
			}
			v, err := projectVenv(p)
			if err != nil {
				return err
			}
			ui.Step("Removing %s", p.Venv)
			if err := os.RemoveAll(p.VenvDir()); err != nil {
				return err
			}
			if err := createVenv(p, v, in); err != nil {
				return err
			}
			p.Python = args[0]
			if err := p.Save(); err != nil {
				return err
			}
			if err := installAll(p, v, true); err != nil {
				return err
			}
			ui.Success("Project now uses Python %s", in.Version)
			return nil
		},
	}
}

// Targets removed by `ppm clean`. cleanDirs match anywhere in the tree,
// rootDirs only at the project root (a package may be named "build"),
// and cleanPatterns match file or directory names.
var (
	cleanDirs     = []string{"__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache"}
	rootDirs      = []string{"build", "dist", "htmlcov"}
	cleanPatterns = []string{"*.pyc", "*.pyo", "*.egg-info", ".coverage"}
)

func newCleanCmd() *cobra.Command {
	var withVenv, dryRun bool
	cmd := &cobra.Command{
		Use:     "clean",
		GroupID: "project",
		Short:   "Delete caches and build output (__pycache__, dist, ...)",
		Long: "Delete " + strings.Join(append(append([]string{}, cleanDirs...), cleanPatterns...), ", ") + `
anywhere in the project, and ` + strings.Join(rootDirs, ", ") + ` at the project root.
.git and the venv are skipped unless --venv is given.`,
		Example: `  ppm clean --dry-run   # show what would be deleted
  ppm clean
  ppm clean --venv      # also delete .venv`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			venv := p.VenvDir()
			var targets []string
			err = filepath.WalkDir(p.Root, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if path == p.Root {
					return nil
				}
				if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules" || path == venv) {
					return filepath.SkipDir
				}
				if matchesClean(d, filepath.Dir(path) == p.Root) {
					targets = append(targets, path)
					if d.IsDir() {
						return filepath.SkipDir
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
			if withVenv && isDir(venv) {
				targets = append(targets, venv)
			}
			if len(targets) == 0 {
				ui.Success("Already clean")
				return nil
			}
			for _, t := range targets {
				rel, _ := filepath.Rel(p.Root, t)
				if dryRun {
					fmt.Println("would remove", rel)
					continue
				}
				if err := os.RemoveAll(t); err != nil {
					return err
				}
				fmt.Println(ui.Dim("removed " + rel))
			}
			if !dryRun {
				ui.Success("Removed %d items", len(targets))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&withVenv, "venv", false, "also delete the virtual environment")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "only print what would be deleted")
	return cmd
}

func matchesClean(d fs.DirEntry, atRoot bool) bool {
	if d.IsDir() {
		dirs := cleanDirs
		if atRoot {
			dirs = append(append([]string{}, cleanDirs...), rootDirs...)
		}
		for _, n := range dirs {
			if d.Name() == n {
				return true
			}
		}
	}
	for _, pat := range cleanPatterns {
		if ok, _ := filepath.Match(pat, d.Name()); ok {
			return true
		}
	}
	return false
}

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "env",
		GroupID: "project",
		Short:   "List, set or remove variables in the project .env file",
		Example: `  ppm env list
  ppm env set DEBUG=1 DATABASE_URL=postgres://localhost/app
  ppm env unset DEBUG`,
	}
	var show bool
	list := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List variables (values hidden unless --show)",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			vars, err := dotenv.Load(p.EnvFilePath())
			if err != nil {
				return err
			}
			if len(vars) == 0 {
				fmt.Printf("%s is empty. Add a variable with `ppm env set KEY=VALUE`.\n", p.EnvFile)
				return nil
			}
			keys := make([]string, 0, len(vars))
			for k := range vars {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			rows := make([][]string, 0, len(keys))
			for _, k := range keys {
				v := ui.Dim("•••••• (use --show)")
				if show {
					v = vars[k]
				}
				rows = append(rows, []string{k, v})
			}
			ui.Table(os.Stdout, []string{"VARIABLE", "VALUE"}, rows)
			return nil
		},
	}
	list.Flags().BoolVarP(&show, "show", "s", false, "print values")

	set := &cobra.Command{
		Use:   "set KEY=VALUE...",
		Short: "Add or change variables",
		Example: `  ppm env set DEBUG=1
  ppm env set "GREETING=hello world" PORT=8000`,
		Args:        needArgs(1, "KEY=VALUE pair"),
		Annotations: argsHelp("KEY=VALUE...   one or more variables to write to .env (required)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			vars := map[string]string{}
			for _, a := range args {
				k, v, ok := strings.Cut(a, "=")
				if !ok || strings.TrimSpace(k) == "" {
					return fmt.Errorf("%q is not KEY=VALUE", a)
				}
				vars[strings.TrimSpace(k)] = v
			}
			if err := dotenv.Set(p.EnvFilePath(), vars); err != nil {
				return err
			}
			ui.Success("Updated %s", p.EnvFile)
			return nil
		},
	}

	unset := &cobra.Command{
		Use:         "unset KEY...",
		Aliases:     []string{"rm"},
		Short:       "Remove variables",
		Example:     `  ppm env unset DEBUG PORT`,
		Args:        needArgs(1, "variable name to remove"),
		Annotations: argsHelp("KEY...   one or more variable names (required)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			removed, err := dotenv.Unset(p.EnvFilePath(), args)
			if err != nil {
				return err
			}
			if len(removed) == 0 {
				ui.Warn("none of %s found in %s", strings.Join(args, ", "), p.EnvFile)
				return nil
			}
			ui.Success("Removed %s from %s", strings.Join(removed, ", "), p.EnvFile)
			return nil
		},
	}
	cmd.AddCommand(list, set, unset)
	return cmd
}

func newScriptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "script",
		Aliases: []string{"scripts"},
		GroupID: "run",
		Short:   "List, add or remove scripts in ppm.yml",
		Example: `  ppm script list
  ppm script add test python -m pytest -q
  ppm script add lint "ruff check . && ruff format --check ."
  ppm script remove lint`,
	}
	list := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List scripts",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			printScripts(p)
			return nil
		},
	}
	add := &cobra.Command{
		Use:   "add <name> <command>...",
		Short: "Add or replace a script",
		Example: `  ppm script add test python -m pytest -q
  ppm script add check "ruff check . && ppm run test"`,
		Long: `Add or replace a script. Words after the name form the command.
Quote the command when it contains shell operators such as && or |.`,
		Args:        needArgs(2, "script name and command"),
		Annotations: argsHelp("<name>         script name, used as `ppm run <name>` (required)", "<command>...   shell command; quote it if it has && or | (required)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			name, command := args[0], strings.Join(args[1:], " ")
			_, existed := p.Scripts[name]
			p.Scripts[name] = command
			if err := p.Save(); err != nil {
				return err
			}
			verb := "Added"
			if existed {
				verb = "Replaced"
			}
			ui.Success("%s script %s: %s", verb, ui.Cyan(name), command)
			return nil
		},
	}
	add.Flags().SetInterspersed(false)
	remove := &cobra.Command{
		Use:         "remove <name>...",
		Aliases:     []string{"rm"},
		Short:       "Remove scripts",
		Example:     `  ppm script remove lint`,
		Args:        needArgs(1, "script name to remove"),
		Annotations: argsHelp("<name>...   one or more script names (required)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			for _, n := range args {
				if _, ok := p.Scripts[n]; !ok {
					return fmt.Errorf("script %q not found", n)
				}
				delete(p.Scripts, n)
			}
			if err := p.Save(); err != nil {
				return err
			}
			ui.Success("Removed %s", strings.Join(args, ", "))
			return nil
		},
	}
	cmd.AddCommand(list, add, remove)
	return cmd
}
