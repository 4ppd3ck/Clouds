@echo off
rem Build the release artifacts with the shared shell script.
where sh >nul 2>nul
if errorlevel 1 (
    echo Git Bash "sh" is required - install Git for Windows, or run release.sh from a POSIX shell.
    exit /b 1
)
sh "%~dp0release.sh" %*
