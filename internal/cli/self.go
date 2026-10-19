package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rahulcodepython/PPM/internal/ui"
)

func newSelfCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "self",
		GroupID: "tool",
		Short:   "Install, update or uninstall ppm itself",
		Example: `  ./ppm self install     # copy this binary to the standard location and add it to PATH
  ppm self update        # update to the latest release
  ppm self uninstall     # remove ppm and its PATH entry`,
	}
	cmd.AddCommand(newSelfInstallCmd(), newSelfUpdateCmd(), newSelfUninstallCmd())
	return cmd
}

// newSelfUpdateAlias keeps `ppm self-update` working.
func newSelfUpdateAlias() *cobra.Command {
	c := newSelfUpdateCmd()
	c.Use = "self-update"
	c.Hidden = true
	return c
}

// defaultInstallDir is where ppm installs itself:
//
//	Windows:        %LOCALAPPDATA%\Programs\ppm
//	Linux/macOS:    ~/.local/bin (or /usr/local/bin when run as root)
func defaultInstallDir() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			return "", errors.New("LOCALAPPDATA is not set; pass --dir")
		}
		return filepath.Join(base, "Programs", "ppm"), nil
	}
	if os.Geteuid() == 0 {
		return "/usr/local/bin", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "bin"), nil
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "ppm.exe"
	}
	return "ppm"
}

func newSelfInstallCmd() *cobra.Command {
	var dir string
	var noPath bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Copy this ppm binary to the standard location and add it to PATH",
		Long: `Copy the running ppm binary into the standard per-user location and
make sure that folder is on PATH, so ppm works from any terminal:

  Linux/macOS  ~/.local/bin   (/usr/local/bin when run as root)
               PATH is added to ~/.profile, ~/.bashrc, ~/.zshrc and fish config
  Windows      %LOCALAPPDATA%\Programs\ppm
               PATH is added to your user environment variables

Running it again is safe. The install scripts and "make install" use it.`,
		Example: `  ./ppm self install
  ./ppm self install --dir /opt/bin --no-path`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				d, err := defaultInstallDir()
				if err != nil {
					return err
				}
				dir = d
			}
			dir, err := filepath.Abs(dir)
			if err != nil {
				return err
			}
			src, err := os.Executable()
			if err != nil {
				return err
			}
			if src, err = filepath.EvalSymlinks(src); err != nil {
				return err
			}
			dst := filepath.Join(dir, binaryName())
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			if same, _ := sameFile(src, dst); same {
				ui.Success("ppm is already installed at %s", dst)
			} else {
				if err := copyExecutable(src, dst); err != nil {
					return fmt.Errorf("installing to %s: %w", dst, err)
				}
				ui.Success("Installed ppm %s to %s", resolvedVersion(), dst)
			}
			if noPath {
				return nil
			}
			changed, err := addToPath(dir)
			if err != nil {
				return fmt.Errorf("adding %s to PATH: %w", dir, err)
			}
			if changed {
				ui.Success("Added %s to PATH", dir)
				fmt.Println("  Open a new terminal, then run: ppm --version")
			} else if onPath(dir) {
				ui.Success("%s is already on PATH. Run: ppm --version", dir)
			} else {
				ui.Success("PATH is already set up for %s; open a new terminal to use it", dir)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "install into this folder instead of the default")
	cmd.Flags().BoolVar(&noPath, "no-path", false, "do not change PATH")
	return cmd
}

func newSelfUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the ppm binary and the PATH entry added by `ppm self install`",
		Long: `Remove the running ppm binary and the PATH entry that "ppm self install"
added. Projects, their .venv folders and ppm.yml files are not touched.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			if exe, err = filepath.EvalSymlinks(exe); err != nil {
				return err
			}
			if err := removeFromPath(filepath.Dir(exe)); err != nil {
				ui.Warn("could not clean PATH: %v", err)
			}
			if runtime.GOOS == "windows" {
				// A running .exe cannot be deleted; rename it so the folder can be removed later.
				old := exe + ".old"
				os.Remove(old)
				if err := os.Rename(exe, old); err != nil {
					return err
				}
				ui.Success("Uninstalled. Delete %s to finish.", filepath.Dir(exe))
				return nil
			}
			if err := os.Remove(exe); err != nil {
				return fmt.Errorf("removing %s: %w (try with sudo)", exe, err)
			}
			ui.Success("Removed %s", exe)
			return nil
		},
	}
}

func sameFile(a, b string) (bool, error) {
	ai, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return os.SameFile(ai, bi), nil
}

// copyExecutable writes src to dst through a temp file and rename, so a
// running copy at dst is replaced safely.
func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	data, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		return replaceExecutable(dst, data)
	}
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

// onPath reports whether dir is in the current PATH.
func onPath(dir string) bool {
	want := filepath.Clean(dir)
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == "" {
			continue
		}
		got := filepath.Clean(p)
		if got == want || (runtime.GOOS == "windows" && strings.EqualFold(got, want)) {
			return true
		}
	}
	return false
}
