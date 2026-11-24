# Install the CLI from the Windows release ZIP this script was extracted from.
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$binary = Get-ChildItem -Path $root -Filter 'cloud-windows-*.exe' -File | Select-Object -First 1
if (-not $binary) { throw 'cloud-windows-*.exe was not found beside this installer' }

$dir = if ($env:CLOUDS_BIN) { $env:CLOUDS_BIN } else { Join-Path $env:LOCALAPPDATA 'Programs\clouds' }
New-Item -ItemType Directory -Force -Path $dir | Out-Null
Copy-Item $binary.FullName (Join-Path $dir 'clouds.exe') -Force

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$pathEntries = @($userPath -split ';' | Where-Object { $_.Trim() })
$normalizedDir = [IO.Path]::GetFullPath($dir).TrimEnd('\')
$alreadyPresent = $false
foreach ($entry in $pathEntries) {
  try {
    if ([IO.Path]::GetFullPath($entry.Trim()).TrimEnd('\').Equals(
        $normalizedDir, [StringComparison]::OrdinalIgnoreCase)) {
      $alreadyPresent = $true
      break
    }
  } catch {
    # Preserve unusual existing PATH entries.
  }
}
if (-not $alreadyPresent) {
  [Environment]::SetEnvironmentVariable('Path', (@($pathEntries) + $dir) -join ';', 'User')
  Write-Host "added $dir to your user PATH"
} else {
  Write-Host "$dir is already on your user PATH"
}
if (($env:Path -split ';' | ForEach-Object { $_.TrimEnd('\') }) -notcontains $normalizedDir) {
  $env:Path = "$dir;$env:Path"
}

& (Join-Path $dir 'clouds.exe') --version
Write-Host "installed -> $dir\clouds.exe"
Write-Host 'Open a new terminal to use clouds from any folder.'
