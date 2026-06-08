package api

import (
	"context"
	"errors"
	"strings"
	"testing"

	gtdiag "github.com/MattCheramie/GopherTrunk/internal/diag"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGRPCDecorateErrPreservesCodeAndGates(t *testing.T) {
	t.Setenv(gtdiag.QuietBannerEnv, "") // banner must render
	orig := status.Error(codes.NotFound, "system not found")

	// Verbose off, no metadata → untouched.
	g := &GRPCServer{diagnostics: gtdiag.NewCollectorWithDongles(nil, nil), verboseErrors: false}
	if got := g.decorateErr(context.Background(), orig); got != orig {
		t.Errorf("verbose off should return the original error, got %v", got)
	}

	// Verbose on → code preserved, banner appended.
	g.verboseErrors = true
	got := g.decorateErr(context.Background(), orig)
	st, _ := status.FromError(got)
	if st.Code() != codes.NotFound {
		t.Errorf("code = %v, want NotFound", st.Code())
	}
	if !strings.Contains(st.Message(), "system not found") {
		t.Errorf("original message lost: %q", st.Message())
	}
	if !strings.Contains(st.Message(), "GopherTrunk diagnostics") {
		t.Errorf("banner not appended: %q", st.Message())
	}
}

func TestGRPCVerboseRequestedViaMetadata(t *testing.T) {
	g := &GRPCServer{diagnostics: gtdiag.NewCollectorWithDongles(nil, nil)}
	if g.verboseRequested(context.Background()) {
		t.Errorf("no metadata, verbose off → should be false")
	}
	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("gophertrunk-verbose", "1"))
	if !g.verboseRequested(ctx) {
		t.Errorf("gophertrunk-verbose=1 metadata should enable verbose")
	}
}

func TestGRPCDecorateNilCollector(t *testing.T) {
	g := &GRPCServer{verboseErrors: true} // no collector
	orig := errors.New("boom")
	if got := g.decorateErr(context.Background(), orig); got != orig {
		t.Errorf("nil collector should leave error untouched, got %v", got)
	}
}
