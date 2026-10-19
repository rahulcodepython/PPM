// Package scaffold holds the starter templates written by `ppm init`.
package scaffold

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"text/template"
)

// Template describes a starter project.
type Template struct {
	Name        string
	Description string
	Files       map[string]string // relative path to text/template source
	Deps        []string
	Scripts     map[string]string
}

// Data is passed to file templates.
type Data struct {
	Name        string
	Description string
	Author      string
	License     string
}

var templates = map[string]Template{
	"app": {
		Name:        "app",
		Description: "plain Python app with src/main.py",
		Files:       map[string]string{"src/main.py": appMain},
		Scripts:     map[string]string{"start": "python src/main.py"},
	},
	"cli": {
		Name:        "cli",
		Description: "command-line tool using argparse",
		Files:       map[string]string{"src/main.py": cliMain},
		Scripts:     map[string]string{"start": "python src/main.py"},
	},
	"fastapi": {
		Name:        "fastapi",
		Description: "FastAPI web API served by uvicorn",
		Files:       map[string]string{"src/main.py": fastapiMain},
		Deps:        []string{"fastapi", "uvicorn[standard]"},
		Scripts:     map[string]string{"start": "python -m uvicorn main:app --app-dir src --reload"},
	},
	"flask": {
		Name:        "flask",
		Description: "Flask web app",
		Files:       map[string]string{"src/app.py": flaskApp},
		Deps:        []string{"flask"},
		Scripts:     map[string]string{"start": "python -m flask --app src/app.py run --debug"},
	},
}

// common files go into every project.
var common = map[string]string{
	"README.md":  readme,
	".gitignore": gitignore,
	".env":       "",
}

// Get returns the named template.
func Get(name string) (Template, bool) {
	t, ok := templates[name]
	return t, ok
}

// Names lists template names in sorted order.
func Names() []string {
	names := make([]string, 0, len(templates))
	for n := range templates {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Write renders the template into dir. Existing files are never
// overwritten; their paths are returned in skipped.
func (t Template) Write(dir string, data Data) (created, skipped []string, err error) {
	files := map[string]string{}
	for k, v := range common {
		files[k] = v
	}
	for k, v := range t.Files {
		files[k] = v
	}
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, rel := range paths {
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		if _, err := os.Stat(dst); !errors.Is(err, fs.ErrNotExist) {
			skipped = append(skipped, rel)
			continue
		}
		tmpl, err := template.New(rel).Parse(files[rel])
		if err != nil {
			return created, skipped, err
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return created, skipped, err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return created, skipped, err
		}
		if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
			return created, skipped, err
		}
		created = append(created, rel)
	}
	return created, skipped, nil
}

const appMain = `def main() -> None:
    print("Hello from {{.Name}}!")


if __name__ == "__main__":
    main()
`

const cliMain = `import argparse


def main() -> None:
    parser = argparse.ArgumentParser(prog="{{.Name}}", description="{{if .Description}}{{.Description}}{{else}}{{.Name}} command-line tool{{end}}")
    parser.add_argument("name", nargs="?", default="world", help="who to greet")
    parser.add_argument("-u", "--upper", action="store_true", help="shout the greeting")
    args = parser.parse_args()

    greeting = f"Hello, {args.name}!"
    print(greeting.upper() if args.upper else greeting)


if __name__ == "__main__":
    main()
`

const fastapiMain = `from fastapi import FastAPI

app = FastAPI(title="{{.Name}}")


@app.get("/")
def root() -> dict[str, str]:
    return {"message": "Hello from {{.Name}}!"}
`

const flaskApp = `from flask import Flask

app = Flask(__name__)


@app.get("/")
def index() -> str:
    return "Hello from {{.Name}}!"
`

const readme = `# {{.Name}}
{{if .Description}}
{{.Description}}
{{end}}
## Getting started

` + "```" + `bash
ppm install      # create .venv and install dependencies
ppm run          # run the "start" script
` + "```" + `

## Common commands

` + "```" + `bash
ppm add <pkg>         # install a package and save it to ppm.yml
ppm add -D <pkg>      # install a dev-only package
ppm remove <pkg>      # uninstall a package
ppm run <script>      # run a script from ppm.yml
ppm exec <cmd>        # run any command inside the venv
ppm env set KEY=VAL   # add a variable to .env
ppm freeze            # write pinned requirements.txt
ppm --help            # everything else
` + "```" + `
`

const gitignore = `# Python
__pycache__/
*.py[cod]
*.egg-info/
.pytest_cache/
.mypy_cache/
.ruff_cache/
.coverage
htmlcov/
build/
dist/

# Environment
.venv/
venv/
.env

# Editors
.idea/
.vscode/
.DS_Store
`
