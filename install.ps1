# install.ps1 — bootstrap for the `harness` installer (Windows).
#
#   iwr -useb https://raw.githubusercontent.com/diegoclair/harness/main/install.ps1 | iex
#
# To choose artifacts, download this script first and pass arguments to it:
#   .\install.ps1 install dev-loop unbiased-reviewer

$ErrorActionPreference = "Stop"

$repo      = if ($env:HARNESS_REPO) { $env:HARNESS_REPO } else { "diegoclair/harness" }
$tagPrefix = "harness-v"
$asset     = "harness"

$version = $env:HARNESS_INSTALLER_VERSION
if (-not $version) {
    # Tags are per component, so /releases/latest cannot be followed.
    $releases = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases?per_page=30"
    $version = ($releases | Where-Object { $_.tag_name -like "$tagPrefix*" } | Select-Object -First 1).tag_name
    if (-not $version) { throw "could not resolve the latest $tagPrefix release; set HARNESS_INSTALLER_VERSION" }
}

$url = "https://github.com/$repo/releases/download/$version/$asset-windows-amd64.exe"
$bin = Join-Path $env:TEMP "harness-installer.exe"

Invoke-WebRequest -Uri $url -OutFile $bin -UseBasicParsing

if ($args.Count -eq 0) {
    & $bin list
    Write-Host ""
    Write-Host "Pick what you want, e.g.:  .\install.ps1 install dev-loop"
    exit 2
}
& $bin @args
exit $LASTEXITCODE
