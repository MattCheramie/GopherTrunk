---
layout: page
title: macOS install
description: Five-minute path from a fresh download to a working gophertrunk sdr list on macOS
nav_group: Install
---

# Installing GopherTrunk on macOS

Five minutes from a fresh download to a working `gophertrunk sdr list`.
GopherTrunk on macOS is a single static binary that talks to RTL-SDR
dongles through IOKit — no kext, no `librtlsdr`, no Homebrew formula
to chase.

## 1. Download the tarball

Go to the **[GopherTrunk releases page]** and grab the asset matching
your CPU:

```
gophertrunk-<version>-darwin-arm64.tar.gz    # Apple Silicon (M1 / M2 / M3 / M4)
gophertrunk-<version>-darwin-amd64.tar.gz    # Intel Macs
```

If you'd rather curl it, see the one-liner under the [downloads page
macOS quick-start]({{ '/downloads.html#macos' | relative_url }}).

[GopherTrunk releases page]: https://github.com/MattCheramie/GopherTrunk/releases

> **Verify the download** against `SHA256SUMS` before installing — see
> the [verify section]({{ '/downloads.html#verify-your-download' | relative_url }})
> on the downloads page for the exact `shasum -a 256 -c` invocation.

## 2. Install the binary

Extract and place `gophertrunk` somewhere on `PATH`. The conventional
spot for a single-binary command-line tool on macOS is
`/usr/local/bin` (Intel) or `/opt/homebrew/bin` (Apple Silicon, if you
use Homebrew); either works:

```sh
tar xzf gophertrunk-<version>-darwin-arm64.tar.gz
cd gophertrunk-<version>-darwin-arm64
sudo install -m 0755 gophertrunk /usr/local/bin/gophertrunk
```

The tarball also bundles `config.example.yaml`, `README.md`, and
`LICENSE`. We'll come back to the config in step 5.

## 3. Clear the Gatekeeper quarantine (one-time, every download)

Builds are unsigned — the first time you run an un-signed,
quarantined binary, macOS will refuse with "cannot be opened because
the developer cannot be verified." Two ways to clear it:

**Easy:** right-click `gophertrunk` in Finder → **Open** → confirm in
the dialog. macOS remembers your approval for that binary.

**CLI:** strip the quarantine xattr directly:

```sh
sudo xattr -dr com.apple.quarantine /usr/local/bin/gophertrunk
```

Re-do this every time you upgrade — a fresh download re-attaches the
`com.apple.quarantine` xattr.

> **No driver swap needed.** Unlike Windows (Zadig → WinUSB) and Linux
> (DVB blacklist + udev rule), macOS lets user-space claim USB
> devices via IOKit without rebinding the kernel driver. Plug in the
> dongle and go.

## 4. Verify everything works

Open Terminal and run:

```sh
gophertrunk version
gophertrunk sdr list
```

`sdr list` should print one line per attached dongle with its driver,
index, serial, tuner, product string, and the gain settings the tuner
exposes. If you see `no SDR devices found` and the dongle is plugged
in:

- Check `system_profiler SPUSBDataType | grep -A 4 RTL2838` shows the
  dongle (typically `0x0bda:0x2838`).
- If you have **SDR++**, **GQRX**, or another RTL-SDR app open, close
  it — IOKit hands the device to whoever claims it first.
- Some USB-C hubs power-cycle low-power devices aggressively; plug
  the dongle directly into the Mac or into a powered hub.

See [`hardware.md`]({{ '/hardware.html' | relative_url }}) for the full
matrix of supported tuners and dongles.

## 5. Configure and start the daemon

The tarball includes `config.example.yaml`. Copy it to a writable
location and edit the device serial + control-channel frequencies:

```sh
mkdir -p ~/.config/gophertrunk
cp config.example.yaml ~/.config/gophertrunk/config.yaml
${EDITOR:-nano} ~/.config/gophertrunk/config.yaml
```

Then run the daemon against it:

```sh
gophertrunk run -config ~/.config/gophertrunk/config.yaml
```

Logs stream to the terminal. Press `Ctrl+C` to stop cleanly.

### Run as a launchd service

For a long-running setup, register GopherTrunk as a per-user
`LaunchAgent` so it starts at login and respawns on crash. Drop the
following at `~/Library/LaunchAgents/org.gophertrunk.daemon.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>org.gophertrunk.daemon</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/gophertrunk</string>
    <string>run</string>
    <string>-config</string>
    <string>/Users/YOUR_USER/.config/gophertrunk/config.yaml</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>/Users/YOUR_USER/Library/Logs/gophertrunk.log</string>
  <key>StandardErrorPath</key>
  <string>/Users/YOUR_USER/Library/Logs/gophertrunk.err.log</string>
</dict>
</plist>
```

Then load it:

```sh
launchctl load ~/Library/LaunchAgents/org.gophertrunk.daemon.plist
launchctl start org.gophertrunk.daemon
tail -f ~/Library/Logs/gophertrunk.log
```

For a system-wide daemon (starts at boot, not login), put the plist
under `/Library/LaunchDaemons/` and `sudo launchctl bootstrap
system/<path>` instead. Note that LaunchDaemons run as `root` by
default — set `UserName` in the plist to drop privileges.

## Uninstall

```sh
launchctl unload ~/Library/LaunchAgents/org.gophertrunk.daemon.plist 2>/dev/null || true
rm -f ~/Library/LaunchAgents/org.gophertrunk.daemon.plist
sudo rm -f /usr/local/bin/gophertrunk
rm -rf ~/.config/gophertrunk
rm -f ~/Library/Logs/gophertrunk.log ~/Library/Logs/gophertrunk.err.log
```

Recordings under your call-log directory are left alone — remove them
manually if you want a clean slate.

## Troubleshooting

| Symptom                                                  | Likely cause                                                                  |
| -------------------------------------------------------- | ----------------------------------------------------------------------------- |
| `cannot be opened because the developer cannot be verified` | Quarantine xattr still attached — re-run the `xattr -dr` from step 3, or right-click → Open. |
| `command not found: gophertrunk`                         | Binary isn't on `PATH` — re-check step 2, or run from the install path directly. |
| `sdr list` prints nothing                                | Another RTL-SDR app (SDR++, GQRX) is holding the device — close it and retry. |
| `usb: claim interface failed`                            | Same — IOKit hands the dongle to whoever opened it first.                     |
| Audio plays as silence                                   | `audio.enabled: false` by default — set `true` in config. CoreAudio is the default backend on Darwin. |
| LaunchAgent says `Load failed: 5: Input/output error`    | plist syntax error — `plutil -lint ~/Library/LaunchAgents/org.gophertrunk.daemon.plist` will show the line. |

For anything else: open an issue at
<https://github.com/MattCheramie/GopherTrunk/issues> with the
`gophertrunk version` output and the first few lines of the daemon
log (`tail ~/Library/Logs/gophertrunk.err.log` if running under
launchd).
