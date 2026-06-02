package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// BookmarkProvider is the read+write surface the bookmarks endpoints
// consume. The daemon implements it on top of storage.BookmarkStore;
// tests substitute a fake. Decoupling keeps the api package free of a
// hard dependency on internal/storage.
type BookmarkProvider interface {
	ListBookmarks() ([]storage.Bookmark, error)
	GetBookmark(id int64) (storage.Bookmark, error)
	CreateBookmark(b storage.Bookmark) (storage.Bookmark, error)
	UpdateBookmark(b storage.Bookmark) (storage.Bookmark, error)
	DeleteBookmark(id int64) error
}

// BookmarkDTO is the JSON wire shape served by the bookmark
// endpoints. Mirrors storage.Bookmark — kept distinct only so the api
// package can stay free of storage-package imports at the type-name
// level (the interface above does the actual decoupling).
type BookmarkDTO struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	FreqHz    uint32    `json:"freq_hz"`
	Mode      string    `json:"mode"`
	CTCSSHz   float64   `json:"ctcss_hz,omitempty"`
	DCSCode   uint16    `json:"dcs_code,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	Group     string    `json:"group,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func bookmarkToDTO(b storage.Bookmark) BookmarkDTO {
	return BookmarkDTO{
		ID:        b.ID,
		Name:      b.Name,
		FreqHz:    b.FreqHz,
		Mode:      b.Mode,
		CTCSSHz:   b.CTCSSHz,
		DCSCode:   b.DCSCode,
		Notes:     b.Notes,
		Group:     b.Group,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
}

func dtoToBookmark(d BookmarkDTO) storage.Bookmark {
	return storage.Bookmark{
		ID:      d.ID,
		Name:    d.Name,
		FreqHz:  d.FreqHz,
		Mode:    d.Mode,
		CTCSSHz: d.CTCSSHz,
		DCSCode: d.DCSCode,
		Notes:   d.Notes,
		Group:   d.Group,
	}
}

// handleListBookmarks answers GET /api/v1/bookmarks. Open route.
func (s *Server) handleListBookmarks(w http.ResponseWriter, _ *http.Request) {
	if s.bookmarks == nil {
		s.writeError(w, http.StatusServiceUnavailable, "bookmarks store not enabled")
		return
	}
	rows, err := s.bookmarks.ListBookmarks()
	if err != nil {
		s.log.Error("api: list bookmarks", "err", err)
		s.writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	out := make([]BookmarkDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, bookmarkToDTO(r))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleCreateBookmark answers POST /api/v1/bookmarks. Gated.
func (s *Server) handleCreateBookmark(w http.ResponseWriter, r *http.Request) {
	if s.bookmarks == nil {
		s.writeError(w, http.StatusServiceUnavailable, "bookmarks store not enabled")
		return
	}
	var body BookmarkDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	created, err := s.bookmarks.CreateBookmark(dtoToBookmark(body))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, bookmarkToDTO(created))
}

// handleUpdateBookmark answers PATCH /api/v1/bookmarks/{id}. Gated.
func (s *Server) handleUpdateBookmark(w http.ResponseWriter, r *http.Request) {
	if s.bookmarks == nil {
		s.writeError(w, http.StatusServiceUnavailable, "bookmarks store not enabled")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	var body BookmarkDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	body.ID = id
	updated, err := s.bookmarks.UpdateBookmark(dtoToBookmark(body))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "bookmark not found")
			return
		}
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, bookmarkToDTO(updated))
}

// handleDeleteBookmark answers DELETE /api/v1/bookmarks/{id}. Gated.
func (s *Server) handleDeleteBookmark(w http.ResponseWriter, r *http.Request) {
	if s.bookmarks == nil {
		s.writeError(w, http.StatusServiceUnavailable, "bookmarks store not enabled")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.bookmarks.DeleteBookmark(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeError(w, http.StatusNotFound, "bookmark not found")
			return
		}
		s.log.Error("api: delete bookmark", "err", err)
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
