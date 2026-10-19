//go:build windows

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// pathScript edits the user PATH in the registry. It keeps the value's
// REG_EXPAND_SZ type (so %USERPROFILE% style entries survive) and then
// touches a dummy variable so Windows tells open apps that PATH changed.
const pathScript = `
$ErrorActionPreference = 'Stop'
$dir = $env:PPM_DIR.TrimEnd('\')
$key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment', $true)
$old = [string]$key.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
$parts = @($old -split ';' | Where-Object { $_ -ne '' })
$has = @($parts | Where-Object { $_.TrimEnd('\') -ieq $dir }).Count -gt 0
if ($env:PPM_MODE -eq 'add') {
  if ($has) { 'unchanged'; exit 0 }
  $parts += $dir
} else {
  if (-not $has) { 'unchanged'; exit 0 }
  $parts = @($parts | Where-Object { $_.TrimEnd('\') -ine $dir })
}
$key.SetValue('Path', ($parts -join ';'), [Microsoft.Win32.RegistryValueKind]::ExpandString)
$key.Close()
[Environment]::SetEnvironmentVariable('PPM_PATH_REFRESH', '1', 'User')
[Environment]::SetEnvironmentVariable('PPM_PATH_REFRESH', $null, 'User')
'changed'
`

func editUserPath(mode, dir string) (bool, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", pathScript)
	cmd.Env = append(os.Environ(), "PPM_DIR="+dir, "PPM_MODE="+mode)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)) == "changed", nil
}

// addToPath adds dir to the user PATH environment variable.
func addToPath(dir string) (bool, error) {
	if onPath(dir) {
		return false, nil
	}
	return editUserPath("add", dir)
}

// removeFromPath removes dir from the user PATH environment variable.
func removeFromPath(dir string) error {
	_, err := editUserPath("remove", dir)
	return err
}
