package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rahulcodepython/PPM/internal/project"
	"github.com/rahulcodepython/PPM/internal/python"
	"github.com/rahulcodepython/PPM/internal/scaffold"
	"github.com/rahulcodepython/PPM/internal/ui"
)

type initOptions struct {
	yes       bool
	template  string
	python    string
	noGit     bool
	noVenv    bool
	remote    string
	installer string
}

func newInitCmd() *cobra.Command {
	var o initOptions
	cmd := &cobra.Command{
		Use:     "init [directory]",
		Aliases: []string{"new", "create"},
		GroupID: "project",
		Short:   "Create a new project (ppm.yml, src/, .venv, git)",
		Long: `Create a new Python project in the given directory (default: current directory).

Writes ppm.yml, a starter src/ layout, README.md, .gitignore and .env,
creates a virtual environment, installs template dependencies and
initializes a git repository. Existing files are never overwritten, and an
existing requirements.txt is imported into ppm.yml.

Templates: ` + strings.Join(scaffold.Names(), ", "),
		Example: `  ppm init                       # interactive, in the current directory
  ppm init myapp -y              # new folder, accept all defaults
  ppm init api -t fastapi -y     # FastAPI starter
  ppm init --python 3.12         # pick the Python version
  ppm init --installer pip       # never use uv for this project`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			return runInit(dir, o)
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&o.yes, "yes", "y", false, "accept defaults, ask no questions")
	f.StringVarP(&o.template, "template", "t", "app", "starter template: "+strings.Join(scaffold.Names(), ", "))
	f.StringVarP(&o.python, "python", "p", "", `Python version or command, e.g. "3.12" or "python3.11"`)
	f.BoolVar(&o.noGit, "no-git", false, "skip git init")
	f.BoolVar(&o.noVenv, "no-venv", false, "skip creating the virtual environment")
	f.StringVar(&o.remote, "remote", "", "git remote URL to add as origin")
	f.StringVar(&o.installer, "installer", "", "package installer: auto (uv if found), pip or uv")
	_ = cmd.RegisterFlagCompletionFunc("template", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return scaffold.Names(), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func runInit(dir string, o initOptions) error {
	tmpl, ok := scaffold.Get(o.template)
	if !ok {
		return fmt.Errorf("unknown template %q (choose from: %s)", o.template, strings.Join(scaffold.Names(), ", "))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.Chdir(dir); err != nil {
		return err
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if project.Exists(root) {
		return fmt.Errorf("%s already exists in %s; this project is already initialized", project.FileName, root)
	}

	p := project.New(root, slug(filepath.Base(root)))
	p.Author = gitConfig("user.name")
	p.Python = o.python
	p.Installer = o.installer
	deps := append([]string{}, tmpl.Deps...)
	existing, _ := readRequirements("requirements.txt")
	gitOK := !o.noGit && hasGit() && !isDir(".git")
	remote := o.remote

	if !o.yes {
		fmt.Printf("Creating a new project in %s. Press Enter to accept a default, Ctrl+C to quit.\n\n", ui.Bold(root))
		p.Name = ui.Ask("Project name", p.Name)
		p.Version = ui.Ask("Version", p.Version)
		p.Description = ui.Ask("Description", p.Description)
		p.Author = ui.Ask("Author", p.Author)
		p.License = ui.Ask("License", p.License)
		p.Python = ui.Ask("Python version (blank = default python3)", p.Python)
		if len(existing) > 0 && !ui.Confirm(fmt.Sprintf("Import %d packages from requirements.txt?", len(existing)), true) {
			existing = nil
		}
		if gitOK {
			gitOK = ui.Confirm("Initialize a git repository?", true)
			if gitOK && remote == "" {
				remote = ui.Ask("Git remote URL (optional)", "")
			}
		}
		fmt.Println()
	}

	if len(existing) > 0 {
		ui.Step("Importing %d packages from requirements.txt", len(existing))
		deps = append(deps, existing...)
	}

	// Fail before writing anything if the requested Python is missing.
	var in *python.Interpreter
	if !o.noVenv {
		if in, err = python.Find(p.Python); err != nil {
			return err
		}
	}

	for k, v := range tmpl.Scripts {
		p.Scripts[k] = v
	}
	created, skipped, err := tmpl.Write(root, scaffold.Data{
		Name: p.Name, Description: p.Description, Author: p.Author, License: p.License,
	})
	if err != nil {
		return err
	}
	if err := p.Save(); err != nil {
		return err
	}
	ui.Success("Created %s", strings.Join(append([]string{project.FileName}, created...), ", "))
	if len(skipped) > 0 {
		ui.Warn("Kept existing %s", strings.Join(skipped, ", "))
	}

	if !o.noVenv {
		v, err := projectVenv(p)
		if err != nil {
			return err
		}
		if err := createVenv(p, v, in); err != nil {
			return err
		}
		if len(deps) > 0 {
			if err := addPackages(p, v, deps, false); err != nil {
				return err
			}
		}
	} else {
		for _, d := range deps {
			p.AddDependency(d, false)
		}
		if err := p.Save(); err != nil {
			return err
		}
	}

	if gitOK {
		initGit(remote)
	}

	fmt.Printf("\n%s Project %s is ready.\n\n", ui.Green("✓"), ui.Bold(p.Name))
	if dir != "." {
		fmt.Printf("  cd %s\n", dir)
	}
	fmt.Println("  ppm run           # run the start script")
	fmt.Println("  ppm add <package> # add a dependency")
	fmt.Println("  ppm --help        # all commands")
	return nil
}

func initGit(remote string) {
	steps := [][]string{
		{"init", "--quiet"},
		{"symbolic-ref", "HEAD", "refs/heads/main"},
	}
	if remote != "" {
		steps = append(steps, []string{"remote", "add", "origin", remote})
	}
	for _, s := range steps {
		if out, err := exec.Command("git", s...).CombinedOutput(); err != nil {
			ui.Warn("git %s failed: %s", s[0], strings.TrimSpace(string(out)))
			return
		}
	}
	if out, err := exec.Command("git", "add", "-A").CombinedOutput(); err != nil {
		ui.Warn("git add failed: %s", strings.TrimSpace(string(out)))
		return
	}
	if out, err := exec.Command("git", "commit", "--quiet", "-m", "Initial commit").CombinedOutput(); err != nil {
		ui.Warn("git repository created but initial commit failed: %s", strings.TrimSpace(string(out)))
		return
	}
	msg := "Git repository initialized on branch main"
	if remote != "" {
		msg += " with origin " + remote
	}
	ui.Success("%s", msg)
}

var slugRe = regexp.MustCompile(`[^a-z0-9._-]+`)

func slug(s string) string {
	s = strings.Trim(slugRe.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if s == "" {
		return "my-project"
	}
	return s
}

func hasGit() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func gitConfig(key string) string {
	if !hasGit() {
		return ""
	}
	out, err := exec.Command("git", "config", "--get", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
