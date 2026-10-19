package project

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestKey(t *testing.T) {
	cases := map[string]string{
		"requests":                  "requests",
		"Flask>=3.0":                "flask",
		"uvicorn[standard]>=0.30":   "uvicorn",
		"zope.interface==6":         "zope-interface",
		"my_pkg ; python_version<4": "my-pkg",
		"git+https://x/y.git":       "git+https://x/y.git",
		"./libs/local":              "./libs/local",
	}
	for in, want := range cases {
		if got := Key(in); got != want {
			t.Errorf("Key(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBumpVersion(t *testing.T) {
	cases := map[string]string{
		"requests>=2.0":          "requests>=2.32.3",
		"requests==2.0":          "requests==2.32.3",
		"uvicorn[standard]>=0.1": "uvicorn[standard]>=2.32.3",
		"requests>=2,<3":         "requests>=2,<3",
		"requests":               "requests",
	}
	for in, want := range cases {
		if got := BumpVersion(in, "2.32.3"); got != want {
			t.Errorf("BumpVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAddRemoveDependency(t *testing.T) {
	p := New(t.TempDir(), "x")
	p.AddDependency("requests>=1", false)
	p.AddDependency("pytest", true)
	p.AddDependency("Requests>=2", true) // moves to dev, replaces old spec
	if !reflect.DeepEqual(p.Dependencies, []string{}) || !reflect.DeepEqual(p.DevDependencies, []string{"pytest", "Requests>=2"}) {
		t.Fatalf("got deps=%v dev=%v", p.Dependencies, p.DevDependencies)
	}
	if !p.RemoveDependency("requests==9") || p.RemoveDependency("nope") {
		t.Fatal("RemoveDependency reported wrong result")
	}
	if !reflect.DeepEqual(p.DevDependencies, []string{"pytest"}) {
		t.Fatalf("dev = %v", p.DevDependencies)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := New(dir, "demo")
	p.Scripts["start"] = "python src/main.py"
	p.AddDependency("requests>=2", false)
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Find(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "demo" || got.Root != dir || got.Scripts["start"] != "python src/main.py" || got.Dependencies[0] != "requests>=2" {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestLoadLegacy(t *testing.T) {
	dir := t.TempDir()
	legacy := `project:
  name: old
  version: 1.3.1
  description: ''
  main_file: main.py
  git_repository: ''
  author: me
  license: ICS
python_version: 3.12.1
environment:
  environment_name: .venv
  virtual_environment_activate_path: .venv/bin/activate
environment_variable:
  environment_variable_name: .env
  environment_variable_path: .env
commands:
  run: python src/main.py
  hello: echo hi
packages:
  - requests==2.32.3
`
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Migrated || p.Name != "old" || p.License != "ISC" || p.Scripts["start"] != "python src/main.py" ||
		p.Scripts["hello"] != "echo hi" || p.Dependencies[0] != "requests==2.32.3" || p.Venv != ".venv" {
		t.Fatalf("legacy conversion wrong: %+v", p)
	}
}
