# Install the latest ppm release on Windows (PowerShell):
#   irm https://raw.githubusercontent.com/rahulcodepython/PPM/main/install.ps1 | iex
#
# Installs to %LOCALAPPDATA%\Programs\ppm and adds it to your user PATH.
# Options (environment variables): PPM_VERSION=v2.0.0, PPM_INSTALL_DIR=C:\tools\ppm
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"  # much faster Invoke-WebRequest on PowerShell 5

$repo = "rahulcodepython/PPM"
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
$base = if ($env:PPM_VERSION) { "https://github.com/$repo/releases/download/$($env:PPM_VERSION)" } else { "https://github.com/$repo/releases/latest/download" }
$archive = "ppm_windows_$arch.zip"

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("ppm-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "Downloading $archive ..."
    Invoke-WebRequest "$base/$archive" -OutFile "$tmp\$archive" -UseBasicParsing
    Invoke-WebRequest "$base/checksums.txt" -OutFile "$tmp\checksums.txt" -UseBasicParsing

    $expected = (Select-String -Path "$tmp\checksums.txt" -Pattern " $archive$").Line.Split(" ")[0]
    $actual = (Get-FileHash "$tmp\$archive" -Algorithm SHA256).Hash.ToLower()
    if ($expected -ne $actual) { throw "checksum mismatch for $archive" }

    Expand-Archive "$tmp\$archive" -DestinationPath $tmp -Force

    # ppm copies itself to the install folder and adds it to the user PATH.
    if ($env:PPM_INSTALL_DIR) {
        & "$tmp\ppm.exe" self install --dir $env:PPM_INSTALL_DIR
    } else {
        & "$tmp\ppm.exe" self install
    }
    if ($LASTEXITCODE -ne 0) { throw "ppm self install failed" }

    # Make ppm usable in this PowerShell window too.
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    $env:Path = "$machinePath;$userPath"
} finally {
    Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
