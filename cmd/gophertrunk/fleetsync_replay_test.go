package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/fleetsync"
	fsafsk "github.com/MattCheramie/GopherTrunk/internal/radio/fleetsync/afsk"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// TestFleetSyncReplay is the on-air verification gate for the FleetSync
// decoder (#437 / #1184): it replays a real Kenwood capture through the
// PRODUCTION front end (fleetsync/afsk: FM → FFSK → symbol timing →
// framer → decoder) and prints a per-burst timeline, then asserts that
// the capture's known ANI decodes CRC-valid. Skip-guarded; run with:
//
//	GT_FLEETSYNC_IQ=<capture> GT_FLEETSYNC_RATE=<Hz> \
//	  go test ./cmd/gophertrunk -run TestFleetSyncReplay -v
//
// Knobs (all optional):
//
//	GT_FLEETSYNC_FORMAT   f32 (default; interleaved IEEE float32 I/Q — the
//	                      #1184 captures), cs16, or audio (mono float32
//	                      discriminator audio). wav/flac containers are
//	                      content-sniffed and carry their own rate.
//	GT_FLEETSYNC_RATE     input sample rate. If unset and the file is
//	                      headerless the harness PROBES a list of common
//	                      rates and reports CRC-valid bursts per rate — the
//	                      CRC is the only trustworthy gate, a wrong rate
//	                      decodes nothing. GT_FLEETSYNC_RATES overrides the
//	                      probe list (comma-separated).
//	GT_FLEETSYNC_TUNE_HZ  channel offset inside a wideband capture (the
//	                      channel is mixed to DC before decimation).
//	GT_FLEETSYNC_BAUD     1200 or 2400; unset sweeps both and reports each.
//	GT_FLEETSYNC_FLEET /  the capture's ground-truth ANI. Default 107 /
//	GT_FLEETSYNC_UNIT     1772 (the reporter's radios). FLEET=0 disables the
//	                      assertion and only reports.
func TestFleetSyncReplay(t *testing.T) {
	path := os.Getenv("GT_FLEETSYNC_IQ")
	if path == "" {
		t.Skip("set GT_FLEETSYNC_IQ (+ GT_FLEETSYNC_RATE) to replay a FleetSync capture")
	}
	capt := loadFleetSyncCapture(t, path)
	t.Logf("capture: %s format=%s samples=%d rate=%s", path, capt.format, capt.length(), rateOrProbe(capt.rate))
	t.Logf("signal: %s", capt.signalStats())

	tuneHz := fsEnvFloat(t, "GT_FLEETSYNC_TUNE_HZ", 0)
	wantFleet := int(fsEnvFloat(t, "GT_FLEETSYNC_FLEET", 107))
	wantUnit := int(fsEnvFloat(t, "GT_FLEETSYNC_UNIT", 1772))

	bauds := []int{1200, 2400}
	if v := os.Getenv("GT_FLEETSYNC_BAUD"); v != "" {
		b, err := strconv.Atoi(v)
		if err != nil || b <= 0 {
			t.Fatalf("bad GT_FLEETSYNC_BAUD %q", v)
		}
		bauds = []int{b}
	}

	// Resolve the rate: explicit / container, else probe.
	if capt.rate == 0 {
		capt.rate = probeFleetSyncRate(t, capt, tuneHz, bauds)
		if capt.rate == 0 {
			t.Fatalf("no candidate rate produced a CRC-valid burst; set GT_FLEETSYNC_RATE to the capture's true rate (and GT_FLEETSYNC_FORMAT if it is not float32 I/Q)")
		}
	}

	verified := false
	for _, baud := range bauds {
		res := runFleetSyncChain(t, capt, baud, tuneHz)
		t.Logf("=== baud %d: sync_locks=%d framed=%d crc_ok=%d crc_fail=%d bits=%d",
			baud, res.framer.BurstsIn, res.framer.BurstsEmitted, res.crcOK, res.framer.BurstsBadCRC, res.front.BitsEmitted)
		for _, m := range res.msgs {
			flag := "  "
			if m.msg.CRCOK {
				flag = "OK"
			}
			t.Logf("  t=%8.3fs %s %-12s fleet=%-4d unit=%-5d raw=%s", m.t, flag, fsLabel(m.msg), m.msg.Fleet, m.msg.Unit, m.msg.RawHex)
		}
		hits := 0
		for _, m := range res.msgs {
			if m.msg.CRCOK && m.msg.Fleet == wantFleet && m.msg.Unit == wantUnit {
				hits++
			}
		}
		if wantFleet != 0 {
			t.Logf("  expected ANI fleet=%d unit=%d: %d CRC-valid hit(s) at %d baud", wantFleet, wantUnit, hits, baud)
			if hits > 0 {
				verified = true
			}
		} else if res.crcOK > 0 {
			verified = true
		}
	}

	switch {
	case wantFleet == 0:
		t.Logf("VERDICT: report only (GT_FLEETSYNC_FLEET=0); CRC-valid bursts seen=%v", verified)
	case verified:
		t.Logf("VERDICT: PASS — the capture's ANI (fleet %d unit %d) decodes CRC-valid through the production front end", wantFleet, wantUnit)
	default:
		t.Errorf("VERDICT: FAIL — expected fleet %d unit %d never decoded CRC-valid (check rate/format/tune first: a wrong rate decodes nothing; a CRC-failed burst with a plausible sync means the framing or field layout is the next suspect)", wantFleet, wantUnit)
	}
}

// fleetSyncCapture is the loaded capture: either IQ or discriminator audio.
type fleetSyncCapture struct {
	format string
	rate   float64 // 0 ⇒ unknown (probe)
	iq     []complex64
	audio  []float32
}

func (c *fleetSyncCapture) length() int {
	if c.audio != nil {
		return len(c.audio)
	}
	return len(c.iq)
}

// signalStats is the gain-staging instrument (#764 lesson): a dead or
// clipped capture answers nothing, so say so before anything else.
func (c *fleetSyncCapture) signalStats() string {
	var sum, peak, dcI, dcQ float64
	n := float64(c.length())
	if n == 0 {
		return "empty"
	}
	if c.audio != nil {
		for _, s := range c.audio {
			v := float64(s)
			sum += v * v
			if a := math.Abs(v); a > peak {
				peak = a
			}
			dcI += v
		}
		return fmt.Sprintf("rms=%.1f dBFS peak=%.1f dBFS dc=%.4f", 10*math.Log10(sum/n+1e-30), 20*math.Log10(peak+1e-30), dcI/n)
	}
	for _, s := range c.iq {
		re, im := float64(real(s)), float64(imag(s))
		p := re*re + im*im
		sum += p
		if p > peak {
			peak = p
		}
		dcI += re
		dcQ += im
	}
	return fmt.Sprintf("rms=%.1f dBFS peak=%.1f dBFS dc=(%.4f,%.4f)", 10*math.Log10(sum/n+1e-30), 10*math.Log10(peak+1e-30), dcI/n, dcQ/n)
}

func loadFleetSyncCapture(t *testing.T, path string) *fleetSyncCapture {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	c := &fleetSyncCapture{}
	if v := os.Getenv("GT_FLEETSYNC_RATE"); v != "" {
		c.rate = fsEnvFloat(t, "GT_FLEETSYNC_RATE", 0)
	}
	if _, isContainer := siglab.SniffContainer(raw); isContainer {
		samples, rate, err := siglab.DecodeContainerFile(path)
		if err != nil {
			t.Fatal(err)
		}
		c.format = "container"
		c.iq = samples
		if c.rate == 0 && rate > 0 {
			c.rate = float64(rate)
		}
		return c
	}
	format := strings.ToLower(os.Getenv("GT_FLEETSYNC_FORMAT"))
	if format == "" {
		format = "f32"
	}
	c.format = format
	switch format {
	case "f32", "cfile":
		c.iq = make([]complex64, len(raw)/8)
		for i := range c.iq {
			re := math.Float32frombits(binary.LittleEndian.Uint32(raw[i*8:]))
			im := math.Float32frombits(binary.LittleEndian.Uint32(raw[i*8+4:]))
			c.iq[i] = complex(re, im)
		}
	case "cs16":
		c.iq = make([]complex64, len(raw)/4)
		for i := range c.iq {
			re := int16(binary.LittleEndian.Uint16(raw[i*4:]))
			im := int16(binary.LittleEndian.Uint16(raw[i*4+2:]))
			c.iq[i] = complex(float32(re)/32768, float32(im)/32768)
		}
	case "audio":
		c.audio = make([]float32, len(raw)/4)
		for i := range c.audio {
			c.audio[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
		}
	default:
		t.Fatalf("GT_FLEETSYNC_FORMAT=%q: want f32, cs16 or audio", format)
	}
	return c
}

// fleetSyncTimed is one framed burst stamped with its capture time.
type fleetSyncTimed struct {
	t   float64
	msg fleetsync.Message
}

type fleetSyncRun struct {
	msgs   []fleetSyncTimed
	crcOK  int
	framer fleetsync.Stats
	front  fsafsk.Stats
}

// fleetSyncDDCTargetHz is the rate the front end is fed from a wideband
// capture: the same 48 kHz channel slice every other NBFM signalling
// path in the tree uses.
const fleetSyncDDCTargetHz = 48000

// runFleetSyncChain replays the capture through the production front end
// at one baud rate. A wideband (or offset) IQ capture goes through the
// production Downconverter to a 48 kHz channel slice first; a narrowband
// one feeds the front end directly, which resamples to its own audio
// rate.
func runFleetSyncChain(t *testing.T, c *fleetSyncCapture, baud int, tuneHz float64) fleetSyncRun {
	t.Helper()
	var res fleetSyncRun
	var consumed uint64
	onMsg := func(m fleetsync.Message) {
		res.msgs = append(res.msgs, fleetSyncTimed{t: float64(consumed) / c.rate, msg: m})
		if m.CRCOK {
			res.crcOK++
		}
	}

	const chunk = 4096
	if c.audio != nil {
		rcv, err := fsafsk.New(fsafsk.Options{InputRateHz: uint32(math.Round(c.rate)), BaudHz: baud, OnMessage: onMsg})
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < len(c.audio); i += chunk {
			end := min(i+chunk, len(c.audio))
			rcv.ProcessAudio(c.audio[i:end])
			consumed = uint64(end)
		}
		res.framer, res.front = rcv.Framer().Stats(), rcv.Stats()
		return res
	}

	var ddc *ccdecoder.Downconverter
	frontRate := c.rate
	if c.rate > fleetSyncDDCTargetHz || tuneHz != 0 {
		ddc = ccdecoder.NewDownconverterWithOffset(c.rate, fleetSyncDDCTargetHz, tuneHz)
		frontRate = ddc.OutRateHz()
	}
	rcv, err := fsafsk.New(fsafsk.Options{InputRateHz: uint32(math.Round(frontRate)), BaudHz: baud, OnMessage: onMsg})
	if err != nil {
		t.Fatal(err)
	}
	var ddcBuf []complex64
	for i := 0; i < len(c.iq); i += chunk {
		end := min(i+chunk, len(c.iq))
		in := c.iq[i:end]
		if ddc != nil {
			ddcBuf = ddc.Process(ddcBuf, in)
			in = ddcBuf
		}
		rcv.ProcessIQ(in)
		consumed = uint64(end)
	}
	res.framer, res.front = rcv.Framer().Stats(), rcv.Stats()
	return res
}

// probeFleetSyncRate sweeps candidate sample rates for a headerless
// capture of unknown rate and returns the one with the most CRC-valid
// bursts (0 if none). The CRC is the gate: a wrong rate mis-times every
// symbol and decodes nothing, so the table it prints is unambiguous.
func probeFleetSyncRate(t *testing.T, c *fleetSyncCapture, tuneHz float64, bauds []int) float64 {
	t.Helper()
	candidates := []float64{8000, 9600, 11025, 12000, 16000, 22050, 24000, 25000, 32000, 44100, 48000, 50000, 96000, 100000, 192000, 200000, 250000, 256000, 500000, 1000000, 1024000, 2000000, 2048000, 2400000, 2500000, 3000000}
	if v := os.Getenv("GT_FLEETSYNC_RATES"); v != "" {
		candidates = nil
		for _, s := range strings.Split(v, ",") {
			f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil {
				t.Fatalf("bad GT_FLEETSYNC_RATES entry %q", s)
			}
			candidates = append(candidates, f)
		}
	}
	type row struct {
		rate  float64
		baud  int
		crcOK int
		locks uint64
	}
	var rows []row
	for _, rate := range candidates {
		for _, baud := range bauds {
			probe := *c
			probe.rate = rate
			res := runFleetSyncChain(t, &probe, baud, tuneHz)
			rows = append(rows, row{rate, baud, res.crcOK, res.framer.BurstsIn})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].crcOK > rows[j].crcOK })
	t.Logf("rate probe (GT_FLEETSYNC_RATE unset): rate/baud → CRC-valid bursts (sync locks)")
	for _, r := range rows {
		if r.crcOK > 0 || r.locks > 0 {
			t.Logf("  %9.0f Hz @ %d baud: crc_ok=%d locks=%d", r.rate, r.baud, r.crcOK, r.locks)
		}
	}
	if len(rows) == 0 || rows[0].crcOK == 0 {
		return 0
	}
	t.Logf("  → using %.0f Hz", rows[0].rate)
	return rows[0].rate
}

func fsLabel(m fleetsync.Message) string {
	if m.IsFS2 {
		return "FleetSync-II"
	}
	return "FleetSync-I"
}

func rateOrProbe(r float64) string {
	if r == 0 {
		return "unknown (probe)"
	}
	return fmt.Sprintf("%.0f Hz", r)
}

func fsEnvFloat(t *testing.T, key string, def float64) float64 {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		t.Fatalf("bad %s=%q: %v", key, v, err)
	}
	return f
}

// ---- harness self-check ----------------------------------------------

// writeSyntheticFleetSyncCapture renders a float32 I/Q file (the #1184
// format) at rate Hz carrying an FS-I burst, a gap of unmodulated carrier
// plus noise, and an FS-II burst, all FFSK-modulated onto an FM carrier
// shifted by offsetHz. Returns the path.
func writeSyntheticFleetSyncCapture(t *testing.T, dir string, rate, offsetHz float64) string {
	t.Helper()
	iq := synthFleetSyncIQ(t, rate, offsetHz)
	buf := make([]byte, 8*len(iq))
	for i, s := range iq {
		binary.LittleEndian.PutUint32(buf[i*8:], math.Float32bits(real(s)))
		binary.LittleEndian.PutUint32(buf[i*8+4:], math.Float32bits(imag(s)))
	}
	path := dir + "/synthetic_fleetsync_f32.iq"
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func synthFleetSyncIQ(t *testing.T, rate, offsetHz float64) []complex64 {
	t.Helper()
	fs1, err := fleetsync.SynthBurst(107, 1772, false)
	if err != nil {
		t.Fatal(err)
	}
	fs2, err := fleetsync.SynthBurst(107, 1772, true)
	if err != nil {
		t.Fatal(err)
	}
	bits := append(fleetsync.Dotting(64), fs1...)
	bits = append(bits, fleetsync.Dotting(48)...)
	bits = append(bits, fleetsync.Dotting(64)...)
	bits = append(bits, fs2...)
	bits = append(bits, fleetsync.Dotting(48)...)
	// Modulate at 48 kHz (the shared FFSK modulator's fixed 0.5 rad/sample
	// deviation is a realistic ±3.8 kHz there — at 250 kS/s it would be
	// ±20 kHz, wider than any NBFM channel or the DDC's 48 kHz slice) and
	// interpolate up to a wideband rate when asked.
	iq := demodModulateFFSK(bits, 48000)
	if rate != 48000 {
		iq = interpolateIQ(iq, 48000, rate)
	}
	// Half a second of quiet carrier either side, then the whole thing
	// shifted off DC by offsetHz.
	quiet := make([]complex64, int(rate/2))
	for i := range quiet {
		quiet[i] = 1
	}
	all := append(append(append([]complex64{}, quiet...), iq...), quiet...)
	if offsetHz != 0 {
		step := 2 * math.Pi * offsetHz / rate
		for i, s := range all {
			c, sn := math.Cos(step*float64(i)), math.Sin(step*float64(i))
			re, im := float64(real(s)), float64(imag(s))
			all[i] = complex(float32(re*c-im*sn), float32(re*sn+im*c))
		}
	}
	return all
}

// TestFleetSyncReplayHarnessSelfCheck pins the harness itself: a
// synthetic float32 capture must load, the rate probe must pick the true
// rate from a CRC-count table, and both burst formats must decode to the
// ANI they carry — with and without a wideband tune offset — so a FAIL on
// the reporter's real capture means the capture or the framing, never a
// broken harness.
func TestFleetSyncReplayHarnessSelfCheck(t *testing.T) {
	dir := t.TempDir()

	t.Run("narrowband f32 + rate probe", func(t *testing.T) {
		path := writeSyntheticFleetSyncCapture(t, dir, 48000, 0)
		t.Setenv("GT_FLEETSYNC_FORMAT", "f32")
		t.Setenv("GT_FLEETSYNC_RATE", "")
		t.Setenv("GT_FLEETSYNC_RATES", "24000,48000,96000")
		c := loadFleetSyncCapture(t, path)
		if c.rate != 0 || len(c.iq) == 0 {
			t.Fatalf("loaded rate=%v samples=%d, want unknown rate and samples", c.rate, len(c.iq))
		}
		if got := probeFleetSyncRate(t, c, 0, []int{1200}); got != 48000 {
			t.Fatalf("rate probe picked %.0f, want 48000", got)
		}
		c.rate = 48000
		assertFleetSyncRun(t, runFleetSyncChain(t, c, 1200, 0))
	})

	t.Run("wideband cs16 with tune offset", func(t *testing.T) {
		const rate, off = 250000.0, 62500.0
		iq := synthFleetSyncIQ(t, rate, off)
		buf := make([]byte, 4*len(iq))
		for i, s := range iq {
			binary.LittleEndian.PutUint16(buf[i*4:], uint16(int16(real(s)*32767)))
			binary.LittleEndian.PutUint16(buf[i*4+2:], uint16(int16(imag(s)*32767)))
		}
		path := dir + "/synthetic_fleetsync.cs16"
		if err := os.WriteFile(path, buf, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GT_FLEETSYNC_FORMAT", "cs16")
		t.Setenv("GT_FLEETSYNC_RATE", "250000")
		c := loadFleetSyncCapture(t, path)
		if c.rate != rate {
			t.Fatalf("loaded rate=%v, want %v", c.rate, rate)
		}
		assertFleetSyncRun(t, runFleetSyncChain(t, c, 1200, off))
	})

	t.Run("discriminator audio", func(t *testing.T) {
		iq := synthFleetSyncIQ(t, 48000, 0)
		audio := demodFM(iq)
		buf := make([]byte, 4*len(audio))
		for i, s := range audio {
			binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(s))
		}
		path := dir + "/synthetic_fleetsync_audio.f32"
		if err := os.WriteFile(path, buf, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GT_FLEETSYNC_FORMAT", "audio")
		t.Setenv("GT_FLEETSYNC_RATE", "48000")
		c := loadFleetSyncCapture(t, path)
		if c.audio == nil {
			t.Fatal("audio format did not load as audio")
		}
		assertFleetSyncRun(t, runFleetSyncChain(t, c, 1200, 0))
	})
}

func assertFleetSyncRun(t *testing.T, res fleetSyncRun) {
	t.Helper()
	var fs1, fs2 int
	for _, m := range res.msgs {
		if !m.msg.CRCOK || m.msg.Fleet != 107 || m.msg.Unit != 1772 {
			t.Errorf("unexpected burst %+v", m.msg)
			continue
		}
		if m.msg.IsFS2 {
			fs2++
		} else {
			fs1++
		}
	}
	if fs1 != 1 || fs2 != 1 {
		t.Fatalf("decoded fs1=%d fs2=%d CRC-valid bursts, want 1 and 1 (framer stats %+v)", fs1, fs2, res.framer)
	}
	if res.msgs[0].t <= 0.4 || res.msgs[0].t > 2.5 {
		t.Errorf("first burst timestamp %.3fs, want inside the modulated span after the 0.5 s lead-in", res.msgs[0].t)
	}
}
