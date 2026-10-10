; Inno Setup 7 script for wopr_<version>_windows_setup.exe: one per-user installer that holds both
; Windows builds and installs the one that fits the computer as wopr.exe. build-installer.ps1 compiles
; it, as .github/workflows/windows.yml does, with the version, the folder of the release files
; (dist/release, checked against checksums.txt) and the output folder, as absolute paths:
;
;   ISCC.exe /Qp /DVersion=1.2.3 /DBinDir=C:\...\dist\release /OC:\...\build\installer wopr.iss
;
; test-installer.ps1 installs and removes it. Setup's own pages, messages and command-line switches
; are Inno Setup's: https://jrsoftware.org/ishelp/

#ifndef Version
  #error Pass the version: /DVersion=X.Y.Z (GoReleaser's .Version)
#endif
; The repository's root, for the icons. BinDir and OutputDir default to dist\release and
; build\installer there.
#define RepoDir AddBackslash(SourcePath) + "..\.."
#ifndef BinDir
  #define BinDir RepoDir + "\dist\release"
#endif
#ifndef OutputDir
  #define OutputDir RepoDir + "\build\installer"
#endif
; Windows wants four numbers in a file version: 1.2.3 becomes 1.2.3.0, and a snapshot such as
; 1.2.4-snapshot.abc1234 becomes 1.2.4.0 (its full version is still AppVersion).
#define NumVersion Copy(Version, 1, Pos("-", Version + "-") - 1) + ".0"
; packaging/description.txt's first line, for Settings > Apps and the shortcuts' tooltips.
#define Description "Fan-made terminal recreation of the computer from the film WarGames"

[Setup]
; Never change AppId: upgrades find the installed copy by it, and the uninstall entry is named after it.
AppId={{A7D86110-58E9-404E-9164-FBDDBDC6AC22}
AppName=WOPR
AppVersion={#Version}
AppVerName=WOPR {#Version}
AppPublisher=Christopher Goes
AppPublisherURL=https://github.com/GhostofGoes/WOPR
AppSupportURL=https://github.com/GhostofGoes/WOPR/issues
AppUpdatesURL=https://github.com/GhostofGoes/WOPR/releases
AppCopyright=Copyright (c) 2026 Christopher Goes
AppComments={#Description}
; The version information of setup.exe itself. Company, product name and copyright come from
; AppPublisher, AppName and AppCopyright.
VersionInfoVersion={#NumVersion}
VersionInfoProductVersion={#NumVersion}
VersionInfoProductTextVersion={#Version}
VersionInfoDescription=WOPR Setup
VersionInfoOriginalFileName=wopr_{#Version}_windows_setup.exe
; Settings > Apps shows the name, the version, the publisher and wopr.exe's icon.
UninstallDisplayName=WOPR
UninstallDisplayIcon={app}\wopr.exe

; Per user, with no administrator prompt: the default {autopf} is then
; %LOCALAPPDATA%\Programs\WOPR, the folder the PowerShell install line uses (as "wopr"; Windows
; ignores the case), so installing over that copy replaces it. "commandline" lets an administrator
; pass /ALLUSERS to install into Program Files for everyone, without adding a page that asks.
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=commandline
DefaultDirName={autopf}\WOPR
; Go 1.27 runs on Windows 10 and later. There is no 32-bit build, so Setup refuses 32-bit Windows;
; Arm64 computers get the arm64 build and the rest the amd64 one ([Files]). Setup itself stays
; 32-bit (the default) so it also runs on Windows 10 on Arm, which cannot run x64 programs.
MinVersion=10.0
ArchitecturesAllowed=x64compatible or arm64
ArchitecturesInstallIn64BitMode=x64compatible or arm64

; A short wizard: the tasks (desktop shortcut, PATH), a summary that says where WOPR goes, the
; progress bar, and Finish with "Launch WOPR". There is no licence to accept (MIT asks for none:
; LICENSE.txt and the notices are installed beside the program), no folder to pick (the per-user
; folder needs no rights, and /DIR= still changes it), and no Start menu folder: one shortcut.
DisableWelcomePage=yes
DisableDirPage=yes
AlwaysShowDirOnReadyPage=yes
DisableProgramGroupPage=yes
DisableReadyPage=no
WizardStyle=modern dynamic
SetupIconFile={#RepoDir}\packaging\icons\wopr.ico
; Setup picks the size closest to the corner it draws in, from 58x58 at 100% to 159x159 at 250%.
WizardSmallImageFile={#RepoDir}\packaging\icons\hicolor\64x64\apps\io.github.ghostofgoes.wopr.png,{#RepoDir}\packaging\icons\hicolor\128x128\apps\io.github.ghostofgoes.wopr.png,{#RepoDir}\packaging\icons\hicolor\256x256\apps\io.github.ghostofgoes.wopr.png

; An upgrade is the same installer run again: same AppId, same folder, and the tasks chosen last
; time. A running wopr.exe would block replacing it, so Setup closes it through Restart Manager
; ("force", since wopr keeps nothing that closing it could lose) and does not try to restart it,
; which only works for programs that register for it. wopr creates no mutex, so AppMutex has
; nothing to check.
CloseApplications=force
RestartApplications=no
; One Setup at a time, in case the download is opened twice.
SetupMutex=WOPRSetup-A7D86110-58E9-404E-9164-FBDDBDC6AC22,Global\WOPRSetup-A7D86110-58E9-404E-9164-FBDDBDC6AC22
; The PATH task changes the environment; Explorer and new terminals see it at once.
ChangesEnvironment=yes

OutputDir={#OutputDir}
OutputBaseFilename=wopr_{#Version}_windows_setup
; Solid LZMA2 across both builds: they share much of their data, which saves about 240 KB.
; notimestamp on every file ([Files]) keeps the source files' times out of the installer.
Compression=lzma2/max
SolidCompression=yes

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[CustomMessages]
PathGroup=Terminal:
AddToPath=&Add WOPR to PATH, so typing wopr in a terminal starts it

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "path"; Description: "{cm:AddToPath}"; GroupDescription: "{cm:PathGroup}"

[Dirs]
; Uninstall removes the folder (once empty) even if it was there before Setup ran, as it is after
; the PowerShell install line; by default it removes only folders that Setup made.
Name: "{app}"; Flags: uninsalwaysuninstall

[Files]
; The notices come from the release files too, so that every file in the installer is one that
; checksums.txt lists. LICENSE gets .txt so that it opens with a double-click.
Source: "{#BinDir}\LICENSE"; DestDir: "{app}"; DestName: "LICENSE.txt"; Flags: ignoreversion notimestamp
Source: "{#BinDir}\NOTICE.md"; DestDir: "{app}"; Flags: ignoreversion notimestamp
Source: "{#BinDir}\THIRD_PARTY_NOTICES.txt"; DestDir: "{app}"; Flags: ignoreversion notimestamp
Source: "{#BinDir}\README.md"; DestDir: "{app}"; Flags: ignoreversion notimestamp
Source: "{#BinDir}\wopr_{#Version}_windows_amd64.exe"; DestDir: "{app}"; DestName: "wopr.exe"; Check: not IsArm64; Flags: ignoreversion notimestamp
Source: "{#BinDir}\wopr_{#Version}_windows_arm64.exe"; DestDir: "{app}"; DestName: "wopr.exe"; Check: IsArm64; Flags: ignoreversion notimestamp

[UninstallDelete]
; The debug log that WOPR_DEBUG turns on (internal/debuglog), in os.UserCacheDir(), which is
; %LOCALAPPDATA% on Windows. Only the log and then its folder, if nothing else is in it.
Type: files; Name: "{localappdata}\wopr\debug.log"
Type: dirifempty; Name: "{localappdata}\wopr"

[Icons]
; One Start menu entry at the top level of All apps. wopr.exe is a console program, so Windows opens
; it in the default terminal: Windows Terminal on Windows 11, the console window on Windows 10.
; wopr reads and writes no files in its working directory; {app} exists for every user, also after
; an /ALLUSERS install.
Name: "{autoprograms}\WOPR"; Filename: "{app}\wopr.exe"; WorkingDir: "{app}"; Comment: "{#Description}"
Name: "{autodesktop}\WOPR"; Filename: "{app}\wopr.exe"; WorkingDir: "{app}"; Comment: "{#Description}"; Tasks: desktopicon

[Run]
Filename: "{app}\wopr.exe"; WorkingDir: "{app}"; Description: "{cm:LaunchProgram,WOPR}"; Flags: nowait postinstall skipifsilent

[Code]
// The "path" task adds the WOPR folder to PATH: the user's, or the system's after an /ALLUSERS
// install (HKEY_AUTO is HKLM or HKCU to match). Uninstall takes the folder off PATH again, also
// when the PowerShell install line put it there, since the folder is then gone. Path is read and
// written without expanding it, so entries such as %USERPROFILE%\bin stay as they are, and it stays
// REG_EXPAND_SZ. Entries match the way Windows matches them: ignoring case and a trailing backslash.

function EnvironmentKey: String;
begin
  if IsAdminInstallMode then
    Result := 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment'
  else
    Result := 'Environment';
end;

function SameDir(A, B: String): Boolean;
begin
  Result := CompareText(RemoveBackslashUnlessRoot(Trim(A)), RemoveBackslashUnlessRoot(Trim(B))) = 0;
end;

// PathWithout returns Path without the entries that name Dir; every other entry, empty ones
// included, keeps its place, so a Path without Dir comes back unchanged.
function PathWithout(Path, Dir: String): String;
var
  Rest, Entry: String;
  I: Integer;
  First: Boolean;
begin
  Result := '';
  First := True;
  Rest := Path + ';';
  while Rest <> '' do begin
    I := Pos(';', Rest);
    Entry := Copy(Rest, 1, I - 1);
    Delete(Rest, 1, I);
    if not SameDir(Entry, Dir) then begin
      if not First then
        Result := Result + ';';
      Result := Result + Entry;
      First := False;
    end;
  end;
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  Path, Dir: String;
begin
  if (CurStep <> ssPostInstall) or not WizardIsTaskSelected('path') then
    Exit;
  Dir := ExpandConstant('{app}');
  if not RegQueryStringValue(HKEY_AUTO, EnvironmentKey, 'Path', Path) then
    Path := '';
  if PathWithout(Path, Dir) <> Path then begin
    Log('PATH already has ' + Dir);
    Exit;
  end;
  if (Path <> '') and (Copy(Path, Length(Path), 1) <> ';') then
    Path := Path + ';';
  if RegWriteExpandStringValue(HKEY_AUTO, EnvironmentKey, 'Path', Path + Dir) then
    Log('Added ' + Dir + ' to PATH')
  else
    SuppressibleMsgBox('Setup could not add ' + Dir + ' to PATH. WOPR is installed; start it from the Start menu.', mbError, MB_OK, IDOK);
end;

procedure RemoveFromPath;
var
  Path, NewPath: String;
begin
  if not RegQueryStringValue(HKEY_AUTO, EnvironmentKey, 'Path', Path) then
    Exit;
  NewPath := PathWithout(Path, ExpandConstant('{app}'));
  if NewPath = Path then
    Exit;
  if NewPath = '' then
    RegDeleteValue(HKEY_AUTO, EnvironmentKey, 'Path')
  else
    RegWriteExpandStringValue(HKEY_AUTO, EnvironmentKey, 'Path', NewPath);
  Log('Removed ' + ExpandConstant('{app}') + ' from PATH');
end;

// Windows may still have the Start menu shortcut open when Uninstall deletes it: the shell and the
// search indexer read a new shortcut to list it. Uninstall tries each file once ("Failed to delete
// the file; it may be in use (32)", seen on windows-11-arm), which would leave WOPR in All apps,
// pointing at nothing. So it is tried again for up to 10 seconds. The desktop shortcut is left to
// Uninstall alone: a WOPR.lnk there may be the user's own, when the desktop icon task was off.
procedure DeleteStartMenuShortcut;
var
  Shortcut: String;
  I: Integer;
begin
  Shortcut := ExpandConstant('{autoprograms}\WOPR.lnk');
  for I := 1 to 40 do begin
    if not FileExists(Shortcut) then
      Exit;
    if DeleteFile(Shortcut) then begin
      Log('Deleted ' + Shortcut + ' (try ' + IntToStr(I) + ')');
      Exit;
    end;
    Sleep(250);
  end;
  Log('Could not delete ' + Shortcut);
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  case CurUninstallStep of
    // usUninstall comes before Uninstall tells running programs that the environment changed.
    usUninstall: RemoveFromPath;
    // usPostUninstall comes after Uninstall has tried to delete every file it installed.
    usPostUninstall: DeleteStartMenuShortcut;
  end;
end;
