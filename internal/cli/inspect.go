package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rahulcodepython/PPM/internal/project"
	"github.com/rahulcodepython/PPM/internal/python"
	"github.com/rahulcodepython/PPM/internal/ui"
)

// pkgRow is one line of `ppm list` / `ppm outdated`, also used for --json.
type pkgRow struct {
	Name      string `json:"name"`
	Required  string `json:"required,omitempty"`
	Installed string `json:"installed,omitempty"`
	Latest    string `json:"latest,omitempty"`
	Type      string `json:"type"` // prod, dev or sub (installed as a dependency of another package)
}

// depIndex maps normalized names to the project's requirement and type.
func depIndex(p *project.Project) map[string]pkgRow {
	idx := map[string]pkgRow{}
	add := func(reqs []string, kind string) {
		for _, req := range reqs {
			row := pkgRow{Name: req, Required: "*", Type: kind}
			if r, ok := project.Parse(req); ok {
				row.Name = r.Name + r.Extras
				if r.Spec != "" {
					row.Required = r.Spec
				}
			}
			idx[project.Key(req)] = row
		}
	}
	add(p.Dependencies, "prod")
	add(p.DevDependencies, "dev")
	return idx
}

func typeLabel(t string) string {
	switch t {
	case "prod":
		return ui.Green("prod")
	case "dev":
		return ui.Cyan("dev")
	}
	return ui.Dim("sub")
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func newListCmd() *cobra.Command {
	var all, asJSON bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		GroupID: "deps",
		Short:   "Show dependencies in a table: required spec, installed version, type",
		Long: `Show the project's dependencies in a table with the version ppm.yml
requires, the version installed in .venv, and the type:

  prod   listed under dependencies
  dev    listed under dev-dependencies
  sub    not in ppm.yml; installed because another package needs it (--all)`,
		Example: `  ppm list
  ppm list --all    # every package in .venv, including sub-dependencies
  ppm list --json   # machine-readable`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			v, err := readyVenv(p)
			if err != nil {
				return err
			}
			pkgs, err := v.Packages()
			if err != nil {
				return err
			}
			deps := depIndex(p)

			var rows []pkgRow
			seen := map[string]bool{}
			if all {
				for _, pkg := range pkgs {
					key := project.Normalize(pkg.Name)
					row, ok := deps[key]
					if !ok {
						row = pkgRow{Name: pkg.Name, Type: "sub"}
					}
					row.Installed = pkg.Version
					rows = append(rows, row)
					seen[key] = true
				}
			}
			installed := map[string]string{}
			for _, pkg := range pkgs {
				installed[project.Normalize(pkg.Name)] = pkg.Version
			}
			for _, req := range p.AllDependencies(true) {
				key := project.Key(req)
				if seen[key] {
					continue
				}
				row := deps[key]
				row.Installed = installed[key]
				rows = append(rows, row)
			}

			if asJSON {
				if rows == nil {
					rows = []pkgRow{}
				}
				return printJSON(rows)
			}
			if len(rows) == 0 {
				fmt.Println("No dependencies yet. Add one with `ppm add <package>`.")
				return nil
			}
			counts := map[string]int{}
			table := make([][]string, 0, len(rows))
			for _, r := range rows {
				counts[r.Type]++
				inst := r.Installed
				if inst == "" {
					inst = ui.Red("missing")
				}
				req := r.Required
				if req == "" {
					req = ui.Dim("-")
				}
				table = append(table, []string{r.Name, req, inst, typeLabel(r.Type)})
			}
			ui.Table(os.Stdout, []string{"PACKAGE", "REQUIRED", "INSTALLED", "TYPE"}, table)

			summary := fmt.Sprintf("%d packages: %d prod, %d dev", len(rows), counts["prod"], counts["dev"])
			if all {
				summary += fmt.Sprintf(", %d sub", counts["sub"])
			}
			fmt.Println(ui.Dim(summary))
			missing := 0
			for _, r := range rows {
				if r.Installed == "" {
					missing++
				}
			}
			if missing > 0 {
				ui.Warn("%d missing; run `ppm install`", missing)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&all, "all", "a", false, "include every installed package")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON instead of a table")
	return cmd
}

func newOutdatedCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "outdated",
		GroupID: "deps",
		Short:   "Show packages with newer versions in a table",
		Example: `  ppm outdated
  ppm outdated --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			v, err := readyVenv(p)
			if err != nil {
				return err
			}
			pkgs, err := v.Outdated()
			if err != nil {
				return err
			}
			deps := depIndex(p)
			rows := []pkgRow{}
			for _, pkg := range pkgs {
				row, ok := deps[project.Normalize(pkg.Name)]
				if !ok {
					row = pkgRow{Name: pkg.Name, Type: "sub"}
				}
				row.Installed, row.Latest = pkg.Version, pkg.Latest
				rows = append(rows, row)
			}
			if asJSON {
				return printJSON(rows)
			}
			if len(rows) == 0 {
				ui.Success("Everything is up to date")
				return nil
			}
			table := make([][]string, 0, len(rows))
			for _, r := range rows {
				req := r.Required
				if req == "" {
					req = ui.Dim("-")
				}
				table = append(table, []string{r.Name, req, r.Installed, ui.Green(r.Latest), typeLabel(r.Type)})
			}
			ui.Table(os.Stdout, []string{"PACKAGE", "REQUIRED", "INSTALLED", "LATEST", "TYPE"}, table)
			fmt.Println(ui.Dim(fmt.Sprintf("%d outdated. Run `ppm update` to upgrade project dependencies.", len(rows))))
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON instead of a table")
	return cmd
}

func newFreezeCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:     "freeze",
		Aliases: []string{"lock"},
		GroupID: "deps",
		Short:   "Write exact installed versions to requirements.txt",
		Example: `  ppm freeze
  ppm freeze -o requirements.lock
  ppm freeze -o -              # print to stdout`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProject()
			if err != nil {
				return err
			}
			v, err := readyVenv(p)
			if err != nil {
				return err
			}
			out, err := v.PipOutput("freeze", "--exclude-editable")
			if err != nil {
				return fmt.Errorf("pip freeze failed: %w", err)
			}
			if output == "-" {
				fmt.Print(string(out))
				return nil
			}
			if err := os.WriteFile(output, out, 0o644); err != nil {
				return err
			}
			n := len(strings.Split(strings.TrimSpace(string(out)), "\n"))
			if strings.TrimSpace(string(out)) == "" {
				n = 0
			}
			ui.Success("Wrote %d packages to %s", n, output)
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "requirements.txt", `output file ("-" for stdout)`)
	return cmd
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "doctor",
		Aliases: []string{"info"},
		GroupID: "project",
		Short:   "Check Python, pip, git and the current project",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ok := func(label, detail string) { fmt.Printf("%s %-12s %s\n", ui.Green("✓"), label, detail) }
			bad := func(label, detail string) { fmt.Printf("%s %-12s %s\n", ui.Red("✗"), label, detail) }

			ok("ppm", fmt.Sprintf("%s (%s/%s)", resolvedVersion(), runtime.GOOS, runtime.GOARCH))
			if in, err := python.Find(""); err == nil {
				ok("python", fmt.Sprintf("%s → %s", in, in.Version))
			} else {
				bad("python", err.Error())
			}
			if out, err := exec.Command("git", "--version").Output(); err == nil {
				ok("git", strings.TrimSpace(string(out)))
			} else {
				bad("git", "not found (optional, needed for `ppm init` git setup)")
			}
			if path, err := exec.LookPath("uv"); err == nil {
				ok("uv", path+" (used for installs; set installer: pip in ppm.yml to opt out)")
			} else {
				fmt.Printf("%s %-12s %s\n", ui.Dim("-"), "uv", "not found (optional, makes installs much faster)")
			}

			p, err := project.Find(".")
			if err != nil {
				bad("project", err.Error())
				return nil
			}
			ok("project", fmt.Sprintf("%s %s (%s)", p.Name, p.Version, p.Path()))
			v, err := projectVenv(p)
			if err != nil {
				bad("installer", err.Error())
				return nil
			}
			if ver, err := v.Version(); err == nil {
				ok("venv", fmt.Sprintf("%s (Python %s, %s)", p.VenvDir(), ver, v.Installer()))
			} else {
				bad("venv", "missing; run `ppm install`")
			}
			ok("deps", fmt.Sprintf("%d prod, %d dev", len(p.Dependencies), len(p.DevDependencies)))
			ok("scripts", strings.Join(p.ScriptNames(), ", "))
			return nil
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the ppm version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ppm %s (commit %s, built %s, %s/%s)\n", resolvedVersion(), commit, date, runtime.GOOS, runtime.GOARCH)
		},
	}
}
