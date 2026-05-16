---
layout: page
title: Linux install
description: Five-minute path from a fresh download to a working gophertrunk sdr list on Linux
nav_group: Install
---

# Installing GopherTrunk on Linux

Five minutes from a fresh download to a working `gophertrunk sdr list`.
GopherTrunk on Linux is a single static binary — `CGO_ENABLED=0`, no
`librtlsdr`, no `libusb`, no glibc version drama. Anything from a 2019-era
distro forward will run it.

## 1. Download the tarball

Go to the **[GopherTrunk releases page]** and grab the asset matching
your CPU:

```
gophertrunk-<version>-linux-amd64.tar.gz    # Intel / AMD x86_64
gophertrunk-<version>-linux-arm64.tar.gz    # Raspberry Pi 4 / 5, most modern ARM SBCs
```

If you'd rather curl it, see the one-liner under the [downloads page
Linux quick-start]({{ '/downloads.html#linux' | relative_url }}).

[GopherTrunk releases page]: https://github.com/MattCheramie/GopherTrunk/releases

> **Verify the download** against `SHA256SUMS` before installing — see
> the [verify section]({{ '/downloads.html#verify-your-download' | relative_url }})
> on the downloads page for the exact `sha256sum -c` invocation.

## 2. Install the binary

Extract and place `gophertrunk` somewhere on `PATH`. The conventional
spot for a single-binary system service is `/usr/local/bin`:

```sh
tar xzf gophertrunk-<version>-linux-amd64.tar.gz
cd gophertrunk-<version>-linux-amd64
sudo install -m 0755 gophertrunk /usr/local/bin/gophertrunk
```

The tarball also bundles `config.example.yaml`, `README.md`, and
`LICENSE`. We'll come back to the config in step 5.

## 3. Grant USB access (one-time, every host)

Linux ships a kernel DVB driver (`dvb_usb_rtl28xxu`) that grabs RTL-SDR
dongles before user-space can claim them. We need to (a) stop the
kernel from binding the device and (b) let your user open the USB
device node without `sudo`. Both are one-shot config files.

**Blacklist the DVB driver** so it leaves the dongle alone:

```sh
sudo tee /etc/modprobe.d/blacklist-dvb_usb_rtl28xxu.conf <<'EOF'
blacklist dvb_usb_rtl28xxu
EOF
sudo modprobe -r dvb_usb_rtl28xxu 2>/dev/null || true
```

**Add a udev rule** so non-root processes can claim the device:

```sh
sudo tee /etc/udev/rules.d/20-rtlsdr.rules <<'EOF'
SUBSYSTEM=="usb", ATTRS{idVendor}=="0bda", ATTRS{idProduct}=="2838", MODE="0666"
EOF
sudo udevadm control --reload
sudo udevadm trigger
```

Unplug and re-plug the dongle once so the new rule takes effect on a
freshly-enumerated device. You only do this once per host — the rule
covers every RTL-SDR you'll ever plug in.

> **Why not `plugdev` group + `0660`?** That works too, and is more
> conservative if the host has untrusted users. The `MODE="0666"` rule
> above is the simplest path for a single-operator box. Swap to
> `MODE="0660", GROUP="plugdev"` if you'd rather scope access.

## 4. Verify everything works

Open a fresh shell (so the udev change takes effect for your session)
and run:

```sh
gophertrunk version
gophertrunk sdr list
```

`sdr list` should print one line per attached dongle with its driver,
index, serial, product string, and (when populated) tuner + gain
ladder. The plain command only reads USB descriptors, so the TUNER
and gains columns stay blank — pass `--probe` to open each device
just long enough to enumerate them:

```sh
gophertrunk sdr list --probe
```

If you see `no SDR devices found` and the dongle is plugged in:

- Check `lsusb` shows the dongle (typically `0bda:2838` for generic
  RTL-SDR Blog units / NESDR Smart v5).
- Check `lsmod | grep dvb_usb_rtl28xxu` — if it's still loaded, the
  blacklist didn't take. Run `sudo modprobe -r dvb_usb_rtl28xxu` and
  re-plug the dongle.
- Check the udev rule applied: `ls -l /dev/bus/usb/<bus>/<dev>` should
  be world-writable (`crw-rw-rw-`).

See [`hardware.md`]({{ '/hardware.html' | relative_url }}) for the full
matrix of supported tuners and dongles.

## 5. Configure and start the daemon

The tarball includes `config.example.yaml`. Drop a copy at
`~/.config/gophertrunk/config.yaml` and edit it — the daemon walks
`$GOPHERTRUNK_CONFIG` → `~/.config/gophertrunk/config.yaml` →
`~/Documents/GopherTrunk/config.yaml` → `./config.yaml` and loads the
first one it finds, so no `-config` flag is needed:

```sh
mkdir -p ~/.config/gophertrunk
cp config.example.yaml ~/.config/gophertrunk/config.yaml
${EDITOR:-nano} ~/.config/gophertrunk/config.yaml
gophertrunk run
```

On startup the daemon prints `config: loaded <path>` so you can
confirm it picked the right file. Override discovery any time with
`-config <path>` or by exporting `GOPHERTRUNK_CONFIG`.

If you keep more than one config in that directory (e.g.
`config.yaml` + `prod.yaml`), `gophertrunk run` prints a numbered
menu and asks which to load. Non-interactive launches (systemd,
cron) auto-pick the first match with a stderr warning; pin a
specific file via `-config` or `GOPHERTRUNK_CONFIG` for those.

Logs stream to the terminal. Press `Ctrl+C` to stop cleanly.

### Run as a systemd service

For a long-running setup, GopherTrunk ships an example unit at
[`docs/gophertrunk.service`](https://github.com/MattCheramie/GopherTrunk/blob/main/docs/gophertrunk.service)
with `DynamicUser=true` and a tight sandbox profile. Install it
system-wide:

```sh
sudo install -d -m 0755 /etc/gophertrunk
sudo install -m 0640 config.example.yaml /etc/gophertrunk/config.yaml
sudo ${EDITOR:-nano} /etc/gophertrunk/config.yaml      # edit serials, frequencies
sudo curl -L -o /etc/systemd/system/gophertrunk.service \
  https://raw.githubusercontent.com/MattCheramie/GopherTrunk/main/docs/gophertrunk.service
sudo systemctl daemon-reload
sudo systemctl enable --now gophertrunk
journalctl -u gophertrunk -f
```

The unit file's install header walks through every hardening knob
(USB device allow-list, supplementary groups, namespace restrictions)
— read it before deploying to a shared host.

## Uninstall

```sh
sudo systemctl disable --now gophertrunk 2>/dev/null || true
sudo rm -f /etc/systemd/system/gophertrunk.service
sudo rm -f /usr/local/bin/gophertrunk
sudo rm -f /etc/modprobe.d/blacklist-dvb_usb_rtl28xxu.conf
sudo rm -f /etc/udev/rules.d/20-rtlsdr.rules
sudo rm -rf /etc/gophertrunk
sudo systemctl daemon-reload
sudo udevadm control --reload
```

Recordings under your call-log directory (default `/var/lib/gophertrunk`)
are left alone — remove them manually if you want a clean slate.

## Troubleshooting

| Symptom                                 | Likely cause                                       |
| --------------------------------------- | -------------------------------------------------- |
| `command not found: gophertrunk`        | Binary isn't on `PATH` — re-check step 2, or run `/usr/local/bin/gophertrunk` directly. |
| `sdr list` prints nothing               | DVB driver still bound — `sudo modprobe -r dvb_usb_rtl28xxu` and re-plug. |
| `permission denied` on `/dev/bus/usb/…` | udev rule didn't apply — re-run `udevadm control --reload && udevadm trigger`, then re-plug. |
| `usb: device disconnected` mid-stream   | Power-saving USB autosuspend kicked in — `echo on > /sys/bus/usb/devices/<id>/power/control`, or pin via udev. |
| `tuner init: ... I2CWrite addr=0x34: broken pipe` | Tuner not ack-ing on the I²C bus. Most common: DVB kernel driver still bound — `sudo modprobe -r dvb_usb_rtl28xxu` and **physically re-plug** the dongle. Also seen with marginal USB power or USB 3.0 hubs — try a USB 2.0 port directly on the host. If `rtl_test` also prints `[R82XX] PLL not locked!` the tuner is in a half-init state; a full power cycle (unplug ~10s) clears it. |
| Audio plays as silence                  | `audio.enabled: false` by default — set `true` in config; on distroless / Alpine without `libasound2`, set `audio.device: "ioctl"` to use the direct-kernel backend. |
| systemd unit fails with `203/EXEC`      | Binary path wrong in the unit — confirm `/usr/local/bin/gophertrunk` exists and is `+x`. |

For anything else: open an issue at
<https://github.com/MattCheramie/GopherTrunk/issues> with the
`gophertrunk version` output and the first few lines of the daemon log
(`journalctl -u gophertrunk -n 50` if running under systemd).
