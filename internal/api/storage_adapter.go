package api

import (
	"context"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// LocationsFromStorage wraps a *storage.LocationLog as an
// api.LocationQuery so the daemon can pass it to NewServer without the
// storage package's row types leaking into the api package. loc is the
// display timezone the ReportedAt timestamp renders in (nil → time.Local).
func LocationsFromStorage(ll *storage.LocationLog, loc *time.Location) LocationQuery {
	if ll == nil {
		return nil
	}
	if loc == nil {
		loc = time.Local
	}
	return &storageLocations{ll: ll, loc: loc}
}

type storageLocations struct {
	ll  *storage.LocationLog
	loc *time.Location
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
			ReportedAt: r.ReportedAt.In(s.loc).Format(time.RFC3339),
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

// RIDSummaries delegates to storage.DB.RIDSummaries, mapping the storage row
// type to the api one. This makes storageHistory a RIDSummaryProvider, so the
// Radio IDs list is backed by the persisted call log.
func (s *storageHistory) RIDSummaries(ctx context.Context, f RIDSummaryFilter) ([]RIDSummary, error) {
	rows, err := s.db.RIDSummaries(ctx, storage.RIDSummaryFilter{
		System:   f.System,
		SourceID: f.SourceID,
		Limit:    f.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]RIDSummary, len(rows))
	for i, r := range rows {
		out[i] = RIDSummary{
			SourceID:      r.SourceID,
			System:        r.System,
			Protocol:      r.Protocol,
			LastTalkgroup: r.LastTalkgroup,
			CallCount:     r.CallCount,
			FirstSeen:     r.FirstSeen,
			LastSeen:      r.LastSeen,
			SourceAlpha:   r.SourceAlpha,
		}
	}
	return out, nil
}

func (s *storageHistory) History(ctx context.Context, f HistoryFilter) ([]CallRow, error) {
	rows, err := s.db.History(ctx, storage.HistoryFilter{
		IDs:       f.IDs,
		System:    f.System,
		GroupID:   f.GroupID,
		SourceID:  f.SourceID,
		Since:     f.Since,
		Until:     f.Until,
		Limit:     f.Limit,
		OnlyEnded: f.OnlyEnded,
		Query:     f.Query,
		Protocol:  f.Protocol,
		Encrypted: f.Encrypted,
		Emergency: f.Emergency,
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
			AlgorithmID:    r.AlgorithmID,
			KeyID:          r.KeyID,
			Emergency:      r.Emergency,
			DataCall:       r.DataCall,
			Individual:     r.Individual,
			Timeslot:       r.Timeslot,
			Priority:       r.Priority,
			DeviceSerial:   r.DeviceSerial,
			StartedAt:      r.StartedAt,
			EndedAt:        r.EndedAt,
			DurationMs:     r.DurationMs,
			EndReason:      r.EndReason,
			TalkgroupAlpha: r.TalkgroupAlpha,
			SourceAlpha:    r.SourceAlpha,
			SignalDbFS:     r.SignalDbFS,
			EVMPct:         r.EVMPct,
			SNRDb:          r.SNRDb,
			HasRecording:   r.HasRecording,
			Transcript:     r.Transcript,
		}
	}
	return out, nil
}

// RecordingPathByID delegates to storage.DB so the audio endpoint can resolve a
// call id to a file. Presence of this method makes storageHistory a
// RecordingProvider (see server.go).
func (s *storageHistory) RecordingPathByID(ctx context.Context, id int64) (string, error) {
	return s.db.RecordingPathByID(ctx, id)
}

// RecordingSegmentsByID delegates to storage.DB so the audio endpoint can play
// every segment of a multi-over call (RecordingSegmentsProvider).
func (s *storageHistory) RecordingSegmentsByID(ctx context.Context, id int64) ([]RecordingSegment, error) {
	segs, err := s.db.RecordingSegmentsByID(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]RecordingSegment, len(segs))
	for i, seg := range segs {
		out[i] = RecordingSegment{Seq: seg.Seq, Path: seg.Path, StartedAt: seg.StartedAt, EndedAt: seg.EndedAt}
	}
	return out, nil
}

// RecordingFeed delegates to storage.DB.RecordingFeed and attaches each
// recording's call row (RecordingFeedProvider).
func (s *storageHistory) RecordingFeed(ctx context.Context, afterID int64, limit int) ([]FinalizedRecording, error) {
	recs, err := s.db.RecordingFeed(ctx, afterID, limit)
	if err != nil || len(recs) == 0 {
		return nil, err
	}
	ids := make([]int64, 0, len(recs))
	seen := map[int64]bool{}
	for _, r := range recs {
		if !seen[r.CallID] {
			seen[r.CallID] = true
			ids = append(ids, r.CallID)
		}
	}
	calls, err := s.History(ctx, HistoryFilter{IDs: ids, Limit: len(ids)})
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]CallRow, len(calls))
	for _, c := range calls {
		byID[c.ID] = c
	}
	out := make([]FinalizedRecording, len(recs))
	for i, r := range recs {
		out[i] = FinalizedRecording{
			ID: r.ID, CallID: r.CallID, Segment: r.Seq,
			StartedAt: r.StartedAt, EndedAt: r.EndedAt, path: r.Path,
			Partial: r.IQLostChunks > 0, IQLostChunks: r.IQLostChunks,
		}
		if c, ok := byID[r.CallID]; ok {
			c := c
			out[i].Call = &c
		}
	}
	return out, nil
}

// RecordingFileByID delegates to storage.DB.RecordingPathByRecordingID.
func (s *storageHistory) RecordingFileByID(ctx context.Context, id int64) (string, error) {
	return s.db.RecordingPathByRecordingID(ctx, id)
}
