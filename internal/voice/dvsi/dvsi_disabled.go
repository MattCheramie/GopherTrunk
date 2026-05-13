//go:build !dvsi

package dvsi

// Default build: no DVSI registration, no Vocoder type, no Transport.
// VocoderName is exported from doc.go so config validation paths can
// reference the key without -tags dvsi linked in.
