# Install clouds from GitHub Releases (Windows). Override with CLOUDS_REPO,
# CLOUDS_VERSION, CLOUDS_BIN. Run:
#
#   irm https://raw.githubusercontent.com/OWNER/REPO/main/scripts/install.ps1 | iex
$ErrorActionPreference = 'Stop'

$repo = if ($env:CLOUDS_REPO) { $env:CLOUDS_REPO } else { 'GITHUB_REPO/clouds' }
$version = if ($env:CLOUDS_VERSION) { $env:CLOUDS_VERSION } else { 'latest' }
$dir = if ($env:CLOUDS_BIN) { $env:CLOUDS_BIN } else { Join-Path $env:LOCALAPPDATA 'Programs\clouds' }

# curl.exe ships with Windows 10+ and negotiates modern TLS out of the box,
# where Invoke-WebRequest on Windows PowerShell 5.1 needs TLS 1.2 forced and
# still fails on some setups (the release download must work on a fresh box).
if (-not (Get-Command curl.exe -ErrorAction SilentlyContinue)) {
  throw "curl.exe not found - it ships with Windows 10 and later"
}

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  'AMD64' { 'amd64' }
  'ARM64' { 'arm64' }
  default { throw "unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}
$base = "cloud-windows-$arch"
$rel = if ($version -eq 'latest') { 'releases/latest/download' } else { "releases/download/$version" }
$url = "https://github.com/$repo/$rel/$base.zip"

New-Item -ItemType Directory -Force -Path $dir | Out-Null
$zip = Join-Path $env:TEMP "$base.zip"
Write-Host "downloading $url"
curl.exe -fL --progress-bar -o $zip $url
if ($LASTEXITCODE -ne 0) { throw "failed to download $url" }

# Checksums when published. Only "release ships no SHA256SUMS" (HTTP 404)
# skips verification; a failed request or a sums file without an entry for
# this artifact is fatal, never a silent pass.
$sums = Join-Path $env:TEMP 'clouds-SHA256SUMS'
$code = curl.exe -Ls -o $sums -w '%{http_code}' "https://github.com/$repo/$rel/SHA256SUMS"
if ($code -eq '200') {
  $want = (Select-String -Path $sums -Pattern "^([0-9a-f]{64})  $base.zip$" |
    ForEach-Object { $_.Matches[0].Groups[1].Value } | Select-Object -First 1)
  if (-not $want) { throw "SHA256SUMS has no entry for $base.zip - refusing to install" }
  $got = (Get-FileHash $zip -Algorithm SHA256).Hash.ToLower()
  if ($got -ne $want) { throw "checksum mismatch: expected $want, got $got" }
  Write-Host 'checksum ok'
} elseif ($code -eq '404') {
  Write-Host "note: no SHA256SUMS published for this release, skipping checksum"
} else {
  throw "downloading SHA256SUMS failed (HTTP $code)"
}

Expand-Archive -Path $zip -DestinationPath $dir -Force
Copy-Item (Join-Path $dir "$base.exe") (Join-Path $dir 'clouds.exe') -Force
Remove-Item (Join-Path $dir "$base.exe") -Force

& (Join-Path $dir 'clouds.exe') --version
Write-Host "installed -> $dir\clouds.exe"
if (($env:Path -split ';') -notcontains $dir) {
  Write-Host "add to PATH (one time):  [Environment]::SetEnvironmentVariable('Path', `$env:Path + `";$dir`", 'User')"
}
