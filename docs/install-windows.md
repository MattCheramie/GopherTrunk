# Installing GopherTrunk on Windows 11

This is the path the prebuilt installer lays down for you. Five
minutes from a fresh download to a working `gophertrunk sdr list`.

## 1. Download the installer

Go to the **[GopherTrunk releases page]** and download the
asset named:

```
gophertrunk-<version>-windows-amd64-setup.exe
```

If you'd rather skip the installer and run from a folder, grab the
matching `gophertrunk-<version>-windows-amd64.zip` and extract it
anywhere — the contents are the same.

[GopherTrunk releases page]: https://github.com/MattCheramie/GopherTrunk/releases

> **Why is the download blocked?** Windows SmartScreen sometimes
> warns on installers it hasn't seen before. Click **More info →
> Run anyway**, or right-click the file → Properties → check
> **Unblock**. The installer is unsigned today; signing is on the
> roadmap.

## 2. Run the installer

Double-click `setup.exe` and accept the defaults. The installer:

- Copies `gophertrunk.exe` to `C:\Program Files\GopherTrunk\`
  alongside the librtlsdr / libusb runtime DLLs.
- Adds Start Menu entries for the daemon, the config template,
  and these instructions.
- Optionally adds `C:\Program Files\GopherTrunk` to your system
  PATH so you can run `gophertrunk` from any PowerShell window
  (off by default — tick the "Add GopherTrunk to my PATH"
  checkbox during install if you want it).

When the wizard finishes, it'll offer to open this document and a
console window. Both are harmless to skip.

## 3. Install the WinUSB driver via Zadig (one-time, for each dongle)

Windows ships an RTL-SDR DVB-T receiver driver by default — that
driver is what you'd use to watch broadcast TV, and it's the wrong
driver for SDR work. We need to swap it for **WinUSB** on a
per-device basis. The standard tool is **Zadig**:

1. Plug in the RTL-SDR dongle.
2. Download Zadig from <https://zadig.akeo.ie> (single .exe, no
   install). Run it as **Administrator**.
3. **Options → List All Devices** so the RTL-SDR shows up.
4. From the dropdown, pick the dongle. It'll typically appear as
   **Bulk-In, Interface (Interface 0)** or **RTL2832U**.
5. With **WinUSB** selected as the target driver, click
   **Replace Driver** (or **Install Driver**, first time).
6. Wait ~10 seconds for the success dialog.

You only do this once per dongle. If you ever want the dongle to
work as a TV tuner again, run Zadig in reverse and pick the
manufacturer driver.

## 4. Verify everything works

Open **Windows Terminal** (or PowerShell) and run:

```powershell
gophertrunk version
gophertrunk sdr list
```

`sdr list` should print one line per attached dongle with its
driver, index, serial, tuner, product string, and the gain settings
the tuner exposes. If you see `no SDR devices found` and you're
sure the dongle is plugged in, the WinUSB driver swap probably
didn't take — re-run Zadig with **Options → List All Devices**
checked and verify the "Driver" column shows **WinUSB** for your
dongle.

If you didn't tick the PATH option during install, run from the
install folder instead:

```powershell
cd "C:\Program Files\GopherTrunk"
.\gophertrunk.exe sdr list
```

## 5. Configure and start the daemon

The installer drops a starter config at:

```
C:\Program Files\GopherTrunk\config.example.yaml
```

Copy it somewhere writable — your home directory is fine — and
edit the device serial + control-channel frequencies:

```powershell
cp "C:\Program Files\GopherTrunk\config.example.yaml" "$HOME\gophertrunk.yaml"
notepad "$HOME\gophertrunk.yaml"
```

Then run the daemon against it:

```powershell
gophertrunk run -config "$HOME\gophertrunk.yaml"
```

Logs stream to the terminal. Press `Ctrl+C` to stop cleanly.

For a long-running setup, register GopherTrunk as a Windows service
with [NSSM](https://nssm.cc) — that's the simplest path until a
native service manifest ships:

```powershell
nssm install GopherTrunk "C:\Program Files\GopherTrunk\gophertrunk.exe" `
  run -config "C:\ProgramData\GopherTrunk\config.yaml"
nssm set GopherTrunk AppDirectory "C:\Program Files\GopherTrunk"
nssm start GopherTrunk
```

## Uninstall

**Settings → Apps → Installed apps → GopherTrunk → Uninstall.**
The uninstaller removes the install folder and every Start Menu
entry, and undoes the PATH change if you opted in. Recordings
under your call-log directory are left alone.

## Troubleshooting

| Symptom                                | Likely cause                                       |
| -------------------------------------- | -------------------------------------------------- |
| `gophertrunk` not recognised           | PATH wasn't added — open a fresh terminal or run from `C:\Program Files\GopherTrunk` directly. |
| `sdr list` prints nothing              | Zadig WinUSB swap didn't take — see step 3.        |
| `LIBUSB_ERROR_ACCESS` on stream        | The DVB driver re-attached itself — re-run Zadig. |
| `librtlsdr.dll was not found`          | The installer's DLL bundle was tampered with — reinstall. |
| Smart Screen blocks the installer      | Right-click → Properties → Unblock, or **More info → Run anyway**. |

For anything else: open an issue at
<https://github.com/MattCheramie/GopherTrunk/issues> with the
`gophertrunk version` output and the first few lines of the
daemon log.
