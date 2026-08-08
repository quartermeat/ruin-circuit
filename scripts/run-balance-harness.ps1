$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$outputRoot = Join-Path $repoRoot "artifacts\balance-run"
$framesRoot = Join-Path $outputRoot "frames"
$ffmpegPath = "C:\Program Files\Bitwig Studio\bin\ffmpeg.exe"

if (Test-Path -LiteralPath $outputRoot) {
    Remove-Item -LiteralPath $outputRoot -Recurse -Force
}
New-Item -ItemType Directory -Path $framesRoot -Force | Out-Null

$env:BALANCE_OUTPUT = $outputRoot
go run . -balance-harness

if (-not (Test-Path -LiteralPath $ffmpegPath)) {
    throw "FFmpeg was not found at $ffmpegPath"
}

& $ffmpegPath -y -framerate 15 -i (Join-Path $framesRoot "frame-%05d.png") -c:v h264_mf -b:v 5M -pix_fmt yuv420p -movflags +faststart (Join-Path $outputRoot "balance-run-tiktok.mp4")
if ($LASTEXITCODE -ne 0) {
    throw "FFmpeg failed with exit code $LASTEXITCODE"
}

Write-Output "Balance replay ready: $outputRoot\balance-run-tiktok.mp4"
