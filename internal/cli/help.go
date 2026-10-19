package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rahulcodepython/PPM/internal/ui"
)

// usageTemplate is cobra's default with two additions: an "Arguments:"
// section (from the "args" annotation) and child commands listed under
// their parent via commandList.
const usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{with index .Annotations "args"}}

Arguments:
{{.}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{commandList .}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

func init() {
	cobra.AddTemplateFunc("commandList", commandList)
}

// commandList renders the command sections of a help page. Each command
// with children is followed by its indented child commands.
func commandList(c *cobra.Command) string {
	type entry struct{ name, short string }
	type section struct {
		title   string
		entries []entry
	}
	visible := func(x *cobra.Command) bool { return x.IsAvailableCommand() || x.Name() == "help" }
	collect := func(match func(*cobra.Command) bool) []entry {
		var out []entry
		for _, cmd := range c.Commands() {
			if !visible(cmd) || !match(cmd) {
				continue
			}
			out = append(out, entry{cmd.Name(), cmd.Short})
			for _, child := range cmd.Commands() {
				if child.IsAvailableCommand() {
					out = append(out, entry{"  " + cmd.Name() + " " + child.Name(), child.Short})
				}
			}
		}
		return out
	}

	var sections []section
	if len(c.Groups()) == 0 {
		sections = append(sections, section{"Available Commands:", collect(func(*cobra.Command) bool { return true })})
	} else {
		for _, g := range c.Groups() {
			id := g.ID
			sections = append(sections, section{g.Title, collect(func(x *cobra.Command) bool { return x.GroupID == id })})
		}
		sections = append(sections, section{"Additional Commands:", collect(func(x *cobra.Command) bool { return x.GroupID == "" })})
	}

	width := 0
	for _, s := range sections {
		for _, e := range s.entries {
			width = max(width, len(e.name))
		}
	}
	var b strings.Builder
	for _, s := range sections {
		if len(s.entries) == 0 {
			continue
		}
		b.WriteString("\n\n" + s.title)
		for _, e := range s.entries {
			b.WriteString("\n  " + ui.Pad(e.name, width) + "   " + e.short)
		}
	}
	return b.String()
}

// needArgs requires at least min positional arguments and, when they are
// missing, explains what to pass instead of cobra's generic message.
func needArgs(min int, what string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) >= min {
			return nil
		}
		return missingArgs(cmd, what)
	}
}

// exactArgs requires exactly n positional arguments.
func exactArgs(n int, what string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < n {
			return missingArgs(cmd, what)
		}
		if len(args) > n {
			return fmt.Errorf("too many arguments: %s takes %d, got %d\n\nUsage:\n  %s", calledPath(cmd), n, len(args), useLine(cmd))
		}
		return nil
	}
}

func missingArgs(cmd *cobra.Command, what string) error {
	msg := fmt.Sprintf("missing %s\n\nUsage:\n  %s", what, useLine(cmd))
	if ex := strings.TrimSpace(strings.SplitN(cmd.Example, "\n", 2)[0]); ex != "" {
		msg += "\n\nExample:\n  " + ex
	}
	return fmt.Errorf("%s\n\nRun `%s --help` for details.", msg, calledPath(cmd))
}

// calledPath is the command path using the name the user typed.
func calledPath(cmd *cobra.Command) string {
	if cmd.HasParent() && cmd.CalledAs() != "" {
		return cmd.Parent().CommandPath() + " " + cmd.CalledAs()
	}
	return cmd.CommandPath()
}

// useLine shows the name the user typed, so `ppm uninstall` errors mention
// `ppm uninstall` rather than `ppm remove`.
func useLine(cmd *cobra.Command) string {
	return strings.Replace(cmd.UseLine(), cmd.CommandPath(), calledPath(cmd), 1)
}

// argsHelp builds the "args" annotation shown under "Arguments:".
func argsHelp(lines ...string) map[string]string {
	return map[string]string{"args": "  " + strings.Join(lines, "\n  ")}
}
