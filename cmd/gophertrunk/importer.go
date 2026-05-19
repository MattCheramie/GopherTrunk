package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/api"
)

// daemonImporter implements api.Importer by writing each upload to a
// tempfile and calling the existing parsePDFFile / parseCSVFile /
// mergeIntoConfig helpers in cmd/gophertrunk. Sharing the mutex
// with the config writer would be ideal but the writer's lock is
// scoped to itself; this importer adds its own coarse lock so two
// commits can't race against each other.
type daemonImporter struct {
	d  *Daemon
	mu sync.Mutex
}

func newDaemonImporter(d *Daemon) *daemonImporter {
	return &daemonImporter{d: d}
}

// Parse turns one uploaded source into a preview DTO. The source's
// bytes are written to a tempfile under os.TempDir; the file is
// removed after the parser returns so failure paths don't leak.
func (i *daemonImporter) Parse(s api.ImportSource) (api.ParsedSystemDTO, error) {
	path, err := writeTempUpload(s)
	if err != nil {
		return api.ParsedSystemDTO{}, err
	}
	defer os.Remove(path)

	parsed, err := parseImportFile(path, s.Kind)
	if err != nil {
		return api.ParsedSystemDTO{}, err
	}
	return parsedToDTO(parsed, s.Filename), nil
}

// Commit writes every source out to disk (so mergeIntoConfig can
// re-parse them via the existing file-path code path) and merges
// them into the daemon's config.yaml.
func (i *daemonImporter) Commit(sources []api.ImportSource, force bool) (api.ImportCommitResult, error) {
	if i.d.cfgPath == "" {
		return api.ImportCommitResult{}, fmt.Errorf("import: daemon started without -config; nothing to merge into")
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	var tmpPaths []string
	defer func() {
		for _, p := range tmpPaths {
			_ = os.Remove(p)
		}
	}()

	parsed := make([]parsedSystem, 0, len(sources))
	for _, s := range sources {
		tp, err := writeTempUpload(s)
		if err != nil {
			return api.ImportCommitResult{}, err
		}
		tmpPaths = append(tmpPaths, tp)
		ps, err := parseImportFile(tp, s.Kind)
		if err != nil {
			return api.ImportCommitResult{}, err
		}
		parsed = append(parsed, ps)
	}

	res, err := mergeIntoConfig(parsed, mergeOptions{
		ConfigPath: i.d.cfgPath,
		CSVDir:     filepath.Dir(i.d.cfgPath),
		Force:      force,
		DryRun:     false,
	})
	if err != nil {
		return api.ImportCommitResult{}, err
	}

	added := make([]string, 0, len(parsed))
	for _, p := range parsed {
		added = append(added, p.Name)
	}
	csvPaths := make([]string, 0, len(res.CSVs))
	for _, c := range res.CSVs {
		csvPaths = append(csvPaths, c.Path)
	}

	// Reload talkgroup CSVs into the in-memory DB so the new alpha
	// tags appear without a daemon restart.
	for _, c := range res.CSVs {
		if i.d.talkgroups != nil {
			_, _ = i.d.talkgroups.LoadCSVFile(c.Path)
		}
	}

	return api.ImportCommitResult{
		SystemsAdded: added,
		CSVPaths:     csvPaths,
		ConfigPath:   i.d.cfgPath,
	}, nil
}

// writeTempUpload writes s.Data to a temp file (extension preserved
// so the parser sees the original filename hint) and returns the path.
func writeTempUpload(s api.ImportSource) (string, error) {
	ext := filepath.Ext(s.Filename)
	if ext == "" {
		switch s.Kind {
		case api.ImportSourcePDF:
			ext = ".pdf"
		case api.ImportSourceCSV:
			ext = ".csv"
		}
	}
	f, err := os.CreateTemp("", "gophertrunk-import-*"+ext)
	if err != nil {
		return "", fmt.Errorf("import: tempfile: %w", err)
	}
	if _, err := f.Write(s.Data); err != nil {
		f.Close()
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("import: write tempfile: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func parseImportFile(path string, kind api.ImportSourceKind) (parsedSystem, error) {
	switch kind {
	case api.ImportSourcePDF:
		return parsePDFFile(path)
	case api.ImportSourceCSV:
		return parseCSVFile(path, csvImportOpts{})
	}
	return parsedSystem{}, fmt.Errorf("import: unsupported kind %q", kind)
}

func parsedToDTO(p parsedSystem, filename string) api.ParsedSystemDTO {
	return api.ParsedSystemDTO{
		Name:        p.Name,
		Location:    p.Location,
		County:      p.County,
		SysID:       p.SysID,
		WACN:        p.WACN,
		SystemType:  p.SystemType,
		Protocol:    p.Protocol,
		SiteCount:   len(p.Sites),
		TalkgroupCt: len(p.Talkgroups),
		SourcePath:  filename,
	}
}
