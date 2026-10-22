param([ValidateSet('22.04','24.04','26.04')][string]$Version)
$ErrorActionPreference = 'Stop'
$distro = "Ubuntu-$Version"
$out = Join-Path $env:RUNNER_TEMP 'wsl-evidence'
New-Item -ItemType Directory -Force $out | Out-Null
Start-Transcript -Path (Join-Path $out 'windows-transcript.log')
function Invoke-Wsl([string[]]$Arguments) {
    & wsl.exe @Arguments
    if ($LASTEXITCODE -ne 0) { throw "WSL command failed with $LASTEXITCODE; this target is not qualified." }
}
try {
    Invoke-Wsl @('--version')
    Invoke-Wsl @('--set-default-version', '2')
    Invoke-Wsl @('--install', '--distribution', $distro, '--no-launch', '--web-download')
    Invoke-Wsl @('--set-version', $distro, '2')
    Invoke-Wsl @('--distribution', $distro, '--user', 'root', '--exec', 'uname', '-a')
    $listing = (& wsl.exe --list --verbose | Out-String) -replace "`0", ''
    if ($LASTEXITCODE -ne 0) { throw 'Unable to inspect WSL generation' }
    $listing | Set-Content (Join-Path $out 'wsl-distributions.txt')
    $pattern = '(?m)^\s*\*?\s*' + [regex]::Escape($distro) + '\s+\S+\s+2\s*$'
    if ($listing -notmatch $pattern) { throw 'The selected distribution is not verified as WSL 2' }
    $linuxSource = (& wsl.exe -d $distro -u root --exec wslpath -u "$env:GITHUB_WORKSPACE/integrations/symphony-agent").Trim()
    if ($LASTEXITCODE -ne 0) { throw 'Source path conversion failed' }
    Invoke-Wsl @('-d', $distro, '-u', 'root', '--exec', 'mkdir', '-p', '/work/source')
    Invoke-Wsl @('-d', $distro, '-u', 'root', '--exec', 'cp', '-R', "$linuxSource/.", '/work/source/')
    Invoke-Wsl @('-d', $distro, '-u', 'root', '--exec', 'bash', '/work/source/ci/provision-linux.sh')
    Invoke-Wsl @('-d', $distro, '-u', 'symphonybuild', '--cd', '/work', '--exec', 'bash', '/work/source/ci/ci-linux.sh', $Version, 'x86_64', 'wsl2')
    Invoke-Wsl @('-d', $distro, '-u', 'symphonybuild', '--exec', 'bash', '-c', 'source /work/toolchain/paths.sh; exec "$PYTHON" /work/source/ci/make_wsl_fixture.py')
    Invoke-Wsl @('--terminate', $distro)
    & python "$env:GITHUB_WORKSPACE/integrations/symphony-agent/ci/wsl_stdio_smoke.py" --distribution $distro --output (Join-Path $out 'windows-stdio.json')
    if ($LASTEXITCODE -ne 0) { throw 'Windows-to-WSL stdio fixture failed' }
    @{ status='wsl2_bridge_fixtures_passed'; distribution=$distro; repository_commit=$env:GITHUB_SHA; agent_ui_qualified=$false; native_owner_qualified=$false } | ConvertTo-Json | Set-Content (Join-Path $out 'RESULTS.json')
} finally {
    # Preserve diagnostics even if nested virtualization or distro setup fails.
    $linuxOut = (& wsl.exe -d $distro -u root --exec wslpath -u $out 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -eq 0 -and $linuxOut) {
        foreach ($path in @('/work/result', '/work/toolchain', '/work/packages')) {
            & wsl.exe -d $distro -u root --exec cp -R $path "$linuxOut/" 2>$null
        }
    }
    Stop-Transcript
}
