package gtbundle

// Role names one artifact's meaning inside a bundle. Roles are a closed,
// append-only vocabulary: new roles may be added but existing ones are never
// renamed or repurposed, so an old reader still understands an old role in a
// newer bundle. A role a reader does not recognize is tolerated (see the
// forward-compat rules in reader.go) — it is preserved verbatim and bucketed
// under misc/ on a rewrite.
type Role string

const (
	RoleCaptureIQ    Role = "capture-iq"    // capture/<stem>.cfile — raw IQ, source of truth
	RoleCaptureSlice Role = "capture-slice" // capture/<stem>.slice.wav — narrowband DDC slice
	RoleCaptureMeta  Role = "capture-meta"  // capture/<stem>.meta.yaml — siglab.Metadata
	RoleCaptureSigMF Role = "capture-sigmf" // capture/<stem>.sigmf-meta — optional SigMF sidecar

	RoleLogEvents   Role = "log-events"   // logs/events.jsonl — all bus events (EventLog)
	RoleLogMessages Role = "log-messages" // logs/messages.log — human decoded-message log

	RoleMappingSystem Role = "mapping-system" // mapping/system.json — hunt.DiscoveredSystem
	RoleMappingSurvey Role = "mapping-survey" // mapping/survey.jsonl — hunt.DetectedSignal stream
	RoleMappingExport Role = "mapping-export" // mapping/system.bundle.csv | .trunk-recorder.json | .rr.md

	RoleSiglabResult Role = "siglab-result" // siglab/result.yaml — siglab.Result
	RoleSiglabEvents Role = "siglab-events" // siglab/events.jsonl — siglab events/PDUs

	RoleCryptolabFrames Role = "cryptolab-frames" // cryptolab/frames.jsonl — cryptocap records
	RoleCryptolabResult Role = "cryptolab-result" // cryptolab/assess.result.yaml — cryptolab.Result

	RoleNotes  Role = "notes"  // notes.md — free-form operator notes
	RoleReadme Role = "readme" // README.md — auto-generated summary
)

// knownRoles is the set of roles this build understands. Order is the canonical
// write order used by the Writer for deterministic archives (§ writer.go).
var knownRoles = []Role{
	RoleReadme,
	RoleNotes,
	RoleCaptureIQ,
	RoleCaptureSlice,
	RoleCaptureMeta,
	RoleCaptureSigMF,
	RoleLogEvents,
	RoleLogMessages,
	RoleMappingSystem,
	RoleMappingSurvey,
	RoleMappingExport,
	RoleSiglabResult,
	RoleSiglabEvents,
	RoleCryptolabFrames,
	RoleCryptolabResult,
}

// roleDir maps a role to the subdirectory it lives in. Roles that live at the
// bundle root (readme, notes) map to "".
var roleDir = map[Role]string{
	RoleCaptureIQ:       "capture",
	RoleCaptureSlice:    "capture",
	RoleCaptureMeta:     "capture",
	RoleCaptureSigMF:    "capture",
	RoleLogEvents:       "logs",
	RoleLogMessages:     "logs",
	RoleMappingSystem:   "mapping",
	RoleMappingSurvey:   "mapping",
	RoleMappingExport:   "mapping",
	RoleSiglabResult:    "siglab",
	RoleSiglabEvents:    "siglab",
	RoleCryptolabFrames: "cryptolab",
	RoleCryptolabResult: "cryptolab",
	RoleNotes:           "",
	RoleReadme:          "",
}

// roleMediaType maps a role to a sensible media type for its FileEntry, so a
// consumer (e.g. a web UI) can render or offer a download without sniffing.
var roleMediaType = map[Role]string{
	RoleCaptureIQ:       "application/octet-stream",
	RoleCaptureSlice:    "audio/wav",
	RoleCaptureMeta:     "application/yaml",
	RoleCaptureSigMF:    "application/json",
	RoleLogEvents:       "application/x-ndjson",
	RoleLogMessages:     "text/plain",
	RoleMappingSystem:   "application/json",
	RoleMappingSurvey:   "application/x-ndjson",
	RoleMappingExport:   "application/octet-stream",
	RoleSiglabResult:    "application/yaml",
	RoleSiglabEvents:    "application/x-ndjson",
	RoleCryptolabFrames: "application/x-ndjson",
	RoleCryptolabResult: "application/yaml",
	RoleNotes:           "text/markdown",
	RoleReadme:          "text/markdown",
}

// miscDir is where an unknown role's file is placed when a newer bundle is
// rewritten by an older reader that does not recognize the role.
const miscDir = "misc"

// Valid reports whether r is a role this build understands.
func (r Role) Valid() bool {
	_, ok := roleDir[r]
	return ok
}

// dir returns the subdirectory for r ("" for root-level roles); an unknown role
// falls back to miscDir so its file is never lost.
func (r Role) dir() string {
	if d, ok := roleDir[r]; ok {
		return d
	}
	return miscDir
}

// mediaType returns the media type for r, or a generic default for an unknown
// role.
func (r Role) mediaType() string {
	if m, ok := roleMediaType[r]; ok {
		return m
	}
	return "application/octet-stream"
}

// roleWriteRank gives the canonical ordering index for a role (for
// deterministic archives). Unknown roles sort last, stably by name.
func roleWriteRank(r Role) int {
	for i, kr := range knownRoles {
		if kr == r {
			return i
		}
	}
	return len(knownRoles)
}
