@REM Written by LLM, untested

@echo off
setlocal

REM Define base URL for the latest release assets on GitHub
set "URLBASE=https://github.com/tnfssc/gai/releases/latest/download/"

REM Determine which Windows binary to download by checking the system architecture
if /I "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
  set "FILE=gai-windows-amd64.exe"
) else if /I "%PROCESSOR_ARCHITECTURE%"=="ARM64" (
  set "FILE=gai-windows-arm64.exe"
) else (
  echo Unsupported architecture: %PROCESSOR_ARCHITECTURE%
  pause
  exit /b 1
)

REM Build the full download URL
set "DOWNLOAD_URL=%URLBASE%%FILE%"
echo Downloading %FILE% from %DOWNLOAD_URL%

REM Download the binary using curl.
REM Note: Windows 10 and later include curl by default.
curl -L -o "%TEMP%\%FILE%" "%DOWNLOAD_URL%"
if errorlevel 1 (
  echo Download failed. Exiting.
  pause
  exit /b 1
)

REM Choose an installation directory. Here we use a bin directory in the user's profile.
set "INSTALL_DIR=%USERPROFILE%\bin"
if not exist "%INSTALL_DIR%" (
  mkdir "%INSTALL_DIR%"
)

echo Installing %FILE% to %INSTALL_DIR%

REM Move (or rename) the downloaded binary to a standard name (gai.exe) in the installation directory.
move /Y "%TEMP%\%FILE%" "%INSTALL_DIR%\gai.exe"
if errorlevel 1 (
  echo Installation failed.
  pause
  exit /b 1
)

echo Installation complete.
echo.
echo To run the program from any command prompt, ensure that "%INSTALL_DIR%" is in your system PATH.
echo You can add it temporarily with:
echo   set PATH=%%PATH%%;%INSTALL_DIR%
echo or permanently via system settings.
pause
endlocal
