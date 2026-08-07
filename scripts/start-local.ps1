param(
    [string]$Address = "127.0.0.1:8080"
)
$ErrorActionPreference = "Stop"

& "$PSScriptRoot\build-wasm.ps1"
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

$parts = $Address -split ":", 2
$bind = $parts[0]
$port = if ($parts.Count -eq 2) { $parts[1] } else { "8080" }
Write-Output "Ruin Circuit WASM server: http://$Address/"
python -m http.server $port --bind $bind -d "$PSScriptRoot\..\web"
