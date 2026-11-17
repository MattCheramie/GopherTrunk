package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/powersweep"
)

// fakeIIODBoard is a Pluto-shaped iiod: it serves the #1329 reporter's real
// board listing, records every RX LO retune, and answers READBUF with 12-bit
// CS16 samples of one carrier at toneHz whenever it is inside the tuned band.
type fakeIIODBoard struct {
	ln     net.Listener
	xml    []byte
	toneHz float64
	// dc and noise, when set, are added to every sample: the AD9361's own
	// DC offset and its noise floor, as fractions of full scale.
	dc, noise float64
	// truncRate makes the board read its sample rate back one hertz low,
	// as the AD9361 driver does for a rate its clock chain lands a fraction
	// of a hertz under.
	truncRate bool

	mu    sync.Mutex
	freq  float64
	rate  float64
	tunes []uint32
	n     int64
	rng   *rand.Rand
}

func newFakeIIODBoard(t *testing.T, toneHz float64) *fakeIIODBoard {
	t.Helper()
	xml, err := os.ReadFile("../../internal/sdr/iio/testdata/plutosky_rev_c_context.xml")
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeIIODBoard{ln: ln, xml: xml, toneHz: toneHz, rate: 2_400_000, rng: rand.New(rand.NewSource(1230))}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go s.handle(c)
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return s
}

func (s *fakeIIODBoard) handle(c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		tok := strings.Fields(line)
		if len(tok) == 0 {
			continue
		}
		switch tok[0] {
		case "VERSION":
			io.WriteString(c, "0.26.v0.26\n")
		case "PRINT":
			fmt.Fprintf(c, "%d\n", len(s.xml))
			c.Write(s.xml)
			io.WriteString(c, "\n")
		case "READ":
			// READ <dev> INPUT|OUTPUT <chn> <attr>
			var v string
			switch tok[4] {
			case "frequency_available":
				v = "[46875001 1 6000000000]"
			case "sampling_frequency":
				s.mu.Lock()
				v = strconv.Itoa(int(s.rate))
				if s.truncRate {
					v = strconv.Itoa(int(s.rate) - 1)
				}
				s.mu.Unlock()
			default:
				io.WriteString(c, "-2\n")
				continue
			}
			fmt.Fprintf(c, "%d\n%s\n", len(v), v)
		case "WRITE":
			n, _ := strconv.Atoi(tok[len(tok)-1])
			payload := make([]byte, n)
			if _, err := io.ReadFull(r, payload); err != nil {
				return
			}
			val := strings.TrimRight(string(payload), "\x00")
			switch {
			case tok[1] == "iio:device0" && tok[3] == "altvoltage0" && tok[4] == "frequency":
				hz, _ := strconv.ParseFloat(val, 64)
				s.mu.Lock()
				s.freq = hz
				s.tunes = append(s.tunes, uint32(hz))
				s.mu.Unlock()
			case tok[1] == "iio:device3" && tok[4] == "sampling_frequency":
				hz, _ := strconv.ParseFloat(val, 64)
				s.mu.Lock()
				s.rate = hz
				s.mu.Unlock()
			}
			fmt.Fprintf(c, "%d\n", n)
		case "OPEN", "CLOSE", "TIMEOUT":
			io.WriteString(c, "0\n")
		case "READBUF":
			nbytes, _ := strconv.Atoi(tok[2])
			s.mu.Lock()
			freq, rate := s.freq, s.rate
			off := s.toneHz - freq
			inBand := math.Abs(off) < rate/2
			buf := make([]byte, nbytes)
			for i := 0; i+4 <= nbytes; i += 4 {
				iv, qv := 0.0, 0.0
				if inBand {
					ph := 2 * math.Pi * off * float64(s.n) / rate
					iv, qv = 0.5*math.Cos(ph), 0.5*math.Sin(ph)
				}
				if s.noise != 0 {
					iv += s.dc + s.noise*s.rng.NormFloat64()
					qv += s.noise * s.rng.NormFloat64()
				}
				// 12-bit LSB-aligned, full scale 2048 (le:S12/16>>0).
				binary.LittleEndian.PutUint16(buf[i:], uint16(int16(math.Round(2047*iv))))
				binary.LittleEndian.PutUint16(buf[i+2:], uint16(int16(math.Round(2047*qv))))
				s.n++
			}
			s.mu.Unlock()
			// Pace it like a board filling a buffer at the sample rate.
			time.Sleep(time.Duration(float64(nbytes/4) / rate * float64(time.Second)))
			fmt.Fprintf(c, "%d\n00000003\n", nbytes)
			if _, err := c.Write(buf); err != nil {
				return
			}
		case "EXIT":
			return
		default:
			io.WriteString(c, "-22\n")
		}
	}
}

func (s *fakeIIODBoard) tuned() []uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]uint32(nil), s.tunes...)
}

// TestPowerSweepOverIIO is the #1230 / #1329 path: `gophertrunk power -iio`
// against a Pluto-shaped iiod serving the reporter's real board listing.
// Before the flag existed `power` read no config, so an `sdr.iio` entry did
// nothing and the reporter's sweep ended in "no SDRs found". Each hop must
// be tuned on the board, the 12-bit samples must arrive at the right level,
// and the CSV's loudest bin must sit at the carrier.
func TestPowerSweepOverIIO(t *testing.T) {
	const (
		rate   = 600_000
		toneHz = 100_333_000.0
	)
	srv := newFakeIIODBoard(t, toneHz)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	dev, name, err := openPowerDevice("", srv.ln.Addr().String(), "", log)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()
	if !strings.HasPrefix(name, "iio ") {
		t.Fatalf("device name %q", name)
	}
	if err := dev.SetSampleRate(rate); err != nil {
		t.Fatal(err)
	}
	r, _ := powersweep.ParseRange("100.0M:100.9M:5k")
	plan, err := powersweep.NewPlan(r, rate, 0.25)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	src, err := newPowerIQSource(ctx, dev, rate, 60*time.Millisecond, log)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	w := bufio.NewWriter(&out)
	err = runPowerSweeps(ctx, src, plan, powersweep.Integration{Samples: 4 * plan.FFTSize}, powerSchedule{every: time.Second, single: true},
		w, nil, time.Now, sleepCtx, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := srv.tuned()
	if len(got) != len(plan.Hops) {
		t.Fatalf("board saw %d tunes %v, want one per hop (%d)", len(got), got, len(plan.Hops))
	}
	for i, h := range plan.Hops {
		if got[i] != h.CenterHz {
			t.Fatalf("tune %d = %d Hz, want %d", i, got[i], h.CenterHz)
		}
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != len(plan.Hops) {
		t.Fatalf("%d CSV lines, want %d:\n%s", len(lines), len(plan.Hops), out.String())
	}
	bestDB, bestHz := -999.0, 0.0
	for _, line := range lines {
		f := strings.Split(line, ", ")
		low, _ := strconv.ParseFloat(f[2], 64)
		step, _ := strconv.ParseFloat(f[4], 64)
		for j, v := range f[6:] {
			db, err := strconv.ParseFloat(v, 64)
			if err != nil {
				t.Fatalf("bad dB %q in %q", v, line)
			}
			if db > bestDB {
				bestDB, bestHz = db, low+float64(j)*step
			}
		}
	}
	if math.Abs(bestHz-toneHz) > plan.BinHz {
		t.Fatalf("loudest bin at %.0f Hz, want within %.0f Hz of %.0f", bestHz, plan.BinHz, toneHz)
	}
	// 0.5 of the 12-bit full scale: about −6 dBFS (−8..−9.5 through the
	// Hann window). Read at the 16-bit scale it would be 24 dB lower (#1331).
	if bestDB < -11 || bestDB > -6 {
		t.Fatalf("loudest bin %.1f dBFS, want about −8 (the 12-bit full scale)", bestDB)
	}
}

// TestPowerSweepOverIIOShowsOnlyWhatIsOnTheAir is the #1230 PlutoSky sweep of
// 16 Nov, reproduced against a board that behaves as the reporter's did:
// nothing on the air, an AD9361 DC offset about 11 dB over the noise in the
// centre bin, and a sample rate read back one hertz low. Their CSV showed a
// three-bin carrier at the centre of every hop and hop edges a hertz further
// off each time (91599999, 95199999, 98799998 …).
func TestPowerSweepOverIIOShowsOnlyWhatIsOnTheAir(t *testing.T) {
	const rate = 600_000
	srv := newFakeIIODBoard(t, 1) // no carrier anywhere near the sweep
	srv.truncRate = true
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	dev, _, err := openPowerDevice("", srv.ln.Addr().String(), "", log)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()
	if err := dev.SetSampleRate(rate); err != nil {
		t.Fatal(err)
	}
	hw, note := captureEffectiveRate(dev, rate)
	if hw != rate || note != "" {
		t.Errorf("the board reads its rate back one hertz low: effective rate %d, note %q; want %d and no note", hw, note, rate)
	}
	r, _ := powersweep.ParseRange("100.0M:100.9M:5k")
	plan, err := powersweep.NewPlan(r, float64(hw), 0.25)
	if err != nil {
		t.Fatal(err)
	}
	// A DC offset 11 dB over the noise in the centre bin (see
	// powersweep.TestSweepRemovesTunerDCLine for the arithmetic).
	srv.noise = 0.01
	srv.dc = math.Sqrt(math.Pow(10, 1.1) * 2 * srv.noise * srv.noise * 1.5 / float64(plan.FFTSize))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	src, err := newPowerIQSource(ctx, dev, rate, 60*time.Millisecond, log)
	if err != nil {
		t.Fatal(err)
	}
	sweep := func(keepDC bool) [][]float64 {
		var out bytes.Buffer
		w := bufio.NewWriter(&out)
		err := runPowerSweeps(ctx, src, plan, powersweep.Integration{Samples: 300 * plan.FFTSize, KeepDC: keepDC},
			powerSchedule{every: time.Second, single: true}, w, nil, time.Now, sleepCtx, nil)
		if err != nil {
			t.Fatal(err)
		}
		var rows [][]float64
		for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
			f := strings.Split(line, ", ")
			// Hop i starts i whole hops of whole bins above the range's low edge.
			wantLow := int64(math.Round(r.LowHz + float64(i*plan.HopBins)*float64(rate)/float64(plan.FFTSize)))
			if low, _ := strconv.ParseInt(f[2], 10, 64); low != wantLow {
				t.Errorf("hop %d starts at %d Hz, want %d", i, low, wantLow)
			}
			var db []float64
			for _, v := range f[6:] {
				x, _ := strconv.ParseFloat(v, 64)
				db = append(db, x)
			}
			rows = append(rows, db)
		}
		return rows
	}
	// How far the highest of a row's three centre bins stands over its median.
	centre := func(db []float64) float64 {
		srt := append([]float64(nil), db...)
		sort.Float64s(srt)
		c := len(db) / 2
		return math.Max(db[c-1], math.Max(db[c], db[c+1])) - srt[len(srt)/2]
	}
	for i, db := range sweep(true) {
		if over := centre(db); over < 8 {
			t.Fatalf("hop %d with -keep-dc: centre %.1f dB over the floor; the fixture should show the board's DC line", i, over)
		}
	}
	for i, db := range sweep(false) {
		if len(db) != plan.Hops[i].Bins {
			t.Fatalf("hop %d has %d bins, want %d", i, len(db), plan.Hops[i].Bins)
		}
		if over := centre(db); over > 1.5 {
			t.Errorf("hop %d: a centre bin stands %.1f dB over the floor with nothing on the air", i, over)
		}
	}
}

// -rtltcp, -iio and -serial pick one radio each; two at once is a mistake.
func TestOpenPowerDeviceRefusesTwoSources(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if _, _, err := openPowerDevice("127.0.0.1:1", "127.0.0.1:2", "", log); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("err = %v", err)
	}
	if _, _, err := openPowerDevice("", "127.0.0.1:2", "ABC", log); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("err = %v", err)
	}
}
