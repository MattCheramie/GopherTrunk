// Package purego is the pure-Go RTL-SDR driver — the [sdr.Device] /
// [sdr.Driver] implementation that composes the platform USB transport
// (internal/sdr/rtlsdr/usb), the RTL2832U register layer
// (internal/sdr/rtlsdr/rtl2832u), and the per-chip tuner drivers
// (internal/sdr/rtlsdr/tuners). It is the consumer-facing layer of
// the librtlsdr → pure-Go rewrite and the only RTL-SDR backend the
// project ships — PR-09 removed the legacy CGO librtlsdr wrapper.
package purego

// knownDevice is one row of the VID/PID table librtlsdr maintains in
// src/librtlsdr.c known_devices. We claim a USB device only if it
// matches a row in this table — this keeps random USB devices the
// kernel would let us open from being mistaken for SDR dongles.
type knownDevice struct {
	VID, PID uint16
	Name     string // friendly product name for sdr.Info.Product
}

// knownDevices mirrors librtlsdr's known_devices verbatim; ordering
// is preserved for cross-referencing against the C source. The entries
// cover OEM rebrands of the Realtek RTL2832U reference board (Terratec,
// Compro, NooElec, etc.) — modern dongles from RTL-SDR.com / NooElec
// use 0x0bda:0x2832 or 0x0bda:0x2838.
var knownDevices = []knownDevice{
	{VID: 0x0bda, PID: 0x2832, Name: "Generic RTL2832U"},
	{VID: 0x0bda, PID: 0x2838, Name: "Generic RTL2832U OEM"},
	{VID: 0x0413, PID: 0x6680, Name: "DigitalNow Quad DVB-T PCI-E"},
	{VID: 0x0413, PID: 0x6f0f, Name: "Leadtek WinFast DTV Dongle Mini D"},
	{VID: 0x0458, PID: 0x707f, Name: "Genius TVGo DVB-T03 USB"},
	{VID: 0x0ccd, PID: 0x00a9, Name: "Terratec Cinergy T Stick Black"},
	{VID: 0x0ccd, PID: 0x00b3, Name: "Terratec NOXON DAB/DAB+ USB v1"},
	{VID: 0x0ccd, PID: 0x00b4, Name: "Terratec Deutschlandradio DAB Stick"},
	{VID: 0x0ccd, PID: 0x00b5, Name: "Terratec NOXON DAB Stick - Radio Energy"},
	{VID: 0x0ccd, PID: 0x00b7, Name: "Terratec Media Broadcast DAB Stick"},
	{VID: 0x0ccd, PID: 0x00b8, Name: "Terratec BR DAB Stick"},
	{VID: 0x0ccd, PID: 0x00b9, Name: "Terratec WDR DAB Stick"},
	{VID: 0x0ccd, PID: 0x00c0, Name: "Terratec MuellerVerlag DAB Stick"},
	{VID: 0x0ccd, PID: 0x00c6, Name: "Terratec Fraunhofer DAB Stick"},
	{VID: 0x0ccd, PID: 0x00d3, Name: "Terratec Cinergy T Stick RC (Rev. 3)"},
	{VID: 0x0ccd, PID: 0x00d7, Name: "Terratec T Stick PLUS"},
	{VID: 0x0ccd, PID: 0x00e0, Name: "Terratec NOXON DAB/DAB+ USB v2"},
	{VID: 0x1554, PID: 0x5020, Name: "PixelView PV-DT235U(RN)"},
	{VID: 0x15f4, PID: 0x0131, Name: "Astrometa DVB-T/DVB-T2"},
	{VID: 0x15f4, PID: 0x0133, Name: "HanfTek DAB+FM+DVB-T"},
	{VID: 0x185b, PID: 0x0620, Name: "Compro Videomate U620F"},
	{VID: 0x185b, PID: 0x0650, Name: "Compro Videomate U650F"},
	{VID: 0x185b, PID: 0x0680, Name: "Compro Videomate U680F"},
	{VID: 0x1b80, PID: 0xd393, Name: "GIGABYTE GT-U7300"},
	{VID: 0x1b80, PID: 0xd394, Name: "DIKOM USB-DVBT HD"},
	{VID: 0x1b80, PID: 0xd395, Name: "Peak 102569AGPK"},
	{VID: 0x1b80, PID: 0xd397, Name: "KWorld KW-UB450-T USB DVB-T Pico TV"},
	{VID: 0x1b80, PID: 0xd398, Name: "Zaapa ZT-MINDVBZP"},
	{VID: 0x1b80, PID: 0xd39d, Name: "SVEON STV20 DVB-T USB & FM"},
	{VID: 0x1b80, PID: 0xd3a4, Name: "Twintech UT-40"},
	{VID: 0x1b80, PID: 0xd3a8, Name: "ASUS U3100MINI_PLUS_V2"},
	{VID: 0x1b80, PID: 0xd3af, Name: "SVEON STV27 DVB-T USB & FM"},
	{VID: 0x1b80, PID: 0xd3b0, Name: "SVEON STV21 DVB-T USB & FM"},
	{VID: 0x1d19, PID: 0x1101, Name: "Dexatek DK DVB-T Dongle (Logilink VG0002A)"},
	{VID: 0x1d19, PID: 0x1102, Name: "Dexatek DK DVB-T Dongle (MSI DigiVox mini II V3.0)"},
	{VID: 0x1d19, PID: 0x1103, Name: "Dexatek Technology Ltd. DK 5217 DVB-T Dongle"},
	{VID: 0x1d19, PID: 0x1104, Name: "MSI DigiVox Micro HD"},
	{VID: 0x1f4d, PID: 0xa803, Name: "Sweex DVB-T USB"},
	{VID: 0x1f4d, PID: 0xb803, Name: "GTek T803"},
	{VID: 0x1f4d, PID: 0xc803, Name: "Lifeview LV5TDeluxe"},
	{VID: 0x1f4d, PID: 0xd286, Name: "MyGica TD312"},
	{VID: 0x1f4d, PID: 0xd803, Name: "PROlectrix DV107669"},
}

// lookupKnown returns the friendly device name for a (vid, pid)
// pair, or nil when the device isn't on the supported list. nil is
// the signal to [Driver.Enumerate] to skip the descriptor.
func lookupKnown(vid, pid uint16) *knownDevice {
	for i := range knownDevices {
		if knownDevices[i].VID == vid && knownDevices[i].PID == pid {
			return &knownDevices[i]
		}
	}
	return nil
}
