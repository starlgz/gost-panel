# 打包 gost-panel 发布版：交叉编译 Linux 二进制 + 前端产物 + 部署脚本 → tar.gz
#
# 用法（在仓库根目录）：
#   pwsh -File scripts/build-release.ps1                          # 默认 v0.1.0，amd64 + arm64
#   pwsh -File scripts/build-release.ps1 -Version v0.2.0
#   pwsh -File scripts/build-release.ps1 -Arch amd64
#   pwsh -File scripts/build-release.ps1 -Compress                # 额外做 UPX 压缩
#   pwsh -File scripts/build-release.ps1 -Compress -UpxMode best  # 用 -9 而非 --lzma
#
# 说明：
#   1. 用 .NET 的 System.Formats.Tar 建包而不是调用 tar.exe，
#      因为 Windows 的 tar.exe 无法写入 Unix 权限位（会得到 0666，二进制没有可执行位）。
#   2. -Compress 需要 upx：优先取 PATH，其次取 release/.tools/upx/**/upx.exe。
param(
    [string]$Version = "v0.1.0",
    [string[]]$Arch = @("amd64", "arm64"),
    [switch]$Compress,
    [ValidateSet("lzma", "best")][string]$UpxMode = "lzma",
    [string]$UpxPath = ""
)

$ErrorActionPreference = "Stop"

$Root    = Split-Path -Parent $PSScriptRoot   # scripts/ 的上一级即仓库根
$Backend = Join-Path $Root "backend"
$Static  = Join-Path $Root "static"
$OutDir  = Join-Path $Root "release"

function Info($m) { Write-Host "[INFO] $m"  -ForegroundColor Green }
function Warn($m) { Write-Host "[WARN] $m"  -ForegroundColor Yellow }
function Fail($m) { Write-Host "[ERROR] $m" -ForegroundColor Red; exit 1 }

# 权限位
$Mode755 = [System.IO.UnixFileMode]::UserRead -bor [System.IO.UnixFileMode]::UserWrite -bor [System.IO.UnixFileMode]::UserExecute -bor `
           [System.IO.UnixFileMode]::GroupRead -bor [System.IO.UnixFileMode]::GroupExecute -bor `
           [System.IO.UnixFileMode]::OtherRead -bor [System.IO.UnixFileMode]::OtherExecute
$Mode644 = [System.IO.UnixFileMode]::UserRead -bor [System.IO.UnixFileMode]::UserWrite -bor `
           [System.IO.UnixFileMode]::GroupRead -bor [System.IO.UnixFileMode]::OtherRead

$ExecutableNames = @("gost-panel", "deploy.sh")

# 自动探测 upx
function Resolve-Upx {
    param([string]$Explicit)
    if ($Explicit) {
        if (Test-Path $Explicit) { return $Explicit }
        Warn "指定的 upx 不存在: $Explicit"
    }
    $cmd = Get-Command upx -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    $found = Get-ChildItem -Path (Join-Path $OutDir ".tools") -Recurse -Filter "upx.exe" -ErrorAction SilentlyContinue |
             Select-Object -First 1
    if ($found) { return $found.FullName }
    return $null
}

# New-TarGz：打包目录为 tar.gz，显式写入 Unix 权限位
function New-TarGz {
    param([string]$SourceDir, [string]$EntryPrefix, [string]$OutFile)

    $fs = [System.IO.File]::Create($OutFile)
    $gz = [System.IO.Compression.GZipStream]::new($fs, [System.IO.Compression.CompressionLevel]::Optimal)
    $tw = [System.Formats.Tar.TarWriter]::new($gz, $false)
    try {
        $root = [System.Formats.Tar.PaxTarEntry]::new([System.Formats.Tar.TarEntryType]::Directory, "$EntryPrefix/")
        $root.Mode = $Mode755
        $tw.WriteEntry($root)

        foreach ($item in (Get-ChildItem -LiteralPath $SourceDir -Recurse -Force | Sort-Object FullName)) {
            $rel  = $item.FullName.Substring($SourceDir.Length).TrimStart('\', '/') -replace '\\', '/'
            $name = "$EntryPrefix/$rel"

            if ($item.PSIsContainer) {
                $e = [System.Formats.Tar.PaxTarEntry]::new([System.Formats.Tar.TarEntryType]::Directory, "$name/")
                $e.Mode = $Mode755
                $e.ModificationTime = $item.LastWriteTimeUtc
                $tw.WriteEntry($e)
            }
            else {
                $mode = $Mode644
                if ($ExecutableNames -contains $item.Name) { $mode = $Mode755 }
                $e = [System.Formats.Tar.PaxTarEntry]::new([System.Formats.Tar.TarEntryType]::RegularFile, $name)
                $e.Mode = $mode
                $e.ModificationTime = $item.LastWriteTimeUtc
                $e.DataStream = [System.IO.File]::OpenRead($item.FullName)
                try { $tw.WriteEntry($e) } finally { $e.DataStream.Dispose() }
            }
        }
    }
    finally {
        $tw.Dispose(); $gz.Dispose(); $fs.Dispose()
    }
}

# 读取 ELF 机器类型
function Get-ElfMachine {
    param([string]$Path)
    $b = [System.IO.File]::ReadAllBytes($Path)
    if ($b.Length -lt 20) { return -1 }
    if (-not ($b[0] -eq 0x7F -and $b[1] -eq 0x45 -and $b[2] -eq 0x4C -and $b[3] -eq 0x46)) { return -1 }
    return $b[18] -bor ($b[19] -shl 8)
}

# ---------- 前置校验 ----------
if (-not (Test-Path (Join-Path $Backend "go.mod"))) { Fail "找不到 backend/go.mod" }
if (-not (Test-Path (Join-Path $Static "index.html"))) {
    Fail "找不到前端产物 static/index.html —— 请先执行 cd web && npm run build"
}
if (-not (Test-Path (Join-Path $Root "scripts\release\deploy.sh"))) { Fail "缺少 scripts/release/deploy.sh" }

$Upx = $null
if ($Compress) {
    $Upx = Resolve-Upx -Explicit $UpxPath
    if (-not $Upx) { Fail "启用了 -Compress 但找不到 upx（可放到 PATH，或 release/.tools/upx/ 下）" }
    Info "UPX: $Upx  (模式: $UpxMode)"
}

$Commit = "unknown"
try { $Commit = (git -C $Root rev-parse --short HEAD 2>$null) } catch { }
if (-not $Commit) { $Commit = "unknown" }
$BuildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$GoVersion = (& go version)

Info "版本 $Version  commit $Commit"

if (-not (Test-Path $OutDir)) { New-Item -ItemType Directory -Force -Path $OutDir | Out-Null }
# 只替换本次指定版本和架构的产物，保留其他发布包。

$results = @()

foreach ($a in $Arch) {
    $pkgName = "gost-panel-$Version-linux-$a"
    $stage   = Join-Path $OutDir $pkgName
    $resolvedStage = [System.IO.Path]::GetFullPath($stage)
    $resolvedOutDir = [System.IO.Path]::GetFullPath($OutDir).TrimEnd('\', '/') + [System.IO.Path]::DirectorySeparatorChar
    if (-not $resolvedStage.StartsWith($resolvedOutDir, [StringComparison]::OrdinalIgnoreCase)) { Fail "打包路径越界" }
    if (Test-Path -LiteralPath $resolvedStage) { Remove-Item -LiteralPath $resolvedStage -Recurse -Force }
    New-Item -ItemType Directory -Force -Path $stage | Out-Null

    # ---------- 1. 交叉编译 ----------
    Info "编译 linux/$a ..."
    $env:CGO_ENABLED = "0"
    $env:GOOS        = "linux"
    $env:GOARCH      = $a
    $binPath  = Join-Path $stage "gost-panel"
    $ldflags  = "-s -w -X gost-panel/internal/api.Version=$Version"
    Push-Location $Backend
    try {
        & go build -trimpath -ldflags $ldflags -o $binPath ./cmd/gost-panel
        if ($LASTEXITCODE -ne 0) { Fail "linux/$a 编译失败" }
    } finally { Pop-Location }
    Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue

    $rawSize = (Get-Item $binPath).Length

    # ELF 与架构校验（在 UPX 之前，避免压缩干扰判断）
    $expected = if ($a -eq "amd64") { 0x3E } else { 0xB7 }
    $machine  = Get-ElfMachine -Path $binPath
    if ($machine -lt 0)         { Fail "$pkgName 不是 ELF 文件" }
    if ($machine -ne $expected) { Fail "$pkgName 架构不符: 0x$("{0:X}" -f $machine)" }

    # ---------- 2. UPX 压缩（可选）----------
    $compressNote = "none"
    if ($Compress) {
        Info "UPX 压缩 linux/$a ..."
        $upxArg = if ($UpxMode -eq "lzma") { "--lzma" } else { "-9" }
        & $Upx $upxArg --force $binPath 2>&1 | Out-Null
        if ($LASTEXITCODE -ne 0) { Fail "UPX 压缩失败: $pkgName" }
        # 压缩后复验 ELF 头未被破坏
        if ((Get-ElfMachine -Path $binPath) -ne $expected) { Fail "UPX 压缩后 ELF 头损坏: $pkgName" }
        $compressNote = "upx $upxArg"
        # 保留可执行位（UPX 在 Windows 上会重写文件属性）
        Info "  $([math]::Round($rawSize/1MB,2)) MB -> $([math]::Round((Get-Item $binPath).Length/1MB,2)) MB"
    }

    # ---------- 3. 前端产物 ----------
    Info "复制前端产物 static/ ..."
    Copy-Item -Path $Static -Destination (Join-Path $stage "static") -Recurse -Force

    # ---------- 4. 部署脚本 ----------
    # Bash 文件统一使用 LF，避免 Windows 的 CRLF 使 Linux 执行失败。
    $deploySource = [System.IO.File]::ReadAllText((Join-Path $Root "scripts\release\deploy.sh")) -replace "`r`n", "`n"
    [System.IO.File]::WriteAllText((Join-Path $stage "deploy.sh"), $deploySource, [System.Text.UTF8Encoding]::new($false))

    # ---------- 5. 文档与许可 ----------
    foreach ($doc in @("README.md", "LICENSE")) {
        $p = Join-Path $Root $doc
        if (Test-Path $p) { Copy-Item $p (Join-Path $stage $doc) -Force }
    }

    # ---------- 6. 构建信息 ----------
    $binSize = (Get-Item $binPath).Length
    @"
gost-panel $Version
build_time: $BuildTime
git_commit: $Commit
target:     linux/$a
go_version: $GoVersion
compression: $compressNote
binary_size: $binSize bytes
"@ | Set-Content -Path (Join-Path $stage "BUILD_INFO.txt") -Encoding UTF8NoBOM

    # ---------- 7. 打包（保留权限位）----------
    Info "打包 $pkgName.tar.gz ..."
    $tarPath = Join-Path $OutDir "$pkgName.tar.gz"
    New-TarGz -SourceDir $stage -EntryPrefix $pkgName -OutFile $tarPath

    # ---------- 8. 校验 ----------
    $hash    = (Get-FileHash $tarPath -Algorithm SHA256).Hash.ToLower()
    $tarSize = (Get-Item $tarPath).Length

    $results += [pscustomobject]@{
        Package  = "$pkgName.tar.gz"
        Tarball  = "$([math]::Round($tarSize/1MB,2)) MB"
        Binary   = "$([math]::Round($binSize/1MB,2)) MB"
        Raw      = "$([math]::Round($rawSize/1MB,2)) MB"
        Arch     = if ($a -eq "amd64") { "x86-64" } else { "aarch64" }
        SHA256   = $hash
    }
    Info "$pkgName 完成（包 $([math]::Round($tarSize/1MB,2)) MB，二进制 $([math]::Round($binSize/1MB,2)) MB）"
}

# ---------- 汇总 ----------
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "================ 打包结果 ================" -ForegroundColor Cyan
$results | Format-Table Package, Tarball, Binary, Raw, Arch -AutoSize

if ($Compress) {
    Write-Host "Raw = UPX 压缩前的原始二进制体积" -ForegroundColor DarkGray
}
Write-Host "输出目录: $OutDir" -ForegroundColor Cyan

$sumFile = Join-Path $OutDir "SHA256SUMS.txt"
$results | ForEach-Object { "$($_.SHA256)  $($_.Package)" } | Set-Content $sumFile -Encoding UTF8NoBOM
Info "校验和已写入 $sumFile"
