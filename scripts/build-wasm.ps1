$ErrorActionPreference = "Stop"
$previousGOOS = $env:GOOS
$previousGOARCH = $env:GOARCH
try {
    $env:GOOS = "js"
    $env:GOARCH = "wasm"
    New-Item -ItemType Directory -Path ".\web" -Force | Out-Null
    go build -buildvcs=false -o ".\web\game.wasm" .
    Copy-Item "$(go env GOROOT)\lib\wasm\wasm_exec.js" ".\web\wasm_exec.js" -Force
    $versionMatch = Select-String -Path ".\main.go" -Pattern 'version\s*=\s*"([^"]+)"'
    $buildVersion = $versionMatch.Matches[0].Groups[1].Value
    $indexPath = ".\web\index.html"
    $index = Get-Content -LiteralPath $indexPath -Raw
    $cacheVersion = $buildVersion.TrimStart('v')
    $index = [regex]::Replace($index, 'game\.wasm\?v=[^"]+', "game.wasm?v=$cacheVersion")
    Set-Content -LiteralPath $indexPath -Value $index -NoNewline
    Write-Output "WASM build ready in web\"
} finally {
    $env:GOOS = $previousGOOS
    $env:GOARCH = $previousGOARCH
}
