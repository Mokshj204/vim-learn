$ErrorActionPreference = "Stop"

$Repo = if ($env:VIMLEARN_REPO) { $env:VIMLEARN_REPO } else { "Mokshj204/vim-learn" }

$Arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "ARM64" { "arm64" }
    default { "amd64" }
}

$Asset = "vimlearn-windows-$Arch.exe"
$Url = "https://github.com/$Repo/releases/latest/download/$Asset"

$InstallDir = if ($env:VIMLEARN_INSTALL_DIR) { $env:VIMLEARN_INSTALL_DIR } else { Join-Path $env:USERPROFILE ".local\bin" }
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null

$TargetPath = Join-Path $InstallDir "vimlearn.exe"
Write-Host "Downloading $Asset from $Url..."
Invoke-WebRequest -Uri $Url -OutFile $TargetPath -UseBasicParsing

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not $UserPath) { $UserPath = "" }
$Parts = $UserPath -split ";" | Where-Object { $_ -ne "" }
if ($Parts -notcontains $InstallDir) {
    $NewUserPath = ($Parts + $InstallDir) -join ";"
    [Environment]::SetEnvironmentVariable("Path", $NewUserPath, "User")
    $env:Path = "$env:Path;$InstallDir"
    Write-Host "Added $InstallDir to your User PATH."
}

Write-Host "Installed vimlearn to $TargetPath"
Write-Host "Run 'vimlearn' from any directory to start!"
