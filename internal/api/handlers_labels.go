package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/storage"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// LabelProvider is the persistence surface behind operator-applied names. The
// daemon implements it over storage.LabelStore; tests substitute a fake.
// Decoupling keeps the api package free of a hard storage dependency, matching
// BookmarkProvider.
//
// Every use is nil-guarded: a daemon with no storage.path keeps the previous
// in-memory-only behaviour rather than failing a rename.
type LabelProvider interface {
	ListLabels(kind storage.LabelKind, system string) ([]storage.Label, error)
	UpsertLabel(l storage.Label) (storage.Label, error)
	DeleteLabel(kind storage.LabelKind, system string, id uint32) error
}

// persistRIDLabel saves the operator-applied fields of a radio record. A record
// carrying none of them is deleted instead, so clearing a name removes the row
// rather than persisting an empty one that would shadow the alias file.
func (s *Server) persistRIDLabel(system string, rec *trunking.RID) {
	if s.labels == nil || rec == nil {
		return
	}
	l := storage.Label{
		Kind: storage.LabelKindRID, System: strings.TrimSpace(system), TargetID: rec.ID,
		Name: rec.Alias, Description: rec.Description, Tag: rec.Tag,
		Group: rec.Group, Owner: rec.Owner, Icon: rec.Icon,
	}
	s.saveLabel(l)
}

// persistTalkgroupLabel is the talkgroup half of persistRIDLabel.
func (s *Server) persistTalkgroupLabel(system string, tg *trunking.TalkGroup) {
	if s.labels == nil || tg == nil {
		return
	}
	l := storage.Label{
		Kind: storage.LabelKindTalkgroup, System: strings.TrimSpace(system), TargetID: tg.ID,
		Name: tg.AlphaTag, Description: tg.Description, Tag: tg.Tag,
		Group: tg.Group, Icon: tg.Icon,
	}
	s.saveLabel(l)
}

func (s *Server) saveLabel(l storage.Label) {
	if l.Empty() {
		if err := s.labels.DeleteLabel(l.Kind, l.System, l.TargetID); err != nil && !isNoRows(err) {
			s.log.Warn("api: delete label", "kind", l.Kind, "id", l.TargetID, "err", err)
		}
		return
	}
	if _, err := s.labels.UpsertLabel(l); err != nil {
		// The in-memory mutation already succeeded and the response is about
		// to say so, so this is a warning rather than a failed request — but
		// it means the name will not survive a restart, which the operator
		// needs to see.
		s.log.Warn("api: persist label — the name will not survive a restart",
			"kind", l.Kind, "id", l.TargetID, "err", err)
	}
}

func isNoRows(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no rows")
}

// handleListLabels returns the persisted operator labels.
//
//	GET /api/v1/labels?kind=rid|talkgroup&system=<name>
func (s *Server) handleListLabels(w http.ResponseWriter, r *http.Request) {
	if s.labels == nil {
		s.writeError(w, http.StatusServiceUnavailable,
			"label persistence is not enabled (set storage.path)")
		return
	}
	kind, err := optionalLabelKind(r.URL.Query().Get("kind"))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rows, err := s.labels.ListLabels(kind, r.URL.Query().Get("system"))
	if err != nil {
		s.log.Warn("api: list labels failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "label query failed")
		return
	}
	if rows == nil {
		rows = []storage.Label{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"labels": rows})
}

// handleDeleteLabel forgets one persisted label. The in-memory catalogue keeps
// whatever it currently holds until the next restart — the row is what governs
// what comes back, so removing it is the durable "revert to the alias file".
//
//	DELETE /api/v1/labels/{kind}/{id}?system=<name>
func (s *Server) handleDeleteLabel(w http.ResponseWriter, r *http.Request) {
	if s.labels == nil {
		s.writeError(w, http.StatusServiceUnavailable,
			"label persistence is not enabled (set storage.path)")
		return
	}
	kind, err := storage.ParseLabelKind(r.PathValue("kind"))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid kind (want rid or talkgroup)")
		return
	}
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.labels.DeleteLabel(kind, r.URL.Query().Get("system"), uint32(id)); err != nil {
		if isNoRows(err) {
			s.writeError(w, http.StatusNotFound, "no such label")
			return
		}
		s.log.Warn("api: delete label failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "label delete failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleExportLabels serves the catalogue as a CSV that drops straight into a
// system's talkgroup_file / rid_alias_file, so operator-applied names can be
// folded back into the files an operator hand-maintains.
//
//	GET /api/v1/labels/export?kind=rid|talkgroup&system=<name>&scope=labels|all
//
// scope=labels (default) exports only the ids that carry a persisted label;
// scope=all exports the whole catalogue. Rows are built from the live
// catalogue — labels are already merged into it at startup — so the policy
// columns (priority, lockout, scan, watch) come out correct even though they
// are not themselves persisted.
func (s *Server) handleExportLabels(w http.ResponseWriter, r *http.Request) {
	kindStr := r.URL.Query().Get("kind")
	if kindStr == "" {
		s.writeError(w, http.StatusBadRequest, "kind is required (rid or talkgroup)")
		return
	}
	kind, err := storage.ParseLabelKind(kindStr)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid kind (want rid or talkgroup)")
		return
	}
	system := r.URL.Query().Get("system")
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "labels"
	}
	if scope != "labels" && scope != "all" {
		s.writeError(w, http.StatusBadRequest, "invalid scope (want labels or all)")
		return
	}
	if scope == "labels" && s.labels == nil {
		s.writeError(w, http.StatusServiceUnavailable,
			"label persistence is not enabled (set storage.path); use scope=all to export the catalogue")
		return
	}

	// The set of ids to emit. Empty when exporting everything.
	var only map[uint32]bool
	if scope == "labels" {
		rows, err := s.labels.ListLabels(kind, system)
		if err != nil {
			s.log.Warn("api: list labels failed", "err", err)
			s.writeError(w, http.StatusInternalServerError, "label query failed")
			return
		}
		only = make(map[uint32]bool, len(rows))
		for _, l := range rows {
			only[l.TargetID] = true
		}
	}

	filename := fmt.Sprintf("gophertrunk-%s-%s.csv", kindStr, exportScopeName(system))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	switch kind {
	case storage.LabelKindTalkgroup:
		if s.talkgroups == nil {
			s.writeError(w, http.StatusServiceUnavailable, "talkgroup catalogue not wired")
			return
		}
		var out []*trunking.TalkGroup
		for _, tg := range s.talkgroups.All() {
			if only == nil || only[tg.ID] {
				out = append(out, tg)
			}
		}
		err = trunking.WriteTalkgroupCSV(w, out)
	default:
		if s.rids == nil {
			s.writeError(w, http.StatusServiceUnavailable, "rid catalogue not wired")
			return
		}
		var out []*trunking.RID
		for _, rec := range s.rids.All() {
			if only == nil || only[rec.ID] {
				out = append(out, rec)
			}
		}
		err = trunking.WriteRIDCSV(w, out)
	}
	if err != nil {
		// Headers are already sent, so this can only be logged.
		s.log.Warn("api: label export failed mid-write", "err", err)
	}
}

// exportScopeName makes a filesystem-safe fragment out of the system filter.
func exportScopeName(system string) string {
	if system == "" {
		return "all"
	}
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_':
			return r
		}
		return '_'
	}, system)
	return safe
}

func optionalLabelKind(s string) (storage.LabelKind, error) {
	if s == "" {
		return "", nil
	}
	k, err := storage.ParseLabelKind(s)
	if err != nil {
		return "", errors.New("invalid kind (want rid or talkgroup)")
	}
	return k, nil
}
