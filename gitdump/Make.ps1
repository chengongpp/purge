$CC = "go build"
$CFLAGS = ""

function Build-GitDump {
    & $CC -o gitdump gitdump.go
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Build failed"
        exit $LASTEXITCODE
    }
    Write-Output "Build succeeded"
}

function Clean-GitDump {
    Remove-Item -Force -ErrorAction SilentlyContinue gitdump
    Write-Output "Clean succeeded"
}

param (
    [switch]$Clean
)

if ($Clean) {
    Clean-GitDump
} else {
    Build-GitDump
}