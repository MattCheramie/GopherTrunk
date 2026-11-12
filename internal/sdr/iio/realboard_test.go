package iio

import (
	"os"
	"testing"
)

// testdata/plutosky_rev_c_context.xml is the PRINT output of the #1329
// reporter's PlutoSky (6 Nov 2026: "Analog Devices PlutoSDR Rev.C
// (Z7020-AD9361)", firmware v0.38, iiod 0.25, reached over the USB network
// link at 192.168.2.1; the same board on tezuka-v0.3.24 firmware reports
// "FISHBall-PlutoSky" and an RX LO floor of 46 875 001 Hz). It replaced a
// listing reconstructed from the published layout, which numbered the
// devices differently (phy iio:device1 / rx iio:device4; the board has
// iio:device0 / iio:device3). Every attribute the driver reads or writes
// must exist on the channel it addresses on this board: a name that is not
// there is a silent -ENOENT on air that no synthetic fixture can catch.
func TestDriverAttributesExistOnRealBoard(t *testing.T) {
	raw, err := os.ReadFile("testdata/plutosky_rev_c_context.xml")
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := ParseContext(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := ctx.Attrs["hw_model"]; got != "Analog Devices PlutoSDR Rev.C (Z7020-AD9361)" {
		t.Errorf("hw_model = %q", got)
	}
	dev, err := bindDevices(ctx)
	if err != nil {
		t.Fatalf("bindDevices on the real board: %v", err)
	}
	if dev.phy.ID != "iio:device0" || dev.rx.ID != "iio:device3" {
		t.Errorf("bound phy=%s rx=%s, want iio:device0 / iio:device3", dev.phy.ID, dev.rx.ID)
	}
	if dev.fullScale != 2048 || dev.bits != 12 || !dev.iFirst || dev.mask != "00000003" {
		t.Errorf("sample path: full_scale=%v bits=%d iFirst=%v mask=%s", dev.fullScale, dev.bits, dev.iFirst, dev.mask)
	}
	for _, a := range []string{attrFreq, attrFreqAv, attrPDown} {
		if !dev.loChan.HasAttr(a) {
			t.Errorf("RX LO (%s %s, output) has no %q", dev.phy.ID, rxLOChan, a)
		}
	}
	for _, a := range []string{attrRate, attrBW, attrGainMd, attrGain, attrPort} {
		if !dev.phyChan.HasAttr(a) {
			t.Errorf("%s %s (input) has no %q", dev.phy.ID, rxChan, a)
		}
	}
	rxI := dev.rx.FindChannel(rxIChan, false)
	if rxI == nil || !rxI.HasAttr(attrRate) {
		t.Errorf("%s %s has no %q (ActualSampleRate reads it)", dev.rx.ID, rxIChan, attrRate)
	}
}
