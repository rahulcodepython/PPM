// Package project loads and saves the ppm.yml project file.
package project

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// FileName is the project file ppm looks for.
const FileName = "ppm.yml"

// ErrNotFound is returned when no ppm.yml exists in the directory or its parents.
var ErrNotFound = errors.New("no " + FileName + " found here or in any parent directory (run `ppm init` first)")

type Project struct {
	Name            string            `yaml:"name"`
	Version         string            `yaml:"version"`
	Description     string            `yaml:"description"`
	Author          string            `yaml:"author"`
	License         string            `yaml:"license"`
	Python          string            `yaml:"python,omitempty"`
	Installer       string            `yaml:"installer,omitempty"`
	Venv            string            `yaml:"venv"`
	EnvFile         string            `yaml:"env_file"`
	Scripts         map[string]string `yaml:"scripts"`
	Dependencies    []string          `yaml:"dependencies"`
	DevDependencies []string          `yaml:"dev-dependencies"`

	// Root is the directory holding ppm.yml.
	Root string `yaml:"-"`
	// Migrated is true when the file used the old ppm3 (Python) layout.
	Migrated bool `yaml:"-"`
}

// New returns a project with default settings rooted at dir.
func New(dir, name string) *Project {
	p := &Project{Name: name, Version: "0.1.0", License: "MIT", Root: dir}
	p.applyDefaults()
	return p
}

func (p *Project) applyDefaults() {
	if p.Venv == "" {
		p.Venv = ".venv"
	}
	if p.EnvFile == "" {
		p.EnvFile = ".env"
	}
	if p.Scripts == nil {
		p.Scripts = map[string]string{}
	}
}

// Find walks up from dir looking for ppm.yml and loads it.
func Find(dir string) (*Project, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	for {
		path := filepath.Join(dir, FileName)
		if _, err := os.Stat(path); err == nil {
			return Load(path)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, ErrNotFound
		}
		dir = parent
	}
}

// Exists reports whether dir contains a ppm.yml.
func Exists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, FileName))
	return !errors.Is(err, fs.ErrNotExist)
}

// Load reads a ppm.yml file, converting the old ppm3 layout if needed.
func Load(path string) (*Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	p := &Project{}
	if _, legacy := raw["project"]; legacy {
		if p, err = fromLegacy(data); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	} else if err := yaml.Unmarshal(data, p); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	p.Root = filepath.Dir(path)
	p.applyDefaults()
	return p, nil
}

// legacyFile mirrors the ppm.yml written by ppm3 1.x.
type legacyFile struct {
	Project struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Description string `yaml:"description"`
		Author      string `yaml:"author"`
		License     string `yaml:"license"`
	} `yaml:"project"`
	Environment struct {
		Name string `yaml:"environment_name"`
	} `yaml:"environment"`
	EnvironmentVariable struct {
		Name string `yaml:"environment_variable_name"`
	} `yaml:"environment_variable"`
	Commands map[string]string `yaml:"commands"`
	Packages []string          `yaml:"packages"`
}

func fromLegacy(data []byte) (*Project, error) {
	var old legacyFile
	if err := yaml.Unmarshal(data, &old); err != nil {
		return nil, err
	}
	p := &Project{
		Name:         old.Project.Name,
		Version:      old.Project.Version,
		Description:  old.Project.Description,
		Author:       old.Project.Author,
		License:      old.Project.License,
		Venv:         old.Environment.Name,
		EnvFile:      old.EnvironmentVariable.Name,
		Scripts:      map[string]string{},
		Dependencies: old.Packages,
		Migrated:     true,
	}
	if p.License == "ICS" { // typo shipped as the ppm3 default
		p.License = "ISC"
	}
	for name, cmd := range old.Commands {
		if name == "run" { // ppm3 used "run" as the default script
			name = "start"
		}
		p.Scripts[name] = cmd
	}
	return p, nil
}

// Path returns the absolute path of ppm.yml.
func (p *Project) Path() string { return filepath.Join(p.Root, FileName) }

// VenvDir returns the absolute path of the virtual environment.
func (p *Project) VenvDir() string { return p.abs(p.Venv) }

// EnvFilePath returns the absolute path of the .env file.
func (p *Project) EnvFilePath() string { return p.abs(p.EnvFile) }

func (p *Project) abs(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(p.Root, path)
}

// Save writes ppm.yml.
func (p *Project) Save() error {
	sortReqs(p.Dependencies)
	sortReqs(p.DevDependencies)

	var buf bytes.Buffer
	buf.WriteString("# Project file for ppm: https://github.com/rahulcodepython/PPM\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(p); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return os.WriteFile(p.Path(), buf.Bytes(), 0o644)
}

// ScriptNames returns the script names in sorted order.
func (p *Project) ScriptNames() []string {
	names := make([]string, 0, len(p.Scripts))
	for n := range p.Scripts {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// AllDependencies returns runtime dependencies, plus dev ones when dev is true.
func (p *Project) AllDependencies(dev bool) []string {
	all := append([]string{}, p.Dependencies...)
	if dev {
		all = append(all, p.DevDependencies...)
	}
	return all
}

// AddDependency records req, replacing any entry for the same package
// in either list.
func (p *Project) AddDependency(req string, dev bool) {
	key := Key(req)
	p.Dependencies = without(p.Dependencies, key)
	p.DevDependencies = without(p.DevDependencies, key)
	if dev {
		p.DevDependencies = append(p.DevDependencies, req)
	} else {
		p.Dependencies = append(p.Dependencies, req)
	}
}

// RemoveDependency drops the package from both lists and reports whether it was present.
func (p *Project) RemoveDependency(name string) bool {
	key := Key(name)
	before := len(p.Dependencies) + len(p.DevDependencies)
	p.Dependencies = without(p.Dependencies, key)
	p.DevDependencies = without(p.DevDependencies, key)
	return len(p.Dependencies)+len(p.DevDependencies) != before
}

// ReplaceDependency swaps the entry for the same package as req, keeping its list.
func (p *Project) ReplaceDependency(req string) {
	key := Key(req)
	for _, list := range [][]string{p.Dependencies, p.DevDependencies} {
		for i, r := range list {
			if Key(r) == key {
				list[i] = req
			}
		}
	}
}

func without(list []string, key string) []string {
	out := list[:0]
	for _, r := range list {
		if Key(r) != key {
			out = append(out, r)
		}
	}
	return out
}

func sortReqs(list []string) {
	sort.SliceStable(list, func(i, j int) bool { return Key(list[i]) < Key(list[j]) })
}
