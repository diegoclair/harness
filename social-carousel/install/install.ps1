# install.ps1 — installs the social-carousel skill on Windows.
#
# Kept at this URL for the published one-liner and for `social-carousel update`.
# The work is done by the `skills` installer binary; this only forwards the
# skill name to it.
#
#   iwr -useb https://raw.githubusercontent.com/diegoclair/skills/main/social-carousel/install/install.ps1 | iex

$ErrorActionPreference = 'Stop'

$Repo = if ($env:SKILL_REPO) { $env:SKILL_REPO } else { 'diegoclair/skills' }
$RootUrl = "https://raw.githubusercontent.com/$Repo/main/install.ps1"

$Bootstrap = Join-Path $env:TEMP "skills-bootstrap-$([guid]::NewGuid()).ps1"
try {
    Invoke-WebRequest -UseBasicParsing -Uri $RootUrl -OutFile $Bootstrap
    & $Bootstrap install social-carousel
    exit $LASTEXITCODE
}
finally {
    Remove-Item -Path $Bootstrap -Force -ErrorAction SilentlyContinue
}
