# install.ps1 — bootstrap for the `skills` installer on Windows.
#
# Downloads the installer binary and runs it; all install logic lives in the
# Go binary, shared with macOS and Linux.
#
#   iwr -useb https://raw.githubusercontent.com/diegoclair/skills/main/install.ps1 | iex
#
# To pass arguments, download first:
#   iwr -useb .../install.ps1 -OutFile i.ps1; ./i.ps1 install confluence-docs
#
# Optional environment:
#   SKILLS_INSTALLER_VERSION   Pin a tag (default: latest installer-v* release)
#   SKILLS_REPO                Install from a fork

param([Parameter(ValueFromRemainingArguments = $true)] [string[]] $Args)

$ErrorActionPreference = 'Stop'

$Repo = if ($env:SKILLS_REPO) { $env:SKILLS_REPO } else { 'diegoclair/skills' }
$TagPrefix = 'installer-v'

$Version = $env:SKILLS_INSTALLER_VERSION
if (-not $Version) {
    # Every component is tagged separately here, so /releases/latest (one
    # pointer per repo) cannot be followed — filter the list by prefix.
    $releases = Invoke-RestMethod -UseBasicParsing `
        -Uri "https://api.github.com/repos/$Repo/releases?per_page=30" `
        -Headers @{ 'User-Agent' = 'skills-installer' }
    $Version = ($releases | Where-Object { $_.tag_name -like "$TagPrefix*" } | Select-Object -First 1).tag_name
    if (-not $Version) {
        throw "Could not resolve the latest $TagPrefix release. Set SKILLS_INSTALLER_VERSION."
    }
}

$Bin = Join-Path $env:TEMP "skills-installer-$([guid]::NewGuid()).exe"
$Url = "https://github.com/$Repo/releases/download/$Version/skills-windows-amd64.exe"

try {
    Invoke-WebRequest -UseBasicParsing -Uri $Url -OutFile $Bin
    # A bare `iex` pipe has no arguments and no tty to prompt on.
    if (-not $Args -or $Args.Count -eq 0) {
        & $Bin install --all
    } else {
        & $Bin @Args
    }
    exit $LASTEXITCODE
}
finally {
    Remove-Item -Path $Bin -Force -ErrorAction SilentlyContinue
}
