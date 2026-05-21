; Inno Setup 脚本 for donk
; 需要先安装 Inno Setup: https://jrsoftware.org/isinfo.php

#define MyAppName "Donk"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "LongstageAI"
#define MyAppExeName "donk.exe"
#define MyAppAssocName MyAppName + " File"
#define MyAppAssocExt ".Donk"
#define MyAppAssocKey StringChange(MyAppAssocName, " ", "") + MyAppAssocExt

[Setup]
; 应用基本信息
AppId={{8B4A5C3D-2E1F-4A5B-9C8D-7E6F5A4B3C2D}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\{#MyAppName}
UninstallDisplayIcon={app}\{#MyAppExeName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes

; 输出设置
OutputDir=..\build\installer
OutputBaseFilename=donk_{#MyAppVersion}
SetupIconFile=..\assets\img\app2.ico
Compression=lzma
SolidCompression=yes
WizardStyle=modern

; 权限设置
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog

; 版本信息
VersionInfoVersion={#MyAppVersion}
VersionInfoCompany={#MyAppPublisher}
VersionInfoDescription={#MyAppName} Setup
VersionInfoTextVersion={#MyAppVersion}

[Languages]
Name: "chinesesimplified"; MessagesFile: "compiler:Languages\ChineseSimplified.isl";

[Messages]
; 自定义中文消息
WelcomeLabel1=欢迎使用 [name] 安装向导
WelcomeLabel2=本向导将指导您完成 [name] 的安装过程。%n%n建议您在继续之前关闭其他所有应用程序。

ClickNext=点击“下一步”继续，或点击“取消”退出安装。
ClickInstall=点击“安装”开始安装，或点击“上一步”查看或修改设置。
ClickFinish=点击“完成”退出安装向导。

LicenseLabel=请阅读以下许可协议：
LicenseLabel3=请阅读以下许可协议。在继续安装之前，您必须接受此协议的条款。

SelectDirLabel=请选择安装位置。
SelectDirDesc=您想将 [name] 安装到哪个文件夹？
SelectDirBrowseLabel=点击“下一步”继续。如果您想选择其他文件夹，请点击“浏览”。

DiskSpaceMBLabel=至少需要 [mb] MB 的磁盘空间。

ReadyLabel=安装程序已准备好开始安装 [name]。
ReadyLabel2=点击“安装”开始安装，或点击“上一步”查看或修改设置。

PreparingDesc=正在准备安装...
InstallingLabel=正在安装 [name]，请稍候...

FinishedLabel=已完成 [name] 的安装。
FinishedLabel2=安装程序已完成 [name] 的安装。%n%n点击“完成”退出安装向导。

FinishedRestartLabel=要完成 [name] 的安装，安装程序必须重新启动您的计算机。您想立即重新启动吗？
FinishedRestartMessage=要完成 [name] 的安装，安装程序必须重新启动您的计算机。%n%n您想立即重新启动吗？

ShowReadmeCheck=安装完成后显示 readme 文件

YesRadio=是，立即重新启动计算机(&Y)
NoRadio=否，稍后重新启动计算机(&N)

RunEntryExec=运行 [name]
RunEntryShellExec=查看 [filename]

[Tasks]
Name: "desktopicon"; Description: "创建桌面快捷方式(&D)"; GroupDescription: "附加任务："; Flags: checkedonce
Name: "quicklaunchicon"; Description: "创建快速启动栏快捷方式(&Q)"; GroupDescription: "附加任务："; Flags: unchecked; OnlyBelowVersion: 6.1; Check: not IsAdminInstallMode

[Files]
; 主程序文件
Source: "..\build\windows\x64\runner\Release\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\build\windows\x64\runner\Release\*.dll"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs
Source: "..\build\windows\x64\runner\Release\data\*"; DestDir: "{app}\data"; Flags: ignoreversion recursesubdirs
Source: "..\build\windows\x64\runner\Release\server\*"; DestDir: "{app}\server"; Flags: ignoreversion recursesubdirs

; 图标文件
Source: "..\assets\img\app2.ico"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
; 开始菜单快捷方式
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\app2.ico"
Name: "{group}\卸载 {#MyAppName}"; Filename: "{uninstallexe}"

; 桌面快捷方式
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\app2.ico"; Tasks: desktopicon

[Run]
; 安装完成后可选启动
Filename: "{app}\{#MyAppExeName}"; Description: "运行 {#MyAppName}(&R)"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
; 卸载时删除服务器程序生成的临时文件
Type: filesandordirs; Name: "{app}\server\*"

[Code]
// 检查是否已安装旧版本
function InitializeSetup(): Boolean;
var
  Version: String;
begin
  if RegQueryStringValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{#SetupSetting("AppId")}_is1', 'DisplayVersion', Version) then
  begin
    if MsgBox('检测到已安装版本 ' + Version + '，是否先卸载旧版本？', mbConfirmation, MB_YESNO) = IDYES then
    begin
      // 这里可以调用卸载程序
    end;
  end;
  Result := true;
end;


