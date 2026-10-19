package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rahulcodepython/PPM/internal/project"
	"github.com/rahulcodepython/PPM/internal/python"
	"github.com/rahulcodepython/PPM/internal/ui"
)

func newAddCmd() *cobra.Command {
	var dev bool
	cmd := &cobra.Command{
		Use:     "add <package>...",
		GroupID: "deps",
		Short:   "Install packages and save them to ppm.yml",
		Long: `Install packages into the venv and record them in ppm.yml.

Packages without a version are saved as "name>=installed-version".
Any pip requirement works: versions, extras, git URLs, local paths.`,
		Example: `  ppm add requests
  ppm add "django>=5,<6" "uvicorn[standard]"
  ppm add -D pytest ruff`,
		Args:        needArgs(1, "package to add"),
		Annotations: argsHelp("<package>...   one or more pip requirements: name, name==1.2, \"name[extra]>=1\", git URL or path (required)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			v, err := readyVenv(p)
			if err != nil {
				return err
			}
			return addPackages(p, v, args, dev)
		},
	}
	cmd.Flags().BoolVarP(&dev, "dev", "D", false, "save as a dev dependency")
	return cmd
}

func newInstallCmd() *cobra.Command {
	var (
		dev, prod, clean bool
		reqFile          string
	)
	cmd := &cobra.Command{
		Use:     "install [package]...",
		Aliases: []string{"i"},
		GroupID: "deps",
		Short:   "Install all dependencies from ppm.yml (or add packages)",
		Long: `Without arguments, create the venv if needed and install every
dependency listed in ppm.yml. With arguments, behave like "ppm add".`,
		Example: `  ppm install                       # after cloning a project
  ppm install --prod                # skip dev-dependencies
  ppm install --clean               # rebuild .venv from scratch
  ppm install -r requirements.txt   # import an existing requirements file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			if clean {
				ui.Step("Removing %s", p.Venv)
				if err := os.RemoveAll(p.VenvDir()); err != nil {
					return err
				}
			}
			if reqFile != "" {
				reqs, err := readRequirements(reqFile)
				if err != nil {
					return err
				}
				args = append(args, reqs...)
			}
			v, created, err := ensureVenv(p)
			if err != nil {
				return err
			}
			if len(args) > 0 {
				if created {
					if err := installAll(p, v, !prod); err != nil {
						return err
					}
				}
				return addPackages(p, v, args, dev)
			}
			if len(p.AllDependencies(!prod)) == 0 {
				ui.Success("Nothing to install; add packages with `ppm add <package>`")
				return nil
			}
			if err := installAll(p, v, !prod); err != nil {
				return err
			}
			return saveIfMigrated(p)
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&dev, "dev", "D", false, "save packages given as arguments as dev dependencies")
	f.BoolVar(&prod, "prod", false, "skip dev-dependencies")
	f.BoolVar(&clean, "clean", false, "delete and recreate the virtual environment first")
	f.StringVarP(&reqFile, "requirements", "r", "", "add every package from a requirements file")
	return cmd
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <package>...",
		Aliases: []string{"rm", "uninstall"},
		GroupID: "deps",
		Short:   "Uninstall Python packages and drop them from ppm.yml",
		Long: `Uninstall Python packages from the project's .venv and remove them
from ppm.yml.

This does not uninstall ppm itself; for that run "ppm self uninstall".`,
		Example: `  ppm remove requests
  ppm rm flask pytest`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("%w\n\nTo uninstall ppm itself, run `ppm self uninstall`.", missingArgs(cmd, "package name to remove"))
			}
			return nil
		},
		Annotations: argsHelp("<package>...   one or more installed package names (required)"),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			names := make([]string, 0, len(args))
			for _, a := range args {
				if r, ok := project.Parse(a); ok {
					names = append(names, r.Name)
				} else {
					names = append(names, a)
				}
				if !p.RemoveDependency(a) {
					ui.Warn("%s is not listed in %s", a, project.FileName)
				}
			}
			v, err := projectVenv(p)
			if err != nil {
				return err
			}
			if v.Exists() {
				ui.Step("Uninstalling %s", strings.Join(names, ", "))
				if err := v.Pip(append([]string{"uninstall", "--yes"}, names...)...); err != nil {
					return err
				}
			}
			if err := p.Save(); err != nil {
				return err
			}
			ui.Success("Removed %s", strings.Join(names, ", "))
			return nil
		},
	}
}

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "update [package]...",
		Aliases: []string{"upgrade", "up"},
		GroupID: "deps",
		Short:   "Upgrade dependencies to their latest versions",
		Long: `Upgrade the named packages, or every dependency in ppm.yml when none
are given. Simple specs such as ">=1.0" or "==1.0" are bumped to the new version.`,
		Example: `  ppm update
  ppm update requests`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			v, err := readyVenv(p)
			if err != nil {
				return err
			}
			targets := args
			if len(targets) == 0 {
				targets = p.AllDependencies(true)
			}
			known := map[string]bool{}
			for _, req := range p.AllDependencies(true) {
				known[project.Key(req)] = true
			}
			for _, a := range args {
				if !known[project.Key(a)] {
					ui.Warn("%s is not in %s; upgrading it without saving (use `ppm add` to save it)", a, project.FileName)
				}
			}
			var pkgs []string
			for _, t := range targets {
				if r, ok := project.Parse(t); ok {
					pkgs = append(pkgs, r.Name+r.Extras) // drop the spec so pip can move past it
				} else {
					pkgs = append(pkgs, t)
				}
			}
			if len(pkgs) == 0 {
				ui.Success("No dependencies to update")
				return nil
			}
			ui.Step("Upgrading %s", strings.Join(pkgs, ", "))
			if err := v.Pip(append([]string{"install", "--upgrade"}, pkgs...)...); err != nil {
				return err
			}
			installed, err := v.Installed()
			if err != nil {
				return err
			}
			for _, req := range p.AllDependencies(true) {
				if ver, ok := installed[project.Key(req)]; ok {
					p.ReplaceDependency(project.BumpVersion(req, ver))
				}
			}
			if err := p.Save(); err != nil {
				return err
			}
			ui.Success("Updated")
			return nil
		},
	}
}

// addPackages installs reqs and records them in ppm.yml.
func addPackages(p *project.Project, v python.Venv, reqs []string, dev bool) error {
	ui.Step("Installing %s", strings.Join(reqs, ", "))
	if err := v.Pip(append([]string{"install"}, reqs...)...); err != nil {
		return err
	}
	installed, err := v.Installed()
	if err != nil {
		return err
	}
	for _, req := range reqs {
		if r, ok := project.Parse(req); ok && r.Spec == "" {
			if ver, ok := installed[project.Normalize(r.Name)]; ok {
				r.Spec = ">=" + ver
				req = r.String()
			}
		}
		p.AddDependency(req, dev)
	}
	if err := p.Save(); err != nil {
		return err
	}
	kind := "dependencies"
	if dev {
		kind = "dev-dependencies"
	}
	ui.Success("Saved to %s %s", project.FileName, kind)
	return nil
}

func saveIfMigrated(p *project.Project) error {
	if !p.Migrated {
		return nil
	}
	if err := p.Save(); err != nil {
		return err
	}
	ui.Success("Rewrote %s in the new format", project.FileName)
	return nil
}

// readRequirements returns the requirement lines of a requirements file,
// skipping comments and pip options.
func readRequirements(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var reqs []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if i := strings.Index(line, " #"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		reqs = append(reqs, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(reqs) == 0 {
		return nil, fmt.Errorf("%s has no requirements", path)
	}
	return reqs, nil
}
