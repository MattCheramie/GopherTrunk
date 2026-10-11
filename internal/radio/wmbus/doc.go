// Package wmbus decodes Wireless M-Bus (EN 13757-4) telegrams, the 868 MHz
// radio protocol of European utility meters: heat cost allocators, water,
// gas, heat and electricity meters, and the OMS (Open Metering System)
// devices built on it (issue #1256).
//
// The decode chain is:
//
//	IQ (any rate)
//	  → Receiver: resample to 800 kS/s, channel filter, FM discriminator,
//	    adaptive slicer, run-length clock recovery (receiver.go)
//	  → Framer: access code 0x543D, then mode T1 (3-of-6 coded) or C1
//	    (NRZ, frame format A or B), CRC check and strip (framer.go, link.go)
//	  → ParseTelegram: data link header (manufacturer, ID, version, device
//	    type), extended link layer, transport layer header and security
//	    mode, and the unencrypted application data records (telegram.go,
//	    records.go)
//
// Modes: T1 and C1, the two meter-to-other modes used by almost all
// installed meters, both at 868.95 MHz and 100 kchip/s. S1 (868.3 MHz,
// Manchester) and the other-to-meter modes are not decoded.
//
// Encryption: most OMS meters encrypt their application data (security
// mode 5 or 7, AES-128 with a per-meter key from the metering company).
// Such a telegram's header (who sent it, when, device type) still decodes;
// its readings do not, and no decryption is attempted.
//
// References, cross-checked rather than copied: EN 13757-3/-4, and two
// independent decoders, rtl-wmbus (BSD-2-Clause; its four sample captures
// are the real-air verification of this package, see the skip-guarded
// TestWMBusRealAirCaptures) and wmbusmeters (GPL; its simulation telegrams
// and their decoded values pin the header and record parser as literal
// test vectors).
package wmbus
