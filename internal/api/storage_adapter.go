package api

import (
	"context"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// LocationsFromStorage wraps a *storage.LocationLog as an
// api.LocationQuery so the daemon can pass it to NewServer without the
// storage package's row types leaking into the api package.
func LocationsFromStorage(ll *storage.LocationLog) LocationQuery {
	if ll == nil {
		return nil
	}
	return &storageLocations{ll: ll}
}

type storageLocations struct {
	ll *storage.LocationLog
}

func (s *storageLocations) RecentLocations(limit int) ([]LocationFix, error) {
	rows, err := s.ll.Recent(limit)
	if err != nil {
		return nil, err
	}
	out := make([]LocationFix, len(rows))
	for i, r := range rows {
		out[i] = LocationFix{
			System:     r.System,
			Protocol:   r.Protocol,
			RadioID:    r.RadioID,
			Talkgroup:  r.Talkgroup,
			Latitude:   r.Latitude,
			Longitude:  r.Longitude,
			SpeedKnots: r.SpeedKnots,
			HeadingDeg: r.HeadingDeg,
			ReportedAt: r.ReportedAt.UTC().Format(time.RFC3339),
		}
	}
	return out, nil
}

// HistoryFromStorage wraps a *storage.DB as an api.HistoryQuery so the
// daemon can pass it to NewServer without the api package's CallRow /
// HistoryFilter types leaking into the storage package.
func HistoryFromStorage(db *storage.DB) HistoryQuery {
	if db == nil {
		return nil
	}
	return &storageHistory{db: db}
}

type storageHistory struct {
	db *storage.DB
}

func (s *storageHistory) History(ctx context.Context, f HistoryFilter) ([]CallRow, error) {
	rows, err := s.db.History(ctx, storage.HistoryFilter{
		System:    f.System,
		GroupID:   f.GroupID,
		Since:     f.Since,
		Until:     f.Until,
		Limit:     f.Limit,
		OnlyEnded: f.OnlyEnded,
	})
	if err != nil {
		return nil, err
	}
	out := make([]CallRow, len(rows))
	for i, r := range rows {
		out[i] = CallRow{
			ID:             r.ID,
			System:         r.System,
			Protocol:       r.Protocol,
			GroupID:        r.GroupID,
			SourceID:       r.SourceID,
			FrequencyHz:    r.FrequencyHz,
			Encrypted:      r.Encrypted,
			Emergency:      r.Emergency,
			DataCall:       r.DataCall,
			DeviceSerial:   r.DeviceSerial,
			StartedAt:      r.StartedAt,
			EndedAt:        r.EndedAt,
			DurationMs:     r.DurationMs,
			EndReason:      r.EndReason,
			TalkgroupAlpha: r.TalkgroupAlpha,
		}
	}
	return out, nil
}
