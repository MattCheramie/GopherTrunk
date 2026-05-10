//go:build !(linux && (amd64 || arm64 || 386 || arm || riscv64 || loong64)) && !(windows && (amd64 || arm64))

package usb

// platformEnumerator returns a stub [Enumerator] whose every method
// reports [ErrUnsupportedPlatform]. PR-10 replaces this on macOS with a
// real IOKit-via-purego implementation.
func platformEnumerator() Enumerator { return unsupportedEnumerator{} }

type unsupportedEnumerator struct{}

func (unsupportedEnumerator) Name() string { return "unsupported" }

func (unsupportedEnumerator) List(vid, pid uint16) ([]Descriptor, error) {
	return nil, ErrUnsupportedPlatform
}

func (unsupportedEnumerator) Open(Descriptor) (Transport, error) {
	return nil, ErrUnsupportedPlatform
}
