$ErrorActionPreference = "Stop"

$version = "2026.9.1"
$expectedSHA256 = "2837888cc0f5d58f15b6dc478376de90b4d3ba5241c7947455d1e0a0df429712"
$root = Split-Path -Parent $PSScriptRoot
$output = Join-Path $root "build\tools\cloudflared.exe"
$outputDir = Split-Path -Parent $output

New-Item -ItemType Directory -Force -Path $outputDir | Out-Null

function Get-SHA256Lower([string] $path) {
    return (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
}

if (Test-Path -LiteralPath $output -PathType Leaf) {
    if ((Get-SHA256Lower $output) -eq $expectedSHA256) {
        Write-Host "cloudflared $version is already installed: $output"
        exit 0
    }
}

$url = "https://github.com/cloudflare/cloudflared/releases/download/$version/cloudflared-windows-amd64.exe"
$temp = "$output.download"

try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Write-Host "Downloading cloudflared $version..."
    Invoke-WebRequest -Uri $url -OutFile $temp -UseBasicParsing
    $actualSHA256 = Get-SHA256Lower $temp
    if ($actualSHA256 -ne $expectedSHA256) {
        throw "cloudflared checksum mismatch: expected $expectedSHA256, got $actualSHA256"
    }
    Move-Item -LiteralPath $temp -Destination $output -Force
}
finally {
    if (Test-Path -LiteralPath $temp) {
        Remove-Item -LiteralPath $temp -Force
    }
}

& $output --version
if ($LASTEXITCODE -ne 0) {
    throw "cloudflared --version failed with exit code $LASTEXITCODE"
}

Write-Host "cloudflared installed: $output"
