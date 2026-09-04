package terms

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The canonical document is TERMS_OF_SERVICE.md at the repository root;
// go:embed cannot reach above the package directory, so the package
// carries a copy. This pin is what keeps the two from drifting apart —
// edit the root document, then `cp` it over internal/terms/'s copy.
func TestEmbeddedTermsMatchCanonicalDocument(t *testing.T) {
	canonical, err := os.ReadFile(filepath.Join("..", "..", "TERMS_OF_SERVICE.md"))
	if err != nil {
		t.Fatalf("read canonical TERMS_OF_SERVICE.md: %v", err)
	}
	if string(canonical) != Text {
		t.Fatalf("internal/terms/TERMS_OF_SERVICE.md has drifted from the root TERMS_OF_SERVICE.md; copy the root file over the embedded copy (and bump terms.Version if the change is material)")
	}
}

// The document announces the version the binary enforces; a bump to the
// Version const without updating the document (or vice versa) would show
// operators one revision and record another.
func TestTermsDocumentStatesCurrentVersion(t *testing.T) {
	if !strings.Contains(Text, fmt.Sprintf("**Version %d ", Version)) {
		t.Fatalf("TERMS_OF_SERVICE.md does not state 'Version %d'; keep the document header and terms.Version in sync (and TermsVersion in installer/windows/gophertrunk.iss)", Version)
	}
}

// The Inno Setup memo control round-trips the document through an ANSI
// string, so any non-ASCII byte would render as mojibake on the
// installer's Terms page.
func TestTermsDocumentIsASCII(t *testing.T) {
	for i := 0; i < len(Text); i++ {
		if Text[i] > 0x7f {
			t.Fatalf("TERMS_OF_SERVICE.md byte %d (%q) is non-ASCII; the Windows installer's terms page renders the file through an ANSI memo control", i, Text[i])
		}
	}
}

func TestRecordAndAcceptedVersionRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "GopherTrunk")
	if IsAccepted(dir) {
		t.Fatal("empty dir should not read as accepted")
	}
	if err := Record(dir, "test"); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if got := AcceptedVersion(dir); got != Version {
		t.Fatalf("AcceptedVersion = %d, want %d", got, Version)
	}
	if !IsAccepted(dir) {
		t.Fatal("IsAccepted = false after Record")
	}
}

// A marker from an older terms revision must force re-acknowledgment.
func TestOlderMarkerVersionIsNotAccepted(t *testing.T) {
	dir := t.TempDir()
	body := "version=0\naccepted_at=2026-01-01T00:00:00Z\n"
	if err := os.WriteFile(MarkerPath(dir), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if IsAccepted(dir) {
		t.Fatal("version=0 marker should not satisfy the current terms version")
	}
}

// The Windows installer writes the marker with CRLF line endings and its
// own field order; the parser must accept it.
func TestAcceptedVersionParsesInstallerStyleMarker(t *testing.T) {
	dir := t.TempDir()
	body := "# GopherTrunk terms-of-service acceptance record (see TERMS_OF_SERVICE.md).\r\n" +
		"version=1\r\naccepted_at=2026-09-04 12:00:00\r\naccepted_via=windows-installer\r\n"
	if err := os.WriteFile(MarkerPath(dir), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := AcceptedVersion(dir); got != 1 {
		t.Fatalf("AcceptedVersion = %d, want 1", got)
	}
}

func TestEnvAccepted(t *testing.T) {
	for val, want := range map[string]bool{
		"1": true, "true": true, "YES": true, "y": true,
		"": false, "0": false, "no": false, "maybe": false,
	} {
		t.Setenv(EnvVar, val)
		if got := EnvAccepted(); got != want {
			t.Errorf("EnvAccepted with %s=%q = %v, want %v", EnvVar, val, got, want)
		}
	}
}
