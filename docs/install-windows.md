---
layout: page
title: Windows install
description: Five-minute path from a fresh download to a working gophertrunk sdr list on Windows 11
nav_group: Install
---

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

- Copies `gophertrunk.exe` to `C:\Program Files\GopherTrunk\` —
  a single static binary, no DLLs to ship.
- Bundles **Zadig** (the WinUSB driver-binding tool) next to the
  daemon and adds a Start Menu shortcut "Install RTL-SDR driver
  (Zadig)" so you don't have to chase a separate download.
- Adds Start Menu entries for the daemon, the config template,
  and these instructions.
- Installs the **browser-based web operator console** (a static
  HTML / JS folder you open in any browser) to a location you
  pick — the wizard offers a "Select web operator console
  location" page after the Tasks step, defaulting to
  `%USERPROFILE%\Documents\GopherTrunk Web Console`. Untick the
  "Install the web operator console" checkbox on the Tasks page
  to skip it (e.g. for a headless server install). Setup +
  quick-start guide for the console: **[Web console]({{ '/web.html' | relative_url }})**.
- Optionally adds `C:\Program Files\GopherTrunk` to your system
  PATH so you can run `gophertrunk` from any PowerShell window
  (off by default — tick the "Add GopherTrunk to my PATH"
  checkbox during install if you want it).

When the wizard finishes, it'll offer to open this document, a
console window, Zadig (to bind the WinUSB driver — see §3), and
(if you installed the web console) the console itself in your
default browser. All four are harmless to skip; the Zadig one
defaults off.

## 3. Install the WinUSB driver via Zadig (one-time, for each dongle)

Windows ships an RTL-SDR DVB-T receiver driver by default — that
driver is what you'd use to watch broadcast TV, and it's the wrong
driver for SDR work. We need to swap it for **WinUSB** on a
per-device basis. The installer **bundles Zadig** (GPL-3.0, from
<https://zadig.akeo.ie>) so you don't have to chase a download.

1. Plug in the RTL-SDR dongle. The same flow works for any
   `0bda:2838` device — generic RTL-SDR Blog units, the **NooElec
   NESDR Smart v5**, and equivalent clones.
2. Launch Zadig via **Start Menu → GopherTrunk → "Install RTL-SDR
   driver (Zadig)"**. Approve the UAC prompt. (Alternatively, on the
   last page of the installer there's an unchecked **"Run Zadig now
   to bind the WinUSB driver"** option — tick it before clicking
   Finish.)
3. **Options → List All Devices** so the RTL-SDR shows up.
4. From the dropdown, pick the dongle. It'll typically appear as
   **Bulk-In, Interface (Interface 0)** or **RTL2832U** (the
   NESDR Smart v5 reports as `RTL2838UHIDIR`).
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
driver, index, serial, and product string. The TUNER and gains
columns are blank by design — they need the device to be opened
(USB claim + RTL2832U baseband init + I2C tuner probe), which `sdr
list` skips so the command is fast and never collides with a
running daemon. Pass `--probe` when you want those columns filled:

```powershell
gophertrunk sdr list --probe
```

If you see `no SDR devices found` and you're sure the dongle is
plugged in, the WinUSB driver swap probably didn't take — re-run
Zadig with **Options → List All Devices** checked and verify the
"Driver" column shows **WinUSB** for your dongle.

If you didn't tick the PATH option during install, run from the
install folder instead:

```powershell
cd "C:\Program Files\GopherTrunk"
.\gophertrunk.exe sdr list
```

## 5. Configure and start the daemon

The installer asked you for an "editable files folder" (default
`Documents\GopherTrunk`) and seeded a `config.yaml` there. It also
set the `GOPHERTRUNK_CONFIG` user environment variable to point at
that file, so the daemon discovers it automatically — no `-config`
flag needed. Use the Start Menu shortcut "Edit my config.yaml
(Notepad)" to open it, set your device serial + control-channel
frequencies, and save.

Then start the daemon:

```powershell
gophertrunk run
```

The daemon prints `config: loaded <path>` on startup so you can
confirm it picked up the right file. To override (e.g. running
against a second config for testing), use `-config`:

```powershell
gophertrunk run -config "C:\path\to\other.yaml"
```

If you drop multiple `*.yaml` files into the editable-files folder
(e.g. `config.yaml` + `prod.yaml` + `test.yaml`), the daemon prints
a numbered menu on startup and asks which one to load. Pick the
number, press Enter, and that file is used. Set `-config` or
`GOPHERTRUNK_CONFIG` to skip the prompt for unattended runs (a
non-interactive launch — Windows service, scheduled task —
auto-selects the first file and logs the choice).

A read-only reference copy of the full annotated template lives at
`C:\Program Files\GopherTrunk\config.example.yaml` (Start Menu →
"Configuration template").

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
entry, strips GopherTrunk from your PATH (if you opted in), and
clears the `GOPHERTRUNK_CONFIG` env var. It then asks whether to
also delete your editable `config.yaml` and the `gophertrunk-web`
folder Setup created — the default is **No** so you don't lose
edits or per-system data.

## Troubleshooting

| Symptom                                | Likely cause                                       |
| -------------------------------------- | -------------------------------------------------- |
| `gophertrunk` not recognised           | PATH wasn't added — open a fresh terminal or run from `C:\Program Files\GopherTrunk` directly. |
| `sdr list` prints nothing              | Zadig WinUSB swap didn't take — see step 3.        |
| `usb: device disconnected` mid-stream  | The DVB driver re-attached itself — re-run Zadig. |
| `WinUsb_Initialize` fails              | The dongle is bound to the wrong driver — re-run Zadig and pick **WinUSB**. |
| Smart Screen blocks the installer      | Right-click → Properties → Unblock, or **More info → Run anyway**. |

For anything else: open an issue at
<https://github.com/MattCheramie/GopherTrunk/issues> with the
`gophertrunk version` output and the first few lines of the
daemon log.
