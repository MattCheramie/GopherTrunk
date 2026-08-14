package voice

// dsd-fme-playable MBE sidecar support (.imb / .amb).
//
// GopherTrunk's .raw sidecar is a bare concatenation of post-FEC vocoder
// frames — the right shape for BYO decoders, but not directly playable by
// DSD-FME, whose offline player (playMbeFiles) only reads its own container:
// a file that STARTS with a 4-byte ASCII cookie naming the type, followed by
// fixed-size per-frame records (lwvmobile/dsd-fme src/dsd_file.c):
//
//	".imb" — P25 Phase 1 IMBE 4400: [1 errs byte][11 bytes = 88 IMBE bits,
//	         MSB-first] per frame (readImbe4400Data).
//	".amb" — AMBE+2 (DMR / NXDN / P25 Phase 2): [1 errs byte][6 bytes =
//	         bits 0..47 MSB-first][1 byte whose LSB is bit 48] per frame
//	         (readAmbe2450Data).
//
// Both record payloads are exactly the post-FEC information bits GT already
// stores in the .raw sidecar (11-byte IMBE / 7-byte 49-bit AMBE+2, MSB-first,
// DSD/OP25 bit order — see internal/voice/{imbe,ambe2}/testdata/README.md),
// so the sidecar is a repack, not a re-decode. The errs byte is the channel
// error count DSD feeds to mbelib's smoothing; the recorder seam has no
// per-frame error counts, so it writes 0 ("clean"), which mbelib treats as
// no-repair — fine for offline analysis.
//
// TETRA ACELP has no DSD-FME playback mode and ProVoice frames are not in
// mbelib parameter-bit form, so neither gets an MBE sidecar.

// MBE sidecar cookies. The file extension deliberately equals the cookie —
// dsd-fme identifies the container by the leading bytes, and matching names
// keep the on-disk tree self-describing.
const (
	mbeCookieIMB = ".imb"
	mbeCookieAMB = ".amb"
)

// Frame sizes (bytes) of the .raw payloads each cookie repacks.
const (
	imbeRawFrameBytes = 11 // 88 IMBE information bits
	ambeRawFrameBytes = 7  // 49 AMBE+2 information bits (7 pad bits)
)

// mbeCookieForProtocol maps a Grant.Protocol to the dsd-fme container its
// frames belong in; ok=false means the protocol has no dsd-fme playback
// format. p25-phase2 is nominally AMBE+2 3600x2400, but dsd-fme files phase 2
// under .amb alongside DMR/NXDN (openMbeOutFile: "dmr, nxdn, phase 2,
// x2-tdma"), so we match that.
func mbeCookieForProtocol(protocol string) (string, bool) {
	switch protocol {
	case "p25":
		return mbeCookieIMB, true
	case "dmr-tier1", "dmr-tier2", "dmr-tier3", "nxdn", "p25-phase2":
		return mbeCookieAMB, true
	default:
		return "", false
	}
}

// mbeRecord converts one .raw-format frame into the corresponding dsd-fme
// record. Returns nil for a frame whose length doesn't match the cookie's
// expected payload (a malformed frame must not desync the fixed-size record
// stream — dropping it keeps every later record readable).
func mbeRecord(cookie string, frame []byte) []byte {
	switch cookie {
	case mbeCookieIMB:
		if len(frame) != imbeRawFrameBytes {
			return nil
		}
		rec := make([]byte, 0, 1+imbeRawFrameBytes)
		rec = append(rec, 0) // errs
		return append(rec, frame...)
	case mbeCookieAMB:
		if len(frame) != ambeRawFrameBytes {
			return nil
		}
		rec := make([]byte, 0, 8)
		rec = append(rec, 0)            // errs
		rec = append(rec, frame[:6]...) // bits 0..47
		return append(rec, frame[6]>>7) // bit 48 → LSB of the trailing byte
	}
	return nil
}
