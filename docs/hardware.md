# Hardware Setup

GopherTrunk ships with a CGO binding to librtlsdr. Building the daemon
requires `librtlsdr-dev` and `libusb-1.0-0-dev` on the host.

## Supported devices

Anything librtlsdr supports — RTL2832U + R820T / R820T2 / R828D / E4000 /
FC0012 / FC0013 / FC2580 — works through a single code path. There's
no per-tuner or per-vendor branching in the daemon.

Tested combinations:

| Device | Tuner | Notes |
| --- | --- | --- |
| **NooElec NESDR Smart v5** | R820T2 | 0.5 ppm TCXO, software-controllable bias-tee. Use `bias_tee: true` in config to power an external LNA via the SMA. |
| NooElec NESDR Smart (v4 and earlier) | R820T2 | TCXO; no bias-tee on early units. |
| Generic RTL-SDR Blog v3 / v4 | R820T2 / R828D | Bias-tee on most units. |
| Plain RTL2832U DVB-T sticks | R820T | No TCXO; expect a few ppm offset — set `ppm:` in config after measuring. |

If you have a v5 (or any modern dongle with a bias-tee) and want to
power an LNA, the config snippet looks like:

```yaml
sdr:
  devices:
    - serial: "00000001"      # whatever `gophertrunk sdr list` shows
      role: control            # or voice / auto
      ppm: 0                   # 0 is fine for TCXO-equipped units
      gain: "auto"             # or a numeric tenths-of-dB string like "496"
      bias_tee: true           # 5V on the SMA — only enable if you want it
```

## Linux

```sh
sudo apt-get install librtlsdr-dev libusb-1.0-0-dev
```

Add a udev rule so non-root processes can claim the dongle:

```
# /etc/udev/rules.d/20-rtlsdr.rules
SUBSYSTEM=="usb", ATTRS{idVendor}=="0bda", ATTRS{idProduct}=="2838", MODE="0666"
```

Reload udev (`sudo udevadm control --reload && sudo udevadm trigger`) and
unplug/replug the dongle.

Blacklist the kernel's DVB driver, which otherwise grabs the device first:

```
# /etc/modprobe.d/blacklist-dvb_usb_rtl28xxu.conf
blacklist dvb_usb_rtl28xxu
```

## Verifying the build

```sh
make build
./bin/gophertrunk sdr list
```

You should see one row per attached dongle with index, serial, tuner type
(usually `R820T` or `R828D`), and the supported gain values.

## Capturing IQ for replay

`librtlsdr` ships `rtl_sdr` for raw captures:

```sh
rtl_sdr -f 851000000 -s 2400000 -g 49.6 -n 24000000 cc.cfile
```

Drop `.cfile` files under `testdata/iq/` to use them with the mock driver.
