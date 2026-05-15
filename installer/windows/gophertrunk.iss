; Inno Setup script for the GopherTrunk Windows installer.
;
; Driven from the GitHub Actions release workflow with:
;
;   iscc /DAppVersion=v0.1.0 installer/windows/gophertrunk.iss
;
; The workflow stages the .exe + DLLs + docs under dist\staging\ first
; (see .github/workflows/release.yml). This script consumes that
; directory and produces a single setup.exe under dist\ named
; gophertrunk-<version>-windows-amd64-setup.exe.
;
; Inno Setup is a freely-distributed Windows installer compiler. Docs:
; https://jrsoftware.org/isinfo.php

#ifndef AppVersion
  #define AppVersion "v0.0.0-dev"
#endif

[Setup]
AppId={{B6B6CC9A-3A70-4B23-8E2E-8E0C7A2F4B30}
AppName=GopherTrunk
AppVersion={#AppVersion}
AppPublisher=GopherTrunk contributors
AppPublisherURL=https://github.com/MattCheramie/GopherTrunk
AppSupportURL=https://github.com/MattCheramie/GopherTrunk/issues
AppUpdatesURL=https://github.com/MattCheramie/GopherTrunk/releases
DefaultDirName={autopf}\GopherTrunk
DefaultGroupName=GopherTrunk
DisableProgramGroupPage=yes
LicenseFile=..\..\LICENSE
OutputDir=..\..\dist
OutputBaseFilename=gophertrunk-{#AppVersion}-windows-amd64-setup
Compression=lzma
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=admin
ChangesEnvironment=yes
UninstallDisplayIcon={app}\gophertrunk.exe

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "addtopath"; Description: "Add GopherTrunk to my PATH (so I can run ""gophertrunk"" from any terminal)"; GroupDescription: "PATH"; Flags: unchecked
Name: "desktopicon"; Description: "Create a &desktop shortcut"; GroupDescription: "Additional shortcuts:"; Flags: unchecked
Name: "webui"; Description: "Install the &web operator console (a static HTML / JS folder you open in any browser)"; GroupDescription: "Web operator console:"
Name: "webui\desktopicon"; Description: "Create a desktop shortcut for the web console"; GroupDescription: "Web operator console:"; Flags: unchecked

[Files]
Source: "..\..\dist\staging\gophertrunk.exe";  DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\dist\staging\config.example.yaml"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\dist\staging\README.md";        DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\dist\staging\LICENSE";          DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\dist\staging\INSTALL-WINDOWS.md"; DestDir: "{app}"; Flags: ignoreversion
; The web console is a standalone static folder — index.html plus the
; bundled JS/CSS/manifest. The user picks the destination on the
; custom WebUIPage below; {code:WebUIDir} resolves to that choice.
Source: "..\..\dist\staging\gophertrunk-web\*"; \
  DestDir: "{code:WebUIDir}"; \
  Flags: ignoreversion recursesubdirs createallsubdirs; \
  Tasks: webui

[Icons]
Name: "{group}\GopherTrunk (PowerShell)"; Filename: "{cmd}"; \
  Parameters: "/k cd /d ""{app}"" && gophertrunk help"; \
  WorkingDir: "{app}"; \
  Comment: "Open a console with GopherTrunk on PATH"
Name: "{group}\Configuration template (open in Notepad)"; \
  Filename: "notepad.exe"; \
  Parameters: """{app}\config.example.yaml"""
Name: "{group}\Windows install instructions"; \
  Filename: "{app}\INSTALL-WINDOWS.md"
Name: "{group}\Visit project on GitHub"; \
  Filename: "https://github.com/MattCheramie/GopherTrunk"
Name: "{group}\Uninstall GopherTrunk"; Filename: "{uninstallexe}"
Name: "{autodesktop}\GopherTrunk"; Filename: "{cmd}"; \
  Parameters: "/k cd /d ""{app}"" && gophertrunk help"; \
  WorkingDir: "{app}"; \
  Tasks: desktopicon
; Web operator console shortcuts. shellexec opens the file in the
; user's default browser; the entry resolves to whatever path the
; user picked on the WebUIPage.
Name: "{group}\Web operator console"; \
  Filename: "{code:WebUIDir}\index.html"; \
  Comment: "Open the GopherTrunk web operator console in your default browser"; \
  Tasks: webui
Name: "{autodesktop}\GopherTrunk Web Console"; \
  Filename: "{code:WebUIDir}\index.html"; \
  Comment: "Open the GopherTrunk web operator console in your default browser"; \
  Tasks: webui\desktopicon

[Registry]
; Append the install dir to the system PATH if the user opted in. Inno
; Setup re-broadcasts WM_SETTINGCHANGE so already-open shells pick it
; up after the next launch.
Root: HKLM; Subkey: "SYSTEM\CurrentControlSet\Control\Session Manager\Environment"; \
  ValueType: expandsz; ValueName: "Path"; \
  ValueData: "{olddata};{app}"; \
  Check: NeedsAddPath('{app}'); \
  Tasks: addtopath

[Run]
Filename: "{app}\INSTALL-WINDOWS.md"; \
  Description: "Open the Windows install instructions (Zadig + first run)"; \
  Flags: postinstall shellexec skipifsilent
Filename: "{cmd}"; \
  Parameters: "/k cd /d ""{app}"" && gophertrunk help"; \
  Description: "Open a console window in the install dir"; \
  Flags: postinstall skipifsilent unchecked
Filename: "{code:WebUIDir}\index.html"; \
  Description: "Open the web operator console now"; \
  Flags: postinstall shellexec skipifsilent; \
  Tasks: webui

[Code]
var
  WebUIPage: TInputDirWizardPage;

procedure InitializeWizard;
begin
  // CreateInputDirPage gives us a "pick a folder" wizard step with a
  // Browse button. Placed AFTER wpSelectTasks so we know whether the
  // user actually wants the web console — ShouldSkipPage hides the
  // page entirely when the task is unchecked.
  WebUIPage := CreateInputDirPage(
    wpSelectTasks,
    'Select web operator console location',
    'Where should Setup put the GopherTrunk web console?',
    'Pick a folder for the standalone web UI. Setup will copy a ' +
    'gophertrunk-web folder there containing an index.html you open ' +
    'in any browser. The default is your Documents folder so it''s ' +
    'easy to find later; you can choose anywhere — a USB stick, a ' +
    'network drive, or your desktop. Use Browse to pick a different ' +
    'folder.',
    False, '');
  WebUIPage.Add('Web console folder:');
  WebUIPage.Values[0] :=
    ExpandConstant('{userdocs}\GopherTrunk Web Console');
end;

function ShouldSkipPage(PageID: Integer): Boolean;
begin
  Result := False;
  // Skip the web-UI directory page when the user unchecked the
  // "Install the web operator console" task.
  if PageID = WebUIPage.ID then begin
    Result := not WizardIsTaskSelected('webui');
  end;
end;

function WebUIDir(Param: string): string;
begin
  Result := WebUIPage.Values[0];
end;

function NeedsAddPath(Param: string): boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKEY_LOCAL_MACHINE,
    'SYSTEM\CurrentControlSet\Control\Session Manager\Environment',
    'Path', OrigPath)
  then begin
    Result := True;
    exit;
  end;
  // Pos returns 0 if the substring isn't found.
  Result := Pos(';' + ExpandConstant(Param) + ';',
                ';' + OrigPath + ';') = 0;
end;
