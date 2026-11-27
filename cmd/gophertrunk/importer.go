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

	parsed, err := parseImportFile(path, s.Kind, s.Filename)
	if err != nil {
		return api.ParsedSystemDTO{}, err
	}
	return parsedToDTO(parsed, s.Filename), nil
}

// ParseAll parses every source of one upload and returns its previews. It
// differs from calling Parse per file in one way: a system's RadioReference
// talkgroup CSV and sites CSV, uploaded together, come back as ONE system
// with both control channels and talkgroups (#1403). An error names the
// file it came from.
func (i *daemonImporter) ParseAll(sources []api.ImportSource) ([]api.ParsedSystemDTO, error) {
	parsed, err := parseImportSources(sources)
	if err != nil {
		return nil, err
	}
	out := make([]api.ParsedSystemDTO, 0, len(parsed))
	for _, p := range parsed {
		out = append(out, parsedToDTO(p, p.SourcePath))
	}
	return out, nil
}

// parseImportSources parses each upload from a tempfile, labels it with the
// name it was uploaded under, and joins RadioReference CSV pairs.
func parseImportSources(sources []api.ImportSource) ([]parsedSystem, error) {
	parsed := make([]parsedSystem, 0, len(sources))
	for _, s := range sources {
		tp, err := writeTempUpload(s)
		if err != nil {
			return nil, err
		}
		ps, err := parseImportFile(tp, s.Kind, s.Filename)
		_ = os.Remove(tp)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", s.Filename, err)
		}
		ps.SourcePath = s.Filename
		parsed = append(parsed, ps)
	}
	return pairRRNativeCSVs(parsed), nil
}

// Commit parses every source and merges the result into the daemon's
// config.yaml.
func (i *daemonImporter) Commit(sources []api.ImportSource, force bool) (api.ImportCommitResult, error) {
	if i.d.cfgPath == "" {
		return api.ImportCommitResult{}, fmt.Errorf("import: daemon started without -config; nothing to merge into")
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	// Parsed the same way the preview was, so what is merged is what the
	// operator was shown (RadioReference CSV pairs joined).
	parsed, err := parseImportSources(sources)
	if err != nil {
		return api.ImportCommitResult{}, err
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

// parseImportFile parses the file at path. filename is the name the
// operator gave the source (the upload's original filename): uploads are
// parsed from a tempfile, so a native RadioReference CSV, which carries
// no system name, takes its default name from filename rather than from
// the tempfile's "gophertrunk-import-<n>" stem.
func parseImportFile(path string, kind api.ImportSourceKind, filename string) (parsedSystem, error) {
	switch kind {
	case api.ImportSourcePDF:
		return parsePDFFile(path)
	case api.ImportSourceCSV:
		var opts csvImportOpts
		if filename != "" {
			opts.Name = filenameStem(filename)
		}
		return parseCSVFile(path, opts)
	}
	return parsedSystem{}, fmt.Errorf("import: unsupported kind %q", kind)
}

func parsedToDTO(p parsedSystem, filename string) api.ParsedSystemDTO {
	// Collapse every site's control-channel frequencies into one ordered,
	// de-duplicated list (the config schema is flat — one system carries
	// all its control channels). Voice frequencies are left to the
	// committed talkgroup/band-plan flow.
	var ccs []uint32
	seen := make(map[uint32]struct{})
	for _, site := range p.Sites {
		for _, f := range site.Frequencies {
			if !f.ControlChannel {
				continue
			}
			if _, dup := seen[f.Hz]; dup {
				continue
			}
			seen[f.Hz] = struct{}{}
			ccs = append(ccs, f.Hz)
		}
	}
	tgs := make([]api.ImportTalkgroupDTO, 0, len(p.Talkgroups))
	for _, tg := range p.Talkgroups {
		tgs = append(tgs, api.ImportTalkgroupDTO{
			Decimal:     tg.Dec,
			AlphaTag:    tg.AlphaTag,
			Description: tg.Description,
			Tag:         tg.Tag,
			Group:       tg.Group,
			Mode:        tg.Mode,
			Encrypted:   tg.Encrypted,
		})
	}
	return api.ParsedSystemDTO{
		Name:            p.Name,
		Location:        p.Location,
		County:          p.County,
		SysID:           p.SysID,
		WACN:            p.WACN,
		SystemType:      p.SystemType,
		Protocol:        p.Protocol,
		SiteCount:       len(p.Sites),
		TalkgroupCt:     len(p.Talkgroups),
		SourcePath:      filename,
		ControlChannels: ccs,
		Talkgroups:      tgs,
	}
}
