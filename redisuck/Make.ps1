$CC = "go build"
$CFLAGS = ""

function Build {
    & $CC -o redisuck redisuck.go
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Build failed"
        exit $LASTEXITCODE
    }
    Write-Output "Build succeeded"
}

function CleanBuild {
    Remove-Item -Force -ErrorAction SilentlyContinue redisuck
    Write-Output "Clean succeeded"
}

param (
    [switch]$Clean
)

if ($Clean) {
    CleanBuild
} else {
    Build
}