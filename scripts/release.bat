@echo off
rem Cross-compile release binaries for every supported OS/arch into release\.
rem Usage: scripts\release.bat [version]   e.g. scripts\release.bat v1.2.3
rem Output: release\cloud-<os>-<arch>[.exe], compressed archives per target,
rem macOS .app bundles (cloud-darwin-*-app.zip), and SHA256SUMS. Commit +
rem build date are always stamped into the binaries.
cd /d "%~dp0.."

rem Version: explicit arg ^> newest v* git tag ^> none (dev default 0.1.0).
rem Note: inside for /f, "=" must be escaped as "^=" or cmd splits the argument.
set "VERSION=%~1"
if "%VERSION%"=="" (
  for /f "delims=" %%v in ('git describe --tags --abbrev^=0 --match v* 2^>nul') do set "VERSION=%%v"
)
set "COMMIT="
for /f "delims=" %%c in ('git rev-parse --short HEAD 2^>nul') do set "COMMIT=%%c"
set "BUILD_DATE="
if defined COMMIT for /f "delims=" %%d in ('git log -1 --format^=%%cs 2^>nul') do set "BUILD_DATE=%%d"

set "LDFLAGS=-s -w"
if defined VERSION set "LDFLAGS=%LDFLAGS% -X clouds/ui.Version=%VERSION%"
if defined COMMIT set "LDFLAGS=%LDFLAGS% -X clouds/ui.Commit=%COMMIT% -X clouds/ui.Date=%BUILD_DATE%"

if exist release rmdir /s /q release
mkdir release

echo generating desktop assets (icon set, desktop entry)
go run ./packaging icons packaging\icons
if errorlevel 1 goto :fail
rem Windows icon resources: clouds.ico plus a COFF .rsrc object per architecture.
rem clouds_windows_<arch>.syso must sit in the main package directory; the Go
rem linker links it into that architecture's windows build automatically.
echo generating Windows icon resources (clouds.ico + clouds_windows_*.syso)
go run ./packaging ico packaging\icons
if errorlevel 1 goto :fail
go run ./packaging syso -arch amd64
if errorlevel 1 goto :fail
go run ./packaging syso -arch arm64
if errorlevel 1 goto :fail
set "APPVERSION=%VERSION%"
if not defined APPVERSION set "APPVERSION=0.1.0"

set CGO_ENABLED=0
call :build darwin amd64  || goto :fail
call :build darwin arm64  || goto :fail
call :build linux  amd64  || goto :fail
call :build linux  arm64  || goto :fail
call :build linux  arm    || goto :fail
call :build windows amd64 || goto :fail
call :build windows arm64 || goto :fail

rem Compressed archives for distribution (Go binaries shrink ~65 percent).
call :archive darwin amd64  || goto :fail
call :archive darwin arm64  || goto :fail
call :archive linux  amd64  || goto :fail
call :archive linux  arm64  || goto :fail
call :archive linux  arm    || goto :fail
call :archive windows amd64 || goto :fail
call :archive windows arm64 || goto :fail

rem SHA256 checksums for the release folder (LF line endings, so `sha256sum -c`
rem works on macOS/Linux too).
set "SHASUMS=release\SHA256SUMS"
if exist %SHASUMS% del %SHASUMS%
powershell -NoProfile -Command "Get-ChildItem 'release\cloud-*' | Sort-Object Name | ForEach-Object { $h = (Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLower(); [IO.File]::AppendAllText('release\SHA256SUMS', \"$h  $($_.Name)`n\") }"

echo.
echo release folder ready:
dir release
exit /b 0

:build
set "GOOS=%~1"
set "GOARCH=%~2"
set "GOARM="
if "%~1"=="linux" if "%~2"=="arm" set "GOARM=7"
set "ext="
if "%~1"=="windows" set "ext=.exe"
echo building cloud-%~1-%~2%ext%
go build -trimpath -ldflags "%LDFLAGS%" -o "release\cloud-%~1-%~2%ext%" .
if errorlevel 1 exit /b 1
exit /b 0

:archive
rem The build loop leaves GOOS/GOARCH/GOARM set; clear them so host tools run
rem with the host toolchain (go run ./packaging otherwise cross-compiles and
rem then fails to execute its own output).
set "GOOS="
set "GOARCH="
set "GOARM="
set "base=cloud-%~1-%~2"
if "%~1"=="windows" (
  powershell -NoProfile -Command "Compress-Archive -Path 'release\%base%.exe' -DestinationPath 'release\%base%.zip' -Force"
) else if "%~1"=="darwin" (
  tar -czf release\%base%.tar.gz -C release %base%
  if errorlevel 1 exit /b 1
  echo bundling %base%-app.zip
  go run ./packaging app -bin release\%base% -arch %~2 -version %APPVERSION% -icons packaging\icons -out release\%base%-app.zip
) else (
  rem linux: ship the desktop entry + icon for shell integration
  rem (absolute -C paths: tar's -C is cumulative, relative would nest)
  tar -czf release\%base%.tar.gz -C "%~dp0..\release" %base% -C "%~dp0..\packaging\icons" clouds.desktop clouds.svg
)
if errorlevel 1 exit /b 1
exit /b 0

:fail
echo release build failed
exit /b 1
