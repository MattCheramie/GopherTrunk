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
Name: "desktopicon"; Description: "Create &desktop shortcuts (GopherTrunk console + web operator console)"; GroupDescription: "Additional shortcuts:"; Flags: unchecked

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
; Seed the config subfolder of the operator's chosen data root with a
; starter config.yaml the first time they install. The shipped
; config.example.yaml uses config-relative paths (../recordings,
; ../data, ../iq, ../logs), so once it lands in <DataRoot>\config the
; daemon writes every other file into the sibling folders created by
; the [Dirs] section below. onlyifdoesntexist preserves any edits
; across re-installs; uninsneveruninstall leaves the file behind on
; uninstall so the operator doesn't lose their config.
Source: "..\..\dist\staging\config.example.yaml"; \
  DestDir: "{code:ConfigDir}"; \
  DestName: "config.yaml"; \
  Flags: onlyifdoesntexist uninsneveruninstall
; The web operator consoles are standalone static folders — index.html
; plus bundled JS/CSS/manifest. The release staging nests all four
; UIs under gophertrunk-web\ (standard console at the top,
; gophertrunk-web\siglab\, gophertrunk-web\rfscope\ and
; gophertrunk-web\configbuilder\ alongside), so this one recursive copy
; reproduces the whole tree under <DataRoot>\web (web\index.html,
; web\siglab\, web\rfscope\, web\configbuilder\).
; {code:WebDir} resolves to <DataRoot>\web.
Source: "..\..\dist\staging\gophertrunk-web\*"; \
  DestDir: "{code:WebDir}"; \
  Flags: ignoreversion recursesubdirs createallsubdirs

[Dirs]
; Logical subfolder tree under the operator's chosen data root. The
; config-relative defaults in config.yaml resolve into these siblings,
; so all of the operator's files live under one parent: config files,
; voice-call recordings, raw IQ baseband captures, CSV/PDF exports, the
; SQLite database + caches, decoded-message logs, and the web consoles.
Name: "{code:DataDir}\config"
Name: "{code:DataDir}\recordings"
Name: "{code:DataDir}\iq"
Name: "{code:DataDir}\exports"
Name: "{code:DataDir}\data"
Name: "{code:DataDir}\logs"
Name: "{code:DataDir}\web"

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
; Web operator console shortcuts. The web\ folder under the data root
; holds all four consoles; opening these index.html files launches
; them in the user's default browser.
Name: "{group}\Web operator console"; \
  Filename: "{code:WebDir}\index.html"; \
  Comment: "Open the GopherTrunk web operator console in your default browser"
Name: "{group}\Signal Lab console"; \
  Filename: "{code:WebDir}\siglab\index.html"; \
  Comment: "Open the GopherTrunk Signal Lab (offline signal analysis) console"
Name: "{group}\RF Scope console"; \
  Filename: "{code:WebDir}\rfscope\index.html"; \
  Comment: "Open the GopherTrunk RF Scope (protocol-agnostic RF analysis) console"
Name: "{group}\Config Builder console"; \
  Filename: "{code:WebDir}\configbuilder\index.html"; \
  Comment: "Open the GopherTrunk Config Builder / editor in your default browser"
Name: "{autodesktop}\GopherTrunk Web Console"; \
  Filename: "{code:WebDir}\index.html"; \
  Comment: "Open the GopherTrunk web operator console in your default browser"; \
  Tasks: desktopicon

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
; Per-user env var pointing at the data root itself. Operators who
; prefer env-var-based config paths can reference ${GOPHERTRUNK_HOME}
; / %GOPHERTRUNK_HOME% in config.yaml, and external tooling can locate
; the data folder without parsing config.yaml.
Root: HKCU; Subkey: "Environment"; \
  ValueType: expandsz; ValueName: "GOPHERTRUNK_HOME"; \
  ValueData: "{code:DataDir}"; \
  Flags: uninsdeletevalue
; Persist the install-time data root so the uninstaller can find it.
; Inno's [Code] state from the install run does NOT survive into the
; uninstall run, so the registry is the only durable bridge.
; uninsdeletekeyifempty sweeps the parent Install subkey once the
; value is gone.
Root: HKLM; Subkey: "Software\GopherTrunk\Install"; \
  ValueType: string; ValueName: "DataDir"; \
  ValueData: "{code:DataDir}"; \
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
Filename: "{code:WebDir}\index.html"; \
  Description: "Open the web operator console now"; \
  Flags: postinstall shellexec skipifsilent

[Code]
var
  DataDirPage: TInputDirWizardPage;

procedure InitializeWizard;
begin
  // Single data-folder choice. The executable always installs to
  // {app} (Program Files); this page picks the SEPARATE, user-owned
  // data root that holds everything else — config files, voice-call
  // recordings, raw IQ baseband captures, CSV/PDF exports, the SQLite
  // database + caches, decoded-message logs, and the web consoles —
  // each in its own subfolder (created by the [Dirs] section). We
  // default to Documents\GopherTrunk because it's the spot non-Admin
  // Windows users can always write to without surprises. The chosen
  // path is written to HKCU\Environment\GOPHERTRUNK_HOME, and
  // GOPHERTRUNK_CONFIG points at <DataRoot>\config\config.yaml so the
  // daemon discovers the seeded config without a -config flag.
  DataDirPage := CreateInputDirPage(
    wpSelectTasks,
    'Select your GopherTrunk data folder',
    'Where should GopherTrunk keep your files?',
    'GopherTrunk installs its program to Program Files, but keeps all ' +
    'of YOUR files in a separate data folder you choose here. Setup ' +
    'creates config, recordings, iq, exports, data, logs, and web ' +
    'subfolders under it and seeds a starter config.yaml. The default ' +
    'is your Documents folder so it''s easy to find and back up; you ' +
    'can choose anywhere you can write to — a USB stick, a network ' +
    'drive, or your desktop. An existing config.yaml is never ' +
    'overwritten.',
    False, '');
  DataDirPage.Add('Data folder:');
  DataDirPage.Values[0] :=
    ExpandConstant('{userdocs}\GopherTrunk');
end;

// DataDir is the user-chosen data root. ConfigDir and WebDir return
// the two subpaths the [Files]/[Icons]/[Registry] sections reference
// by name ({code:ConfigDir}, {code:WebDir}).
function DataDir(Param: string): string;
begin
  Result := DataDirPage.Values[0];
end;

function ConfigDir(Param: string): string;
begin
  Result := AddBackslash(DataDirPage.Values[0]) + 'config';
end;

function WebDir(Param: string): string;
begin
  Result := AddBackslash(DataDirPage.Values[0]) + 'web';
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
// the uninstall run, so the install-time data root is read back from
// HKLM\Software\GopherTrunk\Install (populated by the [Registry]
// section).
// ---------------------------------------------------------------

function ReadInstalledDataDir(): string;
begin
  if not RegQueryStringValue(HKEY_LOCAL_MACHINE,
    'Software\GopherTrunk\Install', 'DataDir', Result)
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

// WipeManagedData deletes the Setup-managed parts of the data root —
// config, the SQLite database + caches, logs, and the web consoles —
// while DELIBERATELY preserving the operator's irreplaceable captures
// (recordings, iq, exports). It never removes the data root itself, so
// those preserved folders stay put.
procedure WipeManagedData();
var
  Root: string;
begin
  Root := ReadInstalledDataDir();
  if Root = '' then exit;
  Root := AddBackslash(Root);
  DelTree(Root + 'config', True, True, True);
  DelTree(Root + 'data',   True, True, True);
  DelTree(Root + 'logs',   True, True, True);
  DelTree(Root + 'web',    True, True, True);
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  WipeAnswer: Integer;
begin
  if CurUninstallStep = usUninstall then begin
    // Always strip our PATH entry — the [Registry] section never
    // got a cleanup flag, so this is the only place it happens.
    // The HKCU GOPHERTRUNK_CONFIG / GOPHERTRUNK_HOME values and the
    // Software\GopherTrunk\Install key clean themselves up via
    // uninsdeletevalue / uninsdeletekeyifempty.
    RemoveAppFromHKLMPath();

    WipeAnswer := MsgBox(
      'Also remove your config, database, logs, and web console?' + #13#10 + #13#10 +
      'Yes = delete the config, data, logs, and web folders under your ' +
      'data root. Your captures (recordings, iq, exports) are KEPT.' + #13#10 +
      'No  = preserve everything (recommended).',
      mbConfirmation, MB_YESNO or MB_DEFBUTTON2);
    if WipeAnswer = IDYES then begin
      WipeManagedData();
    end;
  end;
end;
