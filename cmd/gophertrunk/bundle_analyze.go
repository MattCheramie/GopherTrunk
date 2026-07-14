package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MattCheramie/GopherTrunk/internal/gtbundle"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// extractBundleCapture pulls the raw IQ and capture metadata out of a bundle to
// a temp working directory so the SigLab engine (which reads a file path) can
// run against it. The returned cleanup removes the temp dir. The metadata may be
// nil if the bundle carries none.
func extractBundleCapture(bundlePath string) (iqPath string, meta *siglab.Metadata, cleanup func(), err error) {
	r, err := gtbundle.Open(bundlePath)
	if err != nil {
		return "", nil, func() {}, err
	}
	tmp, err := os.MkdirTemp("", "gtb-analyze-*")
	if err != nil {
		return "", nil, func() {}, err
	}
	cleanup = func() { os.RemoveAll(tmp) }

	written, err := r.ExtractRole(gtbundle.RoleCaptureIQ, tmp)
	if err != nil {
		cleanup()
		return "", nil, func() {}, fmt.Errorf("bundle has no capture-iq: %w", err)
	}
	iqPath = written[0]

	if m, merr := r.CaptureMeta(); merr == nil {
		meta = m
	}
	return iqPath, meta, cleanup, nil
}

// writeResultToBundle deposits a SigLab result into an existing bundle as
// siglab/result.yaml plus siglab/events.jsonl. It writes the two artifacts to a
// temp dir and folds them in via the bundle add (repack) path.
func writeResultToBundle(bundlePath string, res *siglab.Result) error {
	tmp, err := os.MkdirTemp("", "gtb-result-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	resPath := filepath.Join(tmp, "result.yaml")
	rf, err := os.Create(resPath)
	if err != nil {
		return err
	}
	if werr := siglab.WriteResult(rf, res, siglab.FormatYAML); werr != nil {
		rf.Close()
		return werr
	}
	rf.Close()

	evPath := filepath.Join(tmp, "events.jsonl")
	ef, err := os.Create(evPath)
	if err != nil {
		return err
	}
	if werr := siglab.WriteResult(ef, res, siglab.FormatJSONL); werr != nil {
		ef.Close()
		return werr
	}
	ef.Close()

	if err := bundleAddFile(bundlePath, gtbundle.RoleSiglabResult, resPath, "siglab"); err != nil {
		return err
	}
	return bundleAddFile(bundlePath, gtbundle.RoleSiglabEvents, evPath, "siglab")
}
