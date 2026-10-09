package voice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// callMeta is the per-recording metadata sidecar, laid out to match the
// trunk-recorder call `.json` schema so the parsers people already have work
// unchanged. GopherTrunk populates every field it can; the rest carry
// trunk-recorder's own "unknown" conventions (signal/noise = 999, color_code =
// -1) or a best-effort 0 so a strict parser never trips on a missing key.
type callMeta struct {
	CallNum      int    `json:"call_num"`
	Freq         uint32 `json:"freq"`
	FreqError    int    `json:"freq_error"`
	Signal       int    `json:"signal"`
	Noise        int    `json:"noise"`
	SourceNum    int    `json:"source_num"`
	RecorderNum  int    `json:"recorder_num"`
	TDMASlot     int    `json:"tdma_slot"`
	Phase2TDMA   int    `json:"phase2_tdma"`
	StartTime    int64  `json:"start_time"`
	StopTime     int64  `json:"stop_time"`
	StartTimeMs  int64  `json:"start_time_ms"`
	StopTimeMs   int64  `json:"stop_time_ms"`
	Emergency    int    `json:"emergency"`
	Priority     int    `json:"priority"`
	Mode         int    `json:"mode"`
	Duplex       int    `json:"duplex"`
	Encrypted    int    `json:"encrypted"`
	CallLength   int    `json:"call_length"`
	CallLengthMs int64  `json:"call_length_ms"`
	Talkgroup    uint32 `json:"talkgroup"`
	// Individual is a GopherTrunk extension (not in the trunk-recorder schema, so
	// TR parsers ignore it): true marks a unit-to-unit / private call, where
	// `talkgroup` is the target radio's SSI rather than a real talkgroup. Lets a
	// consumer render the recording as an individual call instead of mistaking the
	// radio ID for a talkgroup. Omitted (false) for ordinary group calls.
	Individual bool `json:"individual,omitempty"`
	// AlgorithmID / KeyID are the P25 ALGID / DMR algorithm and key
	// identifier the call's encryption signalling carried (omitted for a
	// clear call) — so a consumer can tell an ADP call from AES without
	// opening the daemon's database.
	AlgorithmID          uint8  `json:"algorithm_id,omitempty"`
	KeyID                uint16 `json:"key_id,omitempty"`
	TalkgroupTag         string `json:"talkgroup_tag"`
	TalkgroupDescription string `json:"talkgroup_description"`
	TalkgroupGroupTag    string `json:"talkgroup_group_tag"`
	TalkgroupGroup       string `json:"talkgroup_group"`
	ColorCode            int    `json:"color_code"`
	AudioType            string `json:"audio_type"`
	ShortName            string `json:"short_name"`
	// Vocoder / FrameBytes are GopherTrunk extensions (TR parsers ignore
	// them): the registry name of the vocoder that decoded this call (e.g.
	// "imbe", "ambe2-dmr") and the fixed byte size of one frame in the .raw
	// sidecar. Together they make the flat .raw self-describing — filesize /
	// FrameBytes = frame count, no config cross-reference needed. Omitted for
	// analog calls and protocols with no in-process vocoder.
	Vocoder    string          `json:"vocoder,omitempty"`
	FrameBytes int             `json:"frame_bytes,omitempty"`
	FreqList   []callFreqEntry `json:"freqList"`
	SrcList    []callSrcEntry  `json:"srcList"`
}

// callFreqEntry mirrors trunk-recorder's freqList element. error_count /
// spike_count have no GopherTrunk source and are emitted as 0.
type callFreqEntry struct {
	Freq       uint32  `json:"freq"`
	Time       int64   `json:"time"`
	Pos        float64 `json:"pos"`
	Len        float64 `json:"len"`
	ErrorCount int     `json:"error_count"`
	SpikeCount int     `json:"spike_count"`
}

// callSrcEntry mirrors trunk-recorder's srcList element — one per talker heard
// in this recording, in order. signal_system / tag / tag_ota are emitted empty
// (GopherTrunk resolves talker aliases elsewhere).
type callSrcEntry struct {
	Src          uint32  `json:"src"`
	Time         int64   `json:"time"`
	Pos          float64 `json:"pos"`
	Emergency    int     `json:"emergency"`
	SignalSystem string  `json:"signal_system"`
	Tag          string  `json:"tag"`
	TagOTA       string  `json:"tag_ota"`
}

// srcEvent / freqEvent are the raw talker / frequency observations accumulated
// on a recordingSession as the call runs; buildCallMeta turns them into the
// pos/len-resolved srcList / freqList above at finalize time.
type srcEvent struct {
	src       uint32
	at        time.Time
	emergency bool
}

type freqEvent struct {
	freq uint32
	at   time.Time
	// audioPos is the recording's decoded-audio length (seconds) when this
	// frequency took effect — the trunk-recorder freqList `pos`.
	audioPos float64
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// buildCallMeta assembles the trunk-recorder metadata for a finished recording
// from the session's grant/talkgroup and its accumulated talker/frequency
// events. startedAt/endedAt bound the file (a per-transmission segment has its
// own span). callNum is the recorder's monotonic call counter.
//
// audioSec is the length of the audio actually in the file. trunk-recorder's
// call_length and freqList pos/len measure the AUDIO (its freqList pos is the
// cumulative transmission length, with no gaps), while start_time/stop_time
// carry the wall-clock span. GopherTrunk used to fill both from the span, so a
// conversation-grouped call held open by a long hangtime claimed a 35 s
// call_length for a 1.6 s WAV (#1242). audioSec <= 0 (unknown) falls back to
// the span, as before.
func buildCallMeta(cs trunking.CallStart, startedAt, endedAt time.Time, callNum int, digital bool, srcs []srcEvent, freqs []freqEvent, audioSec float64) *callMeta {
	g := cs.Grant
	dur := endedAt.Sub(startedAt)
	if audioSec > 0 {
		dur = time.Duration(audioSec * float64(time.Second))
	}
	audioType := "analog"
	if digital {
		audioType = "digital"
	}
	m := &callMeta{
		CallNum:      callNum,
		Freq:         g.FrequencyHz,
		FreqError:    0,
		Signal:       999, // trunk-recorder "unknown" sentinel
		Noise:        999,
		TDMASlot:     int(g.Timeslot),
		StartTime:    startedAt.Unix(),
		StopTime:     endedAt.Unix(),
		StartTimeMs:  startedAt.UnixMilli(),
		StopTimeMs:   endedAt.UnixMilli(),
		Emergency:    boolToInt(g.Emergency),
		Encrypted:    boolToInt(g.Encrypted),
		CallLength:   int(dur.Seconds()),
		CallLengthMs: dur.Milliseconds(),
		Talkgroup:    g.GroupID,
		Individual:   g.Individual,
		AlgorithmID:  g.AlgorithmID,
		KeyID:        g.KeyID,
		ColorCode:    -1, // trunk-recorder "unknown" sentinel
		AudioType:    audioType,
		ShortName:    g.System,
		FreqList:     []callFreqEntry{},
		SrcList:      []callSrcEntry{},
	}
	if tg := cs.Talkgroup; tg != nil {
		m.TalkgroupTag = tg.AlphaTag
		m.TalkgroupDescription = tg.Description
		m.TalkgroupGroupTag = tg.Tag
		m.TalkgroupGroup = tg.Group
		m.Priority = tg.Priority
	}

	// freqList: pos is the audio position the entry starts at; len runs to the
	// next entry (or the end of the audio for the last). Without an audio length
	// both fall back to wall-clock seconds from the file start. TETRA
	// same-carrier calls stay on one freq, so this is usually a single entry
	// spanning the call.
	if len(freqs) == 0 {
		freqs = []freqEvent{{freq: g.FrequencyHz, at: startedAt}}
	}
	for i, fe := range freqs {
		var pos, length float64
		if audioSec > 0 {
			end := audioSec
			if i+1 < len(freqs) {
				end = freqs[i+1].audioPos
			}
			pos, length = fe.audioPos, end-fe.audioPos
		} else {
			end := endedAt
			if i+1 < len(freqs) {
				end = freqs[i+1].at
			}
			pos, length = fe.at.Sub(startedAt).Seconds(), end.Sub(fe.at).Seconds()
		}
		m.FreqList = append(m.FreqList, callFreqEntry{
			Freq: fe.freq,
			Time: fe.at.Unix(),
			Pos:  round2(pos),
			Len:  round2(length),
		})
	}

	// srcList: one entry per talker, in order. pos is seconds from the CALL start
	// (cs.StartedAt), not this file's start — in transmission grouping the list is
	// call-complete, so a talker from an earlier over would otherwise get a negative
	// pos. For a single-transmission call the two starts coincide, so pos is
	// unchanged. Clamp at 0 as a floor against clock skew.
	srcBase := cs.StartedAt
	if srcBase.IsZero() {
		srcBase = startedAt
	}
	for _, se := range srcs {
		pos := se.at.Sub(srcBase).Seconds()
		if pos < 0 {
			pos = 0
		}
		m.SrcList = append(m.SrcList, callSrcEntry{
			Src:       se.src,
			Time:      se.at.Unix(),
			Pos:       round2(pos),
			Emergency: boolToInt(se.emergency),
		})
	}
	return m
}

// writeCallMeta writes m as <wav-basename>.json next to the recording. A nil m
// or empty wavPath is a no-op. The write is small (~1 KB) and is done off the
// recorder lock by the caller.
func writeCallMeta(wavPath string, m *callMeta) error {
	if m == nil || wavPath == "" {
		return nil
	}
	jsonPath := strings.TrimSuffix(wavPath, filepath.Ext(wavPath)) + ".json"
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(jsonPath, b, 0o644)
}
