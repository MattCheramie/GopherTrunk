package api

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// scrubFloats is a payload type the api package has no DTO case for, so it
// exercises the eventToDTO default passthrough — the path the TETRA identify
// events take.
type scrubFloats struct {
	F64 float64 `json:"f64"`
	F32 float32 `json:"f32"`
	OK  float64 `json:"ok"`
	S   string  `json:"s"`
	N   int     `json:"n"`
}

type scrubEmbedExported struct {
	A float64 `json:"a"`
}

type scrubNested struct {
	scrubEmbedExported                    // embedded
	Ptr                *float64           `json:"ptr"`
	Slice              []float64          `json:"slice"`
	Map                map[string]float64 `json:"map"`
	Inner              scrubFloats        `json:"inner"`
}

// scrubUnexported has an unexported pointer field, mirroring siglab structs that
// stash a reused result (e.g. CandidateScore.result). The scrubber must not
// panic reaching it and must preserve it.
type scrubUnexported struct {
	X   float64 `json:"x"`
	ptr *int
}

func TestEventToDTODefaultPassthroughIsZeroed(t *testing.T) {
	dto := eventToDTO(events.Event{
		Kind:    events.Kind("test/unrecognized"),
		Payload: scrubFloats{F64: math.Inf(1), F32: float32(math.Inf(-1)), OK: 3.5},
	})
	b, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got struct {
		Payload scrubFloats `json:"payload"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Payload.F64 != 0 || got.Payload.F32 != 0 {
		t.Errorf("non-finite floats not zeroed: %+v", got.Payload)
	}
	if got.Payload.OK != 3.5 {
		t.Errorf("finite float altered: got %v, want 3.5", got.Payload.OK)
	}
}

func TestScrubNonFiniteDoesNotMutateOriginal(t *testing.T) {
	orig := &scrubFloats{F64: math.Inf(1)}
	dto := eventToDTO(events.Event{Kind: events.Kind("test/x"), Payload: orig})
	b, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// The shared original — held by other bus subscribers / the store — must be
	// untouched, while the wire copy is zeroed.
	if !math.IsInf(orig.F64, 1) {
		t.Errorf("scrub mutated the shared original: F64 = %v, want +Inf", orig.F64)
	}
	var got struct {
		Payload scrubFloats `json:"payload"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Payload.F64 != 0 {
		t.Errorf("wire copy not zeroed: F64 = %v", got.Payload.F64)
	}
}

func TestScrubNonFiniteNestedContainers(t *testing.T) {
	inf := math.Inf(1)
	v := scrubNested{
		scrubEmbedExported: scrubEmbedExported{A: math.Inf(1)},
		Ptr:                &inf,
		Slice:              []float64{math.Inf(-1), 2.0},
		Map:                map[string]float64{"bad": math.NaN(), "good": 1.0},
		Inner:              scrubFloats{F64: math.NaN(), OK: 7},
	}
	b, err := json.Marshal(scrubNonFinite(v))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got scrubNested
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.A != 0 || got.Ptr == nil || *got.Ptr != 0 || got.Slice[0] != 0 || got.Inner.F64 != 0 || got.Map["bad"] != 0 {
		t.Errorf("nested non-finite not zeroed: %s", b)
	}
	if got.Slice[1] != 2.0 || got.Map["good"] != 1.0 || got.Inner.OK != 7 {
		t.Errorf("nested finite values altered: %s", b)
	}
}

func TestScrubNonFiniteUnexportedFieldSafe(t *testing.T) {
	n := 42
	v := scrubUnexported{X: math.Inf(1), ptr: &n}
	out := scrubNonFinite(v) // must not panic on the unexported pointer
	got, ok := out.(scrubUnexported)
	if !ok {
		t.Fatalf("unexpected type %T", out)
	}
	if got.X != 0 {
		t.Errorf("exported float not zeroed: %v", got.X)
	}
	if got.ptr != &n {
		t.Errorf("unexported field not preserved")
	}
}

func TestScrubNonFiniteLeavesEdgeTypesUnchanged(t *testing.T) {
	now := time.Now()
	if got := scrubNonFinite(now); !got.(time.Time).Equal(now) {
		t.Errorf("time.Time changed: %v != %v", got, now)
	}
	if got := scrubNonFinite(json.Number("3.14")); got.(json.Number) != "3.14" {
		t.Errorf("json.Number changed: %v", got)
	}
	c := complex(1.0, 2.0)
	if got := scrubNonFinite(c); got.(complex128) != c {
		t.Errorf("complex128 changed: %v", got)
	}
	if got := scrubNonFinite(nil); got != nil {
		t.Errorf("nil changed: %v", got)
	}
}

func TestScrubNonFiniteIdentifyResult(t *testing.T) {
	res := &siglab.IdentifyResult{
		Confidence:   math.Inf(1),
		SampleRateHz: 48000,
		TuneHz:       math.Inf(-1),
		Candidates: []siglab.CandidateScore{
			{Protocol: "tetra", Score: math.NaN(), FECPassRate: math.Inf(1), LockLatencySec: 1.2},
		},
	}
	b, err := json.Marshal(scrubNonFinite(res))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got siglab.IdentifyResult
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Confidence != 0 || got.TuneHz != 0 || got.Candidates[0].Score != 0 || got.Candidates[0].FECPassRate != 0 {
		t.Errorf("identify non-finite not zeroed: %s", b)
	}
	if got.SampleRateHz != 48000 || got.Candidates[0].LockLatencySec != 1.2 {
		t.Errorf("identify finite values altered: %s", b)
	}
}
