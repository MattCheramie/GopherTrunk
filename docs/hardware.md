# Hardware Setup

GopherTrunk ships with a CGO binding to librtlsdr. Building the daemon
requires `librtlsdr-dev` and `libusb-1.0-0-dev` on the host.

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
