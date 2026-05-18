# donk Inno Setup 打包脚本
# 需要先安装 Inno Setup: https://jrsoftware.org/isinfo.php

$ErrorActionPreference = "Stop"

function Write-Step {
    param([string]$Message)
    Write-Host "`n[$($script:stepCount)/5] $Message" -ForegroundColor Cyan
    $script:stepCount++
}

$script:stepCount = 1

Write-Host "========================================" -ForegroundColor Green
Write-Host "   donk Installer Builder"
Write-Host "========================================" -ForegroundColor Green

# 步骤 1: 检查 Inno Setup
Write-Step "检查 Inno Setup"
$isccPaths = @(
    "${env:ProgramFiles(x86)}\Inno Setup 6\iscc.exe",
    "${env:ProgramFiles}\Inno Setup 6\iscc.exe",
    "${env:LOCALAPPDATA}\Programs\Inno Setup 6\iscc.exe"
)

$isccPath = $null
foreach ($path in $isccPaths) {
    if (Test-Path $path) {
        $isccPath = $path
        break
    }
}

if (-not $isccPath) {
    # 尝试从 PATH 中查找
    $isccPath = (Get-Command iscc.exe -ErrorAction SilentlyContinue)?.Source
}

if (-not $isccPath) {
    Write-Host "`n[错误] 找不到 Inno Setup！" -ForegroundColor Red
    Write-Host "请先下载并安装 Inno Setup 6:"
    Write-Host "https://jrsoftware.org/isinfo.php" -ForegroundColor Yellow
    Write-Host "`n安装后请确保 iscc.exe 在系统 PATH 中，或安装到默认位置。"
    Read-Host "`n按 Enter 键退出"
    exit 1
}

Write-Host "找到 Inno Setup: $isccPath" -ForegroundColor Green

# 切换到项目根目录
$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

# 步骤 2: 获取依赖
Write-Step "获取 Flutter 依赖"
flutter pub get
if ($LASTEXITCODE -ne 0) {
    Write-Host "获取依赖失败！" -ForegroundColor Red
    Read-Host "按 Enter 键退出"
    exit 1
}

# 步骤 3: 构建 Windows 应用
Write-Step "构建 Windows 应用"
flutter build windows --release
if ($LASTEXITCODE -ne 0) {
    Write-Host "构建失败！" -ForegroundColor Red
    Read-Host "按 Enter 键退出"
    exit 1
}

# 步骤 4: 检查服务器程序
Write-Step "检查服务器程序"
$serverPath = "server\donk.exe"
if (-not (Test-Path $serverPath)) {
    Write-Warning "找不到服务器程序: $serverPath"
    Write-Host "请确保服务器程序已放置在 server 目录下" -ForegroundColor Yellow
    
    $continue = Read-Host "`n是否继续打包？ (y/N)"
    if ($continue -ne 'y' -and $continue -ne 'Y') {
        exit 1
    }
} else {
    Write-Host "服务器程序已找到: $serverPath" -ForegroundColor Green
}

# 步骤 5: 创建安装包
Write-Step "创建安装包"
$issPath = Join-Path $PSScriptRoot "setup.iss"
& $isccPath $issPath
if ($LASTEXITCODE -ne 0) {
    Write-Host "创建安装包失败！" -ForegroundColor Red
    Read-Host "按 Enter 键退出"
    exit 1
}

# 显示结果
$installerPath = "build\installer\donk_setup_1.0.4.exe"
if (Test-Path $installerPath) {
    $fileInfo = Get-Item $installerPath
    Write-Host "`n========================================" -ForegroundColor Green
    Write-Host "   打包成功！" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "`n安装包位置: $installerPath" -ForegroundColor White
    Write-Host "文件大小: $([math]::Round($fileInfo.Length / 1MB, 2)) MB" -ForegroundColor Gray
    Write-Host "`n可以直接分发此安装包给用户。" -ForegroundColor Green
} else {
    Write-Warning "找不到生成的安装包"
}

Read-Host "`n按 Enter 键退出"
