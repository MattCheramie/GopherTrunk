package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"text/tabwriter"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/hackrf"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/purego"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// doctorDevice is one VID/PID the doctor inspects for driver binding.
type doctorDevice struct {
	VID  uint16
	PID  uint16
	Name string
}

// knownDoctorDevices is the union of every pure-Go USB SDR's VID/PID
// table the doctor knows how to diagnose: RTL-SDR plus HackRF. Windows
// binds a function driver per VID/PID at plug time, so the doctor has to
// check each identity to tell the operator whether WinUSB is bound. A
// HackRF that appears in `sdr list` but fails to open shows up here with
// the wrong-driver hint instead of being invisible to the doctor.
func knownDoctorDevices() []doctorDevice {
	var out []doctorDevice
	for _, d := range purego.KnownVIDPIDs() {
		out = append(out, doctorDevice{VID: d.VID, PID: d.PID, Name: d.Name})
	}
	for _, d := range hackrf.KnownVIDPIDs() {
		out = append(out, doctorDevice{VID: d.VID, PID: d.PID, Name: d.Name})
	}
	return out
}

// runSDRDoctor implements `gophertrunk sdr doctor`. It iterates the known
// USB SDR VID/PID list (RTL-SDR + HackRF), asks the platform inspector
// which kernel/Windows function driver is bound to each matching dongle,
// and prints a row per device with an actionable next step. Read-only:
// never opens or claims a USB device, so safe to run as a regular
// user alongside a live daemon.
func runSDRDoctor(args []string) {
	fs := flag.NewFlagSet("sdr doctor", flag.ExitOnError)
	verbose := fs.Bool("v", false, "include extra columns (driver description)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	inspector := usb.DefaultDriverInspector()
	var rows []usb.DriverBinding
	var failures []error
	seen := make(map[string]bool)
	for _, pair := range knownDoctorDevices() {
		bindings, err := inspector.Inspect(pair.VID, pair.PID)
		if err != nil {
			if errors.Is(err, usb.ErrUnsupportedPlatform) {
				fmt.Fprintf(os.Stderr, "sdr doctor: no driver inspector for %s/%s\n", runtime.GOOS, runtime.GOARCH)
				return
			}
			failures = append(failures, fmt.Errorf("inspect %04x:%04x: %w", pair.VID, pair.PID, err))
			continue
		}
		for _, b := range bindings {
			key := fmt.Sprintf("%04x:%04x:%s", b.Descriptor.VID, b.Descriptor.PID, b.Descriptor.Path)
			if seen[key] {
				continue
			}
			seen[key] = true
			rows = append(rows, b)
		}
	}

	for _, err := range failures {
		fmt.Fprintln(os.Stderr, "sdr doctor:", err)
	}

	if len(rows) == 0 {
		fmt.Println("No USB SDR dongles found (checked RTL-SDR and HackRF).")
		fmt.Println("If a dongle is plugged in but missing here:")
		switch runtime.GOOS {
		case "windows":
			fmt.Println("  - Open Device Manager and confirm the dongle appears (look for an exclamation mark).")
			fmt.Println("  - Re-plug into a different USB port (preferably USB 2.0 for first-time bring-up).")
			fmt.Println("  - Run Zadig from the Start Menu and verify the dongle is listed with Options → List All Devices.")
		case "linux":
			fmt.Println("  - Run `lsusb` and confirm the dongle's VID:PID appears (RTL-SDR is typically 0bda:2832 or 0bda:2838; HackRF is 1d50:6089).")
			fmt.Println("  - Confirm /sys/bus/usb/devices exists and your user has read permission.")
		}
		return
	}

	printDoctorRows(rows, *verbose)
}

// printDoctorRows writes a tab-aligned per-dongle status table to
// stdout. Mirrors listSDRs's column widths so operators see a
// consistent layout across `sdr list` and `sdr doctor`.
func printDoctorRows(rows []usb.DriverBinding, verbose bool) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if verbose {
		fmt.Fprintln(tw, "VID:PID\tSERIAL\tDRIVER\tEXPECTED\tSTATUS\tDESCRIPTION\tNEXT-STEP")
	} else {
		fmt.Fprintln(tw, "VID:PID\tSERIAL\tDRIVER\tEXPECTED\tSTATUS\tNEXT-STEP")
	}
	for _, r := range rows {
		status := "BAD"
		if r.OK {
			status = "OK"
		}
		driver := r.DriverName
		if driver == "" {
			driver = "(none)"
		}
		hint := r.Hint
		if hint == "" {
			hint = "-"
		}
		vidpid := fmt.Sprintf("%04x:%04x", r.Descriptor.VID, r.Descriptor.PID)
		serial := r.Descriptor.Serial
		if serial == "" {
			serial = "-"
		}
		if verbose {
			descr := r.DriverDesc
			if descr == "" {
				descr = "-"
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				vidpid, serial, driver, r.Expected, status, descr, hint)
		} else {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
				vidpid, serial, driver, r.Expected, status, hint)
		}
	}
	tw.Flush()
}
