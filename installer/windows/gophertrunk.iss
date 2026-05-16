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
; Zadig WinUSB driver installer — bundled so the operator doesn't
; have to chase a download. GPL-3.0; upstream source is at
; https://github.com/pbatard/libwdi (see THIRD_PARTY_LICENSES.md).
; Zadig's embedded manifest requests admin elevation on its own.
Source: "..\..\dist\staging\zadig.exe"; DestDir: "{app}"; Flags: ignoreversion
; Seed the operator's chosen editable-files folder with a starter
; config.yaml the first time they install. onlyifdoesntexist
; preserves any edits across re-installs; uninsneveruninstall
; leaves the file behind on uninstall so the operator doesn't lose
; the config (and any per-system trunking data alongside it).
Source: "..\..\dist\staging\config.example.yaml"; \
  DestDir: "{code:ConfigDir}"; \
  DestName: "config.yaml"; \
  Flags: onlyifdoesntexist uninsneveruninstall
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
Name: "{group}\Edit my config.yaml (Notepad)"; \
  Filename: "notepad.exe"; \
  Parameters: """{code:ConfigDir}\config.yaml"""; \
  Comment: "Open the config file the daemon will load on startup"
Name: "{group}\Configuration template (read-only reference)"; \
  Filename: "notepad.exe"; \
  Parameters: """{app}\config.example.yaml"""
Name: "{group}\Windows install instructions"; \
  Filename: "{app}\INSTALL-WINDOWS.md"
Name: "{group}\Visit project on GitHub"; \
  Filename: "https://github.com/MattCheramie/GopherTrunk"
Name: "{group}\Install RTL-SDR driver (Zadig)"; \
  Filename: "{app}\zadig.exe"; \
  WorkingDir: "{app}"; \
  Comment: "Swap your RTL-SDR's driver to WinUSB (one-time, per dongle). Triggers UAC on launch."
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
; Per-user env var pointing at the operator's chosen config.yaml.
; The daemon's internal/config.Discover() reads this first when no
; -config flag is passed, so launching the daemon from any shell
; "just works". ChangesEnvironment=yes (above) triggers Inno's
; WM_SETTINGCHANGE broadcast so newly-opened shells see the value.
; uninsdeletevalue cleans the variable up if the operator
; uninstalls, even though the config.yaml file itself is preserved.
Root: HKCU; Subkey: "Environment"; \
  ValueType: expandsz; ValueName: "GOPHERTRUNK_CONFIG"; \
  ValueData: "{code:ConfigDir}\config.yaml"; \
  Flags: uninsdeletevalue
; Persist install-time ConfigDir / WebUIDir so the uninstaller can
; find them. Inno's [Code] state from the install run does NOT
; survive into the uninstall run, so the registry is the only
; durable bridge. uninsdeletekeyifempty on the last entry sweeps
; the parent Install subkey once both values are gone.
Root: HKLM; Subkey: "Software\GopherTrunk\Install"; \
  ValueType: string; ValueName: "ConfigDir"; \
  ValueData: "{code:ConfigDir}"; \
  Flags: uninsdeletevalue
Root: HKLM; Subkey: "Software\GopherTrunk\Install"; \
  ValueType: string; ValueName: "WebUIDir"; \
  ValueData: "{code:WebUIDir}"; \
  Flags: uninsdeletevalue uninsdeletekeyifempty

[Run]
Filename: "{app}\INSTALL-WINDOWS.md"; \
  Description: "Open the Windows install instructions (Zadig + first run)"; \
  Flags: postinstall shellexec skipifsilent
Filename: "{cmd}"; \
  Parameters: "/k cd /d ""{app}"" && gophertrunk help"; \
  Description: "Open a console window in the install dir"; \
  Flags: postinstall skipifsilent unchecked
Filename: "{app}\zadig.exe"; \
  WorkingDir: "{app}"; \
  Description: "Run Zadig now to bind the WinUSB driver to your RTL-SDR"; \
  Flags: postinstall shellexec skipifsilent unchecked
Filename: "{code:WebUIDir}\index.html"; \
  Description: "Open the web operator console now"; \
  Flags: postinstall shellexec skipifsilent; \
  Tasks: webui

[Code]
var
  ConfigPage: TInputDirWizardPage;
  WebUIPage:  TInputDirWizardPage;

procedure InitializeWizard;
begin
  // Editable-files folder: where the operator's config.yaml (and
  // any per-system data that lands next to it) lives. We default
  // to Documents\GopherTrunk because it's the spot non-Admin
  // Windows users can always write to without surprises — and the
  // [Files] step seeds a starter config.yaml there. The path is
  // also written to HKCU\Environment\GOPHERTRUNK_CONFIG so the
  // daemon discovers it without needing -config on the command
  // line. Placed first so the operator sees the most important
  // path choice before anything else.
  ConfigPage := CreateInputDirPage(
    wpSelectTasks,
    'Select your editable-files folder',
    'Where should your config.yaml and per-system data live?',
    'Pick a folder Setup can drop a starter config.yaml in. The ' +
    'GopherTrunk daemon will look for config.yaml in this folder ' +
    'automatically — no -config flag needed. The default is your ' +
    'Documents folder so it''s easy to find and back up; you can ' +
    'choose anywhere you can write to. If a config.yaml already ' +
    'exists in the folder it will NOT be overwritten.',
    False, '');
  ConfigPage.Add('Editable files folder:');
  ConfigPage.Values[0] :=
    ExpandConstant('{userdocs}\GopherTrunk');

  // CreateInputDirPage gives us a "pick a folder" wizard step with a
  // Browse button. Placed AFTER ConfigPage so the page order matches
  // the order of importance — config first, web UI second. ShouldSkipPage
  // hides this one entirely when the webui task is unchecked.
  WebUIPage := CreateInputDirPage(
    ConfigPage.ID,
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

function ConfigDir(Param: string): string;
begin
  Result := ConfigPage.Values[0];
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

// ---------------------------------------------------------------
// Uninstall helpers.
//
// Inno's [Code] state from the install run does NOT survive into
// the uninstall run, so the install-time ConfigDir / WebUIDir
// choices are read back from HKLM\Software\GopherTrunk\Install
// (populated by the [Registry] section).
// ---------------------------------------------------------------

function ReadInstalledConfigDir(): string;
begin
  if not RegQueryStringValue(HKEY_LOCAL_MACHINE,
    'Software\GopherTrunk\Install', 'ConfigDir', Result)
  then
    Result := '';
end;

function ReadInstalledWebUIDir(): string;
begin
  if not RegQueryStringValue(HKEY_LOCAL_MACHINE,
    'Software\GopherTrunk\Install', 'WebUIDir', Result)
  then
    Result := '';
end;

// Strip the {app} entry from the HKLM system Path. Sandwich with
// ';' so we match start / middle / end and never chop a path
// that's a suffix of another (C:\App vs C:\AppX). No-op if our
// entry isn't there.
procedure RemoveAppFromHKLMPath();
var
  OrigPath, NewPath, AppDir, Needle: string;
  P: Integer;
begin
  AppDir := ExpandConstant('{app}');
  if not RegQueryStringValue(HKEY_LOCAL_MACHINE,
    'SYSTEM\CurrentControlSet\Control\Session Manager\Environment',
    'Path', OrigPath)
  then exit;

  Needle := ';' + AppDir + ';';
  NewPath := ';' + OrigPath + ';';
  P := Pos(Needle, NewPath);
  if P = 0 then exit;

  Delete(NewPath, P, Length(Needle) - 1); // leave one ';' in place
  if (Length(NewPath) > 0) and (NewPath[1] = ';') then
    Delete(NewPath, 1, 1);
  if (Length(NewPath) > 0) and (NewPath[Length(NewPath)] = ';') then
    Delete(NewPath, Length(NewPath), 1);

  RegWriteExpandStringValue(HKEY_LOCAL_MACHINE,
    'SYSTEM\CurrentControlSet\Control\Session Manager\Environment',
    'Path', NewPath);
end;

procedure WipeConfig();
var
  Dir, Cfg: string;
begin
  Dir := ReadInstalledConfigDir();
  if Dir = '' then exit;
  Cfg := AddBackslash(Dir) + 'config.yaml';
  if FileExists(Cfg) then
    DeleteFile(Cfg);
  // RemoveDir only succeeds on empty dirs — that's the right
  // semantics: don't blow away a folder still holding the
  // operator's call-log database or recordings.
  if DirExists(Dir) then
    RemoveDir(Dir);
end;

procedure WipeWebConsole();
var
  Dir, Sub: string;
begin
  Dir := ReadInstalledWebUIDir();
  if Dir = '' then exit;
  Sub := AddBackslash(Dir) + 'gophertrunk-web';
  if DirExists(Sub) then
    DelTree(Sub, True, True, True);
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  WipeAnswer: Integer;
begin
  if CurUninstallStep = usUninstall then begin
    // Always strip our PATH entry — the [Registry] section never
    // got a cleanup flag, so this is the only place it happens.
    // The HKCU GOPHERTRUNK_CONFIG value and the
    // Software\GopherTrunk\Install key clean themselves up via
    // uninsdeletevalue / uninsdeletekeyifempty.
    RemoveAppFromHKLMPath();

    WipeAnswer := MsgBox(
      'Also remove your editable config.yaml and the web console folder?' + #13#10 + #13#10 +
      'Yes = full wipe (delete config.yaml + the gophertrunk-web folder Setup created).' + #13#10 +
      'No  = preserve your user data (recommended).',
      mbConfirmation, MB_YESNO or MB_DEFBUTTON2);
    if WipeAnswer = IDYES then begin
      WipeConfig();
      WipeWebConsole();
    end;
  end;
end;
