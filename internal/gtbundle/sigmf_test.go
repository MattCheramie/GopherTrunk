package gtbundle

import (
	"encoding/json"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

func TestMetadataToSigMF(t *testing.T) {
	m := &siglab.Metadata{Protocol: "p25", SampleRateHz: 2_400_000, CenterFreqHz: 851_012_500, Format: "f32", Source: "unit"}
	raw, ok, err := MetadataToSigMF(m)
	if err != nil || !ok {
		t.Fatalf("MetadataToSigMF ok=%v err=%v", ok, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("emitted SigMF is not valid JSON: %v", err)
	}
	global, _ := doc["global"].(map[string]any)
	if global["core:datatype"] != "cf32_le" {
		t.Errorf("datatype = %v, want cf32_le", global["core:datatype"])
	}
	if global["core:sample_rate"].(float64) != 2_400_000 {
		t.Errorf("sample_rate = %v", global["core:sample_rate"])
	}
	caps, _ := doc["captures"].([]any)
	if len(caps) != 1 {
		t.Fatalf("captures = %v", caps)
	}
	if _, has := doc["annotations"]; !has {
		t.Errorf("annotations key missing (SigMF requires it)")
	}
}

func TestMetadataToSigMF_U8(t *testing.T) {
	raw, ok, err := MetadataToSigMF(&siglab.Metadata{Protocol: "dmr", SampleRateHz: 2_400_000, Format: "u8"})
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	var doc map[string]any
	json.Unmarshal(raw, &doc)
	if doc["global"].(map[string]any)["core:datatype"] != "cu8" {
		t.Errorf("u8 → datatype %v, want cu8", doc["global"].(map[string]any)["core:datatype"])
	}
}

func TestMetadataToSigMF_SkipsWAV(t *testing.T) {
	_, ok, err := MetadataToSigMF(&siglab.Metadata{Protocol: "p25", SampleRateHz: 48000, Format: "wav"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ok {
		t.Errorf("wav format should have no SigMF datatype equivalent (ok should be false)")
	}
}
