package soapyremote

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/baseband"
)

// branchRecorder dumps the PRE-COMBINE per-branch IQ of a diversity stream.
//
// Every other IQ tap in GopherTrunk — baseband.auto_record, the iqtap brokers,
// the scope taps — sits downstream of the combiner, so a capture taken anywhere
// else has already had one particular combiner applied to it and cannot be used
// to evaluate a different one. This is the only tap that can answer "would
// tracking have beaten static on this signal?", or "is the second antenna
// contributing anything?", offline and reproducibly.
//
// Layout: one file per branch plus a JSON sidecar. The branch container is
// headerless CS16 by default, or lossless two-channel FLAC when
// diversity_capture_format: flac is set (typically 30-50% smaller — the "MRC
// autocapture should use FLAC for space saving" field request):
//
//	<prefix>.br0.cs16   (or .br0.flac)
//	<prefix>.br1.cs16   (or .br1.flac)
//	<prefix>.diversity.json
//
// Separate files rather than one interleaved blob, because each branch file is
// then immediately playable by everything that already exists — `gophertrunk
// replay -format cs16` (or the content-sniffing FLAC mount), `analyze`, siglab
// — so an operator can sanity-check each receiver on its own before anyone
// writes combiner code, and the offline harness is a plain pair of reads. A
// bespoke container would buy nothing and cost a parser plus a tool that
// understands it. FLAC is bit-exact: a .flac branch decodes to the identical
// int16 samples its .cs16 twin would carry, so nothing downstream can tell the
// containers apart and the alignment invariant below is container-independent.
//
// ALIGNMENT INVARIANT: the two files are only meaningful if sample i of one is
// simultaneous with sample i of the other. A datagram that did not carry every
// branch is therefore dropped from BOTH files and counted, never written short
// — a single short write would silently desynchronise them and quietly
// invalidate every conclusion drawn afterwards.
//
// write runs on the stream goroutine only. finish is also called from
// device.Close (mu makes that safe): without it, a capture interrupted before
// its budget filled — daemon stopped mid-dump — left branch files with an
// unflushed tail and NO sidecar, which the offline harness refuses to load (an
// operator shipped exactly that: a 64 s dump of a 120 s budget, sidecar-less
// and unusable). A partial capture with a sidecar is still a valid capture.
// Any write error disables the recorder rather than disturbing the stream: a
// capture that induces the overruns it is meant to diagnose is worse than no
// capture.
type branchRecorder struct {
	log      *slog.Logger
	prefix   string
	branches int
	mu       sync.Mutex

	files   []*os.File
	bufs    []*bufio.Writer           // cs16 path only
	encs    []*baseband.FLACIQEncoder // flac path only
	scratch []byte

	budgetSamples int64
	written       int64
	dropped       int64
	datagrams     int64
	done          bool
	failed        bool

	meta branchCaptureMeta
}

// branchCaptureMeta is the JSON sidecar. It is a capture-provenance record, not
// a decode-acceptance contract, so it deliberately does not reuse
// siglab.Metadata (which requires protocol/expected fields that mean nothing
// here).
type branchCaptureMeta struct {
	Addr             string   `json:"addr"`
	DeviceArgs       string   `json:"device_args,omitempty"`
	Antennas         []string `json:"antennas,omitempty"`
	SampleRateHz     float64  `json:"sample_rate_hz"`
	CenterFreqHz     uint32   `json:"center_freq_hz,omitempty"`
	GainTenthDB      int      `json:"gain_tenth_db,omitempty"`
	Format           string   `json:"format"`
	DiversityMode    string   `json:"diversity_mode"`
	Branches         int      `json:"branches"`
	BranchFiles      []string `json:"branch_files"`
	SamplesPerBranch int64    `json:"samples_per_branch"`
	Datagrams        int64    `json:"datagrams"`
	DroppedDatagrams int64    `json:"dropped_datagrams"`
	StreamMTU        int      `json:"stream_mtu,omitempty"`
}

// defaultDiversityCaptureSeconds bounds a capture when none is configured.
const defaultDiversityCaptureSeconds = 5

// startBranchCapture arms the pre-combine dump. It is one-shot per DEVICE, not
// per stream: a control-channel hunt re-opens the stream repeatedly, and
// re-arming would truncate the files on every cycle and leave the operator with
// whatever the last few seconds happened to be. Once a recorder exists — running
// or finished — the capture stays as it was.
//
// A failure to open the files is logged and otherwise ignored: a diagnostic tap
// must never keep the radio from streaming.
func (d *device) startBranchCapture(rateHz float64) {
	if d.capturePrefix == "" || d.mrc == nil || d.mrc.rec != nil {
		return
	}
	secs := d.captureSeconds
	if secs <= 0 {
		secs = defaultDiversityCaptureSeconds
	}
	budget := int64(0)
	if rateHz > 0 {
		budget = int64(rateHz) * int64(secs)
	}
	format := d.captureFormat
	if format == "flac" && rateHz > flacCaptureMaxRateHz {
		// FLAC's STREAMINFO rate field tops out near 1 MS/s, and the encode runs
		// on the stream goroutine — at high native rates it would risk inducing
		// the very overruns the capture exists to diagnose. Keep the capture
		// rather than the container.
		d.log.Warn("soapyremote: diversity_capture_format flac not usable at this rate — falling back to cs16",
			"rate_hz", rateHz, "max_flac_rate_hz", flacCaptureMaxRateHz)
		format = "cs16"
	}
	rec, err := newBranchRecorder(d.capturePrefix, format, d.rxChannelCount(), budget, rateHz, d.log)
	if err != nil {
		d.log.Warn("soapyremote: diversity capture not started", "err", err)
		return
	}
	rec.meta.Addr = d.addr
	rec.meta.SampleRateHz = rateHz
	rec.meta.DiversityMode = d.diversity.String()
	rec.meta.StreamMTU = d.mtu
	// Without these a replay cannot work out which channel to tune to, or what
	// hardware and antenna ports produced the branches — which is most of what
	// makes a capture reproducible.
	rec.meta.DeviceArgs = d.deviceArgs
	rec.meta.Antennas = append([]string(nil), d.antennas...)
	d.mu.Lock()
	rec.meta.CenterFreqHz = d.centerHz
	rec.meta.GainTenthDB = d.gainTenth
	d.mu.Unlock()
	d.mrc.rec = rec
	d.log.Info("soapyremote: diversity capture armed — dumping pre-combine per-branch IQ",
		"prefix", rec.prefix, "seconds", secs, "branches", d.rxChannelCount())
}

// branchCaptureMaxBytes caps a capture independently of the configured seconds,
// so a mis-set rate cannot fill a disk. Two CS16 branches at 6.25 MS/s is about
// 50 MB/s.
const branchCaptureMaxBytes = 1 << 30 // 1 GiB per branch

// resolveCapturePrefix turns a directory-style prefix — trailing separator, or
// naming an existing directory — into a timestamped file prefix inside that
// directory. Without this, "mrc_autocaptures/" + ".br0.cs16" produces HIDDEN
// dot-files (an operator shipped a capture nobody could see with plain ls).
// A plain filename prefix passes through untouched.
func resolveCapturePrefix(prefix string, now time.Time) string {
	isDir := os.IsPathSeparator(prefix[len(prefix)-1])
	if !isDir {
		if fi, err := os.Stat(prefix); err == nil && fi.IsDir() {
			isDir = true
		}
	}
	if !isDir {
		return prefix
	}
	return filepath.Join(prefix, "diversity_"+now.UTC().Format("20060102T150405Z"))
}

// flacCaptureMaxRateHz bounds the sample rate at which a flac branch capture is
// honoured: FLAC's STREAMINFO sample-rate field is 20 bits (1,048,575 Hz), and
// the encode's prediction analysis runs on the stream goroutine, so a
// multi-MS/s native rate falls back to cs16 (see startBranchCapture). Every
// narrowband MRC rig (the 200-250 kS/s X310/TwinRX use case) is far under it.
const flacCaptureMaxRateHz = 1_000_000

// newBranchRecorder opens the per-branch files under prefix in the given
// container format ("" or "cs16", or "flac"). budgetSamples bounds the capture
// per branch; <=0 means the byte cap alone applies. rateHz is stamped into a
// FLAC stream's header (cs16 ignores it).
func newBranchRecorder(prefix, format string, branches int, budgetSamples int64, rateHz float64, log *slog.Logger) (*branchRecorder, error) {
	if branches < 1 {
		return nil, fmt.Errorf("soapyremote: branch capture needs at least one branch")
	}
	if prefix == "" {
		return nil, fmt.Errorf("soapyremote: branch capture needs a file prefix")
	}
	switch format {
	case "", "cs16":
		format = "cs16"
	case "flac":
		if rateHz <= 0 || rateHz > flacCaptureMaxRateHz {
			return nil, fmt.Errorf("soapyremote: flac branch capture needs a sample rate in (0, %d] Hz, got %g", flacCaptureMaxRateHz, rateHz)
		}
	default:
		return nil, fmt.Errorf("soapyremote: unknown branch capture format %q (want cs16 or flac)", format)
	}
	prefix = resolveCapturePrefix(prefix, time.Now())
	// Auto-create the directory like every other output path (the config
	// validator no longer hard-fails on a missing one). Cheap and idempotent.
	if dir := filepath.Dir(prefix); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("soapyremote: branch capture dir %s: %w", dir, err)
		}
	}
	r := &branchRecorder{
		log:           log,
		prefix:        prefix,
		branches:      branches,
		budgetSamples: budgetSamples,
	}
	for i := 0; i < branches; i++ {
		name := prefix + ".br" + strconv.Itoa(i) + "." + format
		f, err := os.Create(name)
		if err != nil {
			r.closeFiles()
			return nil, fmt.Errorf("soapyremote: branch capture %s: %w", name, err)
		}
		r.files = append(r.files, f)
		if format == "flac" {
			enc, err := baseband.NewFLACIQEncoder(f, uint32(rateHz))
			if err != nil {
				r.closeFiles()
				return nil, fmt.Errorf("soapyremote: branch capture %s: %w", name, err)
			}
			r.encs = append(r.encs, enc)
		} else {
			r.bufs = append(r.bufs, bufio.NewWriterSize(f, 1<<20))
		}
		r.meta.BranchFiles = append(r.meta.BranchFiles, filepath.Base(name))
	}
	r.meta.Branches = branches
	r.meta.Format = format
	return r, nil
}

// write records one datagram's branches. Call immediately after de-interleaving
// and before any combining, so the capture is byte-for-byte what the combiner
// saw.
func (r *branchRecorder) write(branches [][]complex64) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done || r.failed {
		return
	}
	r.datagrams++
	if len(branches) != r.branches {
		// Degenerate datagram: writing it would desynchronise the files.
		r.dropped++
		return
	}
	n := len(branches[0])
	for _, b := range branches {
		if len(b) != n {
			r.dropped++
			return
		}
	}
	if n == 0 {
		return
	}
	if r.budgetSamples > 0 && r.written+int64(n) > r.budgetSamples {
		n = int(r.budgetSamples - r.written)
		if n <= 0 {
			r.finishLocked()
			return
		}
	}
	if r.encs != nil {
		// FLAC path: feed each branch's encoder the SAME clamped int16 samples
		// the cs16 path would write, so the two containers are bit-exact twins.
		for i, b := range branches {
			for _, z := range b[:n] {
				if err := r.encs[i].WriteI16(clampInt16(real(z)*32768), clampInt16(imag(z)*32768)); err != nil {
					r.fail(err)
					return
				}
			}
		}
	} else {
		need := n * 4
		if cap(r.scratch) < need {
			r.scratch = make([]byte, need)
		}
		buf := r.scratch[:need]
		for i, b := range branches {
			encodeCS16Into(buf, b[:n])
			if _, err := r.bufs[i].Write(buf); err != nil {
				r.fail(err)
				return
			}
		}
	}
	r.written += int64(n)
	if (r.budgetSamples > 0 && r.written >= r.budgetSamples) || r.written*4 >= branchCaptureMaxBytes {
		r.finishLocked()
	}
}

// finish flushes, writes the sidecar, and logs the paths. Idempotent, and safe
// to call from outside the stream goroutine (device.Close finalizes a capture
// interrupted before its budget filled, so the sidecar always lands).
func (r *branchRecorder) finish() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.finishLocked()
}

func (r *branchRecorder) finishLocked() {
	if r.done {
		return
	}
	r.done = true
	for _, b := range r.bufs {
		if err := b.Flush(); err != nil && !r.failed {
			r.fail(err)
		}
	}
	for _, e := range r.encs {
		// Finalize flushes the last partial block and patches the STREAMINFO
		// (sample count + MD5) via a seek on the file; the file close is ours.
		if err := e.Finalize(); err != nil && !r.failed {
			r.fail(err)
		}
	}
	r.closeFiles()
	if r.failed {
		return
	}
	r.meta.SamplesPerBranch = r.written
	r.meta.Datagrams = r.datagrams
	r.meta.DroppedDatagrams = r.dropped
	path := r.prefix + ".diversity.json"
	blob, err := json.MarshalIndent(&r.meta, "", "  ")
	if err == nil {
		err = os.WriteFile(path, append(blob, '\n'), 0o644)
	}
	if err != nil {
		r.log.Warn("soapyremote: diversity capture sidecar not written", "path", path, "err", err)
		return
	}
	r.log.Info("soapyremote: diversity capture complete — replay it with GT_DIVERSITY_CAPTURE to A/B combiners offline",
		"sidecar", path,
		"samples_per_branch", r.written,
		"dropped_datagrams", r.dropped)
}

// fail disables the recorder after a write error, warning once. The stream is
// never disturbed: a capture that induces overruns is worse than no capture.
func (r *branchRecorder) fail(err error) {
	if r.failed {
		return
	}
	r.failed = true
	r.done = true
	r.closeFiles()
	r.log.Warn("soapyremote: diversity capture disabled after a write error", "prefix", r.prefix, "err", err)
}

func (r *branchRecorder) closeFiles() {
	for _, f := range r.files {
		_ = f.Close()
	}
	r.files = nil
	r.bufs = nil
}

// encodeCS16Into writes iq as interleaved little-endian int16 I,Q — the inverse
// of convertCS16, and the same layout every other cs16 capture in the repo uses.
// dst must hold at least len(iq)*4 bytes.
func encodeCS16Into(dst []byte, iq []complex64) {
	for i, z := range iq {
		binary.LittleEndian.PutUint16(dst[4*i:], uint16(clampInt16(real(z)*32768)))
		binary.LittleEndian.PutUint16(dst[4*i+2:], uint16(clampInt16(imag(z)*32768)))
	}
}

func clampInt16(v float32) int16 {
	if v > 32767 {
		return 32767
	}
	if v < -32768 {
		return -32768
	}
	return int16(v)
}
