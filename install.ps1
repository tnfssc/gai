<#
.SYNOPSIS
Downloads and installs the latest release of gai CLI from GitHub.

.DESCRIPTION
This script downloads the appropriate gai CLI binary for your operating system (Windows, Linux, macOS) and architecture (amd64, arm64) from the latest GitHub release.
It then moves the binary to a directory in your PATH environment variable, making the 'gai' command available system-wide.

.NOTES
Requires PowerShell 7 or later.
#>

# Script Parameters (can be modified if needed)
$repoOwner = "tnfssc"
$repoName = "gai"
$binaryNamePrefix = "gai"
$installDirWindows = "$env:LOCALAPPDATA\Microsoft\WindowsApps"
$installDirLinuxMac = "/usr/local/bin"

# Determine Operating System
$osPlatform = Get-WmiObject -Class Win32_OperatingSystem | Select-Object Caption
if ($osPlatform.Caption -like "*Windows*") {
  $os = "windows"
} elseif ($osPlatform.Caption -like "*macOS*" -or $osPlatform.Caption -like "*Darwin*") {
  $os = "darwin"
} else {
  $os = "linux"
}

# Determine Architecture
$architecture = Get-ProcessorBits
if ($architecture -eq 64) {
  $arch = "amd64"
} elseif ($architecture -eq 32) {
  Write-Error "32-bit architecture is not supported for gai CLI."
  return
} else {
  # Assuming ARM64 if not AMD64/x64 or x86
  if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64" -or $env:PROCESSOR_IDENTIFIER -like "*ARM64*") {
    $arch = "arm64"
  } else {
    $arch = "amd64" # Default to amd64 if detection fails on 64-bit OS
  }
}


# Construct API URL
$apiUrl = "https://api.github.com/repos/$repoOwner/$repoName/releases/latest"

# Get latest release info from GitHub API
Write-Host "Fetching latest release information from GitHub..."
try {
  $response = Invoke-WebRequest -Uri $apiUrl -UseBasicParsing
  $releaseInfo = ConvertFrom-Json $response.Content
}
catch {
  Write-Error "Failed to fetch release information from GitHub API: $_"
  return
}

# Determine binary name based on OS and architecture
if ($os -eq "windows") {
  $binaryFileName = "$binaryNamePrefix-$os-$arch.exe"
  $installDirectory = $installDirWindows
} else {
  $binaryFileName = "$binaryNamePrefix-$os-$arch"
  $installDirectory = $installDirLinuxMac
}

# Find the correct asset
Write-Host "Finding binary for $($os) $($arch)..."
$downloadUrl = ""
foreach ($asset in $releaseInfo.assets) {
  if ($asset.name -eq $binaryFileName) {
    $downloadUrl = $asset.browser_download_url
    break
  }
}

if (-not $downloadUrl) {
  Write-Error "No binary found for $($os) $($arch) in the latest release."
  Write-Error "Looking for binary name: $($binaryFileName)"
  Write-Error "Available assets are:"
  $releaseInfo.assets | ForEach-Object {$_.name}
  return
}

Write-Host "Download URL found: $($downloadUrl)"

# Download the binary
$tempBinaryPath = Join-Path -Path ([System.IO.Path]::GetTempPath()) -ChildPath $binaryFileName
Write-Host "Downloading binary to $($tempBinaryPath)..."
try {
  Invoke-WebRequest -Uri $downloadUrl -OutFile $tempBinaryPath
  Write-Host "Binary downloaded successfully."
}
catch {
  Write-Error "Failed to download binary: $_"
  return
}

# Create install directory if it doesn't exist
if (!(Test-Path -Path $installDirectory -PathType Container)) {
  Write-Host "Creating install directory: $($installDirectory)..."
  try {
    New-Item -ItemType Directory -Path $installDirectory -Force | Out-Null
    Write-Host "Install directory created."
  }
  catch {
    Write-Error "Failed to create install directory: $_"
    return
  }
}

# Move binary to install directory and rename to 'gai' (without extension if windows)
if ($os -eq "windows") {
  $finalBinaryPath = Join-Path -Path $installDirectory -ChildPath "gai.exe"
} else {
  $finalBinaryPath = Join-Path -Path $installDirectory -ChildPath "gai"
}

Write-Host "Moving binary to $($finalBinaryPath)..."
try {
  Move-Item -Path $tempBinaryPath -Destination $finalBinaryPath -Force
  Write-Host "Binary moved to install directory."
}
catch {
  Write-Error "Failed to move binary to install directory: $_"
  return
}

# Set execute permissions for Linux and macOS
if ($os -ne "windows") {
  Write-Host "Setting execute permissions..."
  try {
    chmod +x $finalBinaryPath
    Write-Host "Execute permissions set."
  }
  catch {
    Write-Warning "Failed to set execute permissions. You may need to manually set execute permissions for $($finalBinaryPath): $_"
  }
}

Write-Host "gai CLI installed successfully to $($installDirectory)"
Write-Host "Make sure '$installDirectory' is in your PATH environment variable."
if ($os -eq "windows") {
  Write-Host "It is automatically added for Windows if using $installDirWindows."
} else {
  Write-Host "You might need to add '$installDirectory' to your PATH manually if it's not already included."
}
Write-Host "You can now run 'gai' from your terminal."

# TESTING
# docker run --rm -v "$(pwd):/app" ubuntu:latest bash -c "apt-get update && apt-get install -y wget apt-transport-https software-properties-common && source /etc/os-release && wget -q https://packages.microsoft.com/config/ubuntu/\$VERSION_ID/packages-microsoft-prod.deb && dpkg -i packages-microsoft-prod.deb && rm packages-microsoft-prod.deb && apt-get update && apt-get install -y powershell curl jq && cd /app && pwsh ./install.ps1 && ls /usr/local/bin/gai"
