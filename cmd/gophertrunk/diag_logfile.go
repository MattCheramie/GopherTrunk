package main

import (
	"github.com/MattCheramie/GopherTrunk/internal/config"
	gtlog "github.com/MattCheramie/GopherTrunk/internal/log"
)

// openDiagLogFile opens the log.file sink, the on-disk copy of the
// daemon's slog output. It returns (nil, nil) when log.file is off. The
// composer/recorder quality lines and every WARN go to stderr only, and
// on Windows or under the TUI operators could not find them (#1242), so
// the file gives support requests one stable place to point to.
func openDiagLogFile(c config.LogFileConfig) (*gtlog.RotatingFile, error) {
	if !c.Enabled || c.Path == "" {
		return nil, nil
	}
	return gtlog.OpenRotatingFile(c.Path, c.MaxSizeMB)
}
