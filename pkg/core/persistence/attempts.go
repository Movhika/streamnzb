package persistence

import (
	"database/sql"
	"time"
)

// NZBAttempt represents one recorded play attempt (preload, success, or failure).
type NZBAttempt struct {
	ID            int64     `json:"id"`
	TriedAt       time.Time `json:"tried_at"`
	ContentType   string    `json:"content_type"`
	ContentID     string    `json:"content_id"`
	ContentTitle  string    `json:"content_title"`
	ReleaseTitle  string    `json:"release_title"`
	ReleaseURL    string    `json:"release_url"`
	ReleaseSize   int64     `json:"release_size"`
	ServedFile    string    `json:"served_file,omitempty"`
	Success       bool      `json:"success"`
	FailureReason string    `json:"failure_reason,omitempty"`
	SlotPath      string    `json:"slot_path,omitempty"`
	Preload       bool      `json:"preload"` // true = attempt started, result not yet known
}

// RecordAttemptParams holds the fields needed to record an NZB attempt.
type RecordAttemptParams struct {
	ContentType   string // "movie" or "series"
	ContentID     string // e.g. "tt123" or "tmdb:123:1:2"
	ContentTitle  string
	ReleaseTitle  string
	ReleaseURL    string
	ReleaseSize   int64
	ServedFile    string
	Success       bool
	FailureReason string
	SlotPath      string
}

// RecordPreloadAttempt inserts a preload row for a slot (attempt started, result not yet known).
// It is idempotent: if an unresolved preload row already exists for the same slot_path it is a no-op,
// so Stremio's multiple automatic requests to the same play URL don't create duplicate rows.
// Safe to call with nil receiver (no-op).
func (m *StateManager) RecordPreloadAttempt(p RecordAttemptParams) {
	if m == nil || m.db == nil || p.SlotPath == "" {
		return
	}
	// INSERT only when no active (preload=1) row exists for this slot yet.
	_ = m.withWriteLock(func(db *sql.DB) error {
		_, err := db.Exec(`INSERT INTO nzb_attempts (tried_at, content_type, content_id, content_title, release_title, release_url, release_size, served_file, success, failure_reason, slot_path, preload)
				SELECT ?, ?, ?, ?, ?, ?, ?, ?, 0, '', ?, 1
			WHERE NOT EXISTS (SELECT 1 FROM nzb_attempts WHERE slot_path = ? AND preload = 1)`,
			time.Now().UnixMilli(),
			p.ContentType,
			p.ContentID,
			p.ContentTitle,
			p.ReleaseTitle,
			p.ReleaseURL,
			p.ReleaseSize,
			p.ServedFile,
			p.SlotPath,
			p.SlotPath, // for the NOT EXISTS sub-query
		)
		return err
	})
}

// RecordAttempt writes one NZB attempt row, or updates an existing preload row by slot_path. Safe to call with nil receiver (no-op).
func (m *StateManager) RecordAttempt(p RecordAttemptParams) {
	if m == nil || m.db == nil {
		return
	}
	success := 0
	if p.Success {
		success = 1
	}
	err := m.withWriteLock(func(db *sql.DB) error {
		if p.SlotPath != "" {
			// Only update the currently-pending preload row (preload=1). Historical resolved rows
			// for the same slot_path (previous plays) must not be mutated.
			res, err := db.Exec(`UPDATE nzb_attempts SET preload = 0, success = ?, failure_reason = ?, served_file = ? WHERE slot_path = ? AND preload = 1`,
				success, p.FailureReason, p.ServedFile, p.SlotPath)
			if err == nil {
				affected, _ := res.RowsAffected()
				if affected > 0 {
					return nil
				}
			}
		}

		_, err := db.Exec(`INSERT INTO nzb_attempts (tried_at, content_type, content_id, content_title, release_title, release_url, release_size, served_file, success, failure_reason, slot_path, preload)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
			time.Now().UnixMilli(),
			p.ContentType,
			p.ContentID,
			p.ContentTitle,
			p.ReleaseTitle,
			p.ReleaseURL,
			p.ReleaseSize,
			p.ServedFile,
			success,
			p.FailureReason,
			p.SlotPath,
		)
		return err
	})
	if err != nil {
		// Best-effort; don't fail playback
		return
	}
}

// ListAttemptsOptions filters and paginates NZB attempts.
type ListAttemptsOptions struct {
	ContentType string
	ContentID   string
	Limit       int
	Offset      int
	Since       *time.Time
}

// ListAttempts returns attempts newest first. Limit default 100, max 500.
func (m *StateManager) ListAttempts(opts ListAttemptsOptions) ([]NZBAttempt, error) {
	if m == nil || m.db == nil {
		return nil, nil
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	query := `SELECT id, tried_at, content_type, content_id, content_title, release_title, release_url, release_size, served_file, success, failure_reason, slot_path, COALESCE(preload, 0)
		FROM nzb_attempts WHERE 1=1`
	args := []interface{}{}
	if opts.ContentType != "" {
		query += ` AND content_type = ?`
		args = append(args, opts.ContentType)
	}
	if opts.ContentID != "" {
		query += ` AND content_id = ?`
		args = append(args, opts.ContentID)
	}
	if opts.Since != nil {
		query += ` AND tried_at >= ?`
		args = append(args, opts.Since.UnixMilli())
	}
	query += ` ORDER BY tried_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []NZBAttempt
	for rows.Next() {
		var a NZBAttempt
		var triedAtMs int64
		var success int
		var preload int
		var releaseURL, servedFile, failureReason, slotPath sql.NullString
		var contentTitle sql.NullString
		var releaseSize sql.NullInt64
		err := rows.Scan(&a.ID, &triedAtMs, &a.ContentType, &a.ContentID, &contentTitle, &a.ReleaseTitle, &releaseURL, &releaseSize, &servedFile, &success, &failureReason, &slotPath, &preload)
		if err != nil {
			return nil, err
		}
		a.TriedAt = time.UnixMilli(triedAtMs)
		a.Success = success != 0
		a.Preload = preload != 0
		if releaseURL.Valid {
			a.ReleaseURL = releaseURL.String
		}
		if servedFile.Valid {
			a.ServedFile = servedFile.String
		}
		if failureReason.Valid {
			a.FailureReason = failureReason.String
		}
		if slotPath.Valid {
			a.SlotPath = slotPath.String
		}
		if contentTitle.Valid {
			a.ContentTitle = contentTitle.String
		}
		if releaseSize.Valid {
			a.ReleaseSize = releaseSize.Int64
		}
		list = append(list, a)
	}
	return list, rows.Err()
}
