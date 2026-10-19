// Command gendocs writes man pages and shell completions from the ppm
// command tree into ./man and ./completions. Releases run it automatically.
// Run: go run ./tools/gendocs
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/rahulcodepython/PPM/internal/cli"
)

func main() {
	root := cli.NewRootCmd()
	root.DisableAutoGenTag = true // keep output stable between runs

	must(os.RemoveAll("man"))
	must(os.MkdirAll("man", 0o755))
	must(doc.GenManTree(root, &doc.GenManHeader{Title: "PPM", Section: "1", Source: "ppm", Manual: "ppm manual"}, "man"))

	must(os.RemoveAll("completions"))
	must(os.MkdirAll("completions", 0o755))
	gens := map[string]func(*cobra.Command, string) error{
		"ppm.bash": func(c *cobra.Command, f string) error { return c.GenBashCompletionFileV2(f, true) },
		"ppm.zsh":  func(c *cobra.Command, f string) error { return c.GenZshCompletionFile(f) },
		"ppm.fish": func(c *cobra.Command, f string) error { return c.GenFishCompletionFile(f, true) },
		"ppm.ps1":  func(c *cobra.Command, f string) error { return c.GenPowerShellCompletionFileWithDesc(f) },
	}
	for name, gen := range gens {
		must(gen(root, filepath.Join("completions", name)))
	}
	log.Println("wrote man, completions")
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
