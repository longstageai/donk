@echo off
chcp 65001 >nul
echo ========================================
echo   donk Installer Builder
echo ========================================
echo.

:: 切换到项目根目录
cd /d "%~dp0\.."

:: 检查 Inno Setup 是否安装
set "ISCC_PATH="
for %%i in (iscc.exe) do set "ISCC_PATH=%%~$PATH:i"

if not defined ISCC_PATH (
    echo [错误] 找不到 Inno Setup！
    echo.
    echo 请先安装 Inno Setup:
    echo https://jrsoftware.org/isinfo.php
    echo.
    echo 安装后请确保 iscc.exe 在系统 PATH 中
    pause
    exit /b 1
)

echo [1/4] 检查 Inno Setup... 找到: %ISCC_PATH%
echo.

:: 步骤 2: 获取依赖
echo [2/4] 获取 Flutter 依赖...
call flutter pub get
if errorlevel 1 (
    echo [错误] 获取依赖失败！
    pause
    exit /b 1
)
echo.

:: 步骤 3: 构建 Windows 应用
echo [3/4] 构建 Windows 应用...
call flutter build windows --release
if errorlevel 1 (
    echo [错误] 构建失败！
    pause
    exit /b 1
)
echo.

:: 步骤 4: 检查服务器程序
echo [4/4] 检查服务器程序...
if not exist "server\donk.exe" (
    echo [警告] 找不到服务器程序: server\donk.exe
    echo 请确保服务器程序已放置在 server 目录下
    echo.
    choice /C YN /M "是否继续打包"
    if errorlevel 2 exit /b 1
)
echo.

:: 步骤 5: 创建安装包
echo [5/5] 创建安装包...
"%ISCC_PATH%" "scripts\setup.iss"
if errorlevel 1 (
    echo [错误] 创建安装包失败！
    pause
    exit /b 1
)

echo.
echo ========================================
echo   打包成功！
echo ========================================
echo.
echo 安装包位置: build\installer\donk_setup_1.0.4.exe
echo.
pause
