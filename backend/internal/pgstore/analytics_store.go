package pgstore

import (
	"context"
	"time"
)

// AnalyticsOverview is the org-wide snapshot behind the Command Center's
// stat row (superuser-only, see AdminHandler.HandleGetAnalyticsOverview).
type AnalyticsOverview struct {
	TotalUsers          int            `json:"totalUsers"`
	UsersByRole         map[string]int `json:"usersByRole"` // doctor, superuser, admin, user (demo)
	TotalSessions       int            `json:"totalSessions"`
	SessionsByType      map[string]int `json:"sessionsByType"`
	SessionsLast24h     int            `json:"sessionsLast24h"`
	SessionsLast7d      int            `json:"sessionsLast7d"`
	// SessionsPrev7d is the 7 days immediately before SessionsLast7d's
	// window (i.e. 8-14 days ago) — both rolling from now(), so the
	// week-over-week trend computed from them is directly comparable to
	// SessionsLast7d itself. Deliberately NOT derived from the calendar-day
	// buckets in GetUsageTimeseries, which use a different window
	// definition (UTC day boundaries) and could disagree by up to a day's
	// worth of sessions right at the boundary.
	SessionsPrev7d      int            `json:"sessionsPrev7d"`
	SessionsLast30d     int            `json:"sessionsLast30d"`
	PendingApprovals    int            `json:"pendingApprovals"`
	DemoAccountsCount   int            `json:"demoAccountsCount"`
	DemoAccountsAtLimit int            `json:"demoAccountsAtLimit"`
}

// DayUsage is one point on the Command Center's 30-day usage chart.
type DayUsage struct {
	Date      string `json:"date"` // YYYY-MM-DD
	Ambient   int    `json:"ambient"`
	File      int    `json:"file"`
	Dictation int    `json:"dictation"`
}

// DemoFunnelUser is one row of the "trial account funnel" table — a demo
// ("user" role) account plus its lifetime per-feature trial usage.
type DemoFunnelUser struct {
	UserID         string         `json:"userId"`
	Username       string         `json:"username"`
	RequestedRole  string         `json:"requestedRole"`
	ApprovalStatus string         `json:"approvalStatus"`
	CreatedAt      string         `json:"createdAt"`
	FeatureUsage   map[string]int `json:"featureUsage"` // ambient, file_transcription, dictation, ai_summary, document_generation, search
}

// RecentSessionEntry is one row of the Command Center's cross-user activity
// feed.
type RecentSessionEntry struct {
	ID         string `json:"id"`
	UserID     string `json:"userId"`
	UserName   string `json:"userName"`
	Username   string `json:"username"`
	IsDemo     bool   `json:"isDemo"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	Transcript string `json:"transcript"` // truncated snippet
	Document   bool   `json:"document"`
	CreatedAt  string `json:"createdAt"`
}

// GetAnalyticsOverview computes the org-wide counts behind the Command
// Center's stat row in three round trips (user/role counts, session counts,
// session-by-type counts) rather than pulling every row into Go.
func (s *Store) GetAnalyticsOverview(ctx context.Context) (*AnalyticsOverview, error) {
	o := &AnalyticsOverview{UsersByRole: map[string]int{}, SessionsByType: map[string]int{}}

	// Covers all four DefaultRoles (auth.DefaultRoles: superuser, doctor,
	// admin, user/demo) — a user with a custom (non-system) role is only
	// reflected in TotalUsers, not in this breakdown, same as any role
	// created later via Role Management.
	var doctors, superusers, admins, demoUsers int
	if err := s.pool.QueryRow(ctx, `
		SELECT
			count(*),
			count(*) FILTER (WHERE 'doctor' = ANY(roles)),
			count(*) FILTER (WHERE 'superuser' = ANY(roles)),
			count(*) FILTER (WHERE 'admin' = ANY(roles)),
			count(*) FILTER (WHERE 'user' = ANY(roles)),
			count(*) FILTER (WHERE status = 'pending')
		FROM users`).Scan(&o.TotalUsers, &doctors, &superusers, &admins, &demoUsers, &o.PendingApprovals); err != nil {
		return nil, err
	}
	o.UsersByRole["doctor"] = doctors
	o.UsersByRole["superuser"] = superusers
	o.UsersByRole["admin"] = admins
	o.UsersByRole["user"] = demoUsers
	o.DemoAccountsCount = demoUsers

	if err := s.pool.QueryRow(ctx, `
		SELECT
			count(*),
			count(*) FILTER (WHERE created_at >= now() - interval '1 day'),
			count(*) FILTER (WHERE created_at >= now() - interval '7 days'),
			count(*) FILTER (WHERE created_at >= now() - interval '14 days' AND created_at < now() - interval '7 days'),
			count(*) FILTER (WHERE created_at >= now() - interval '30 days')
		FROM sessions`).Scan(&o.TotalSessions, &o.SessionsLast24h, &o.SessionsLast7d, &o.SessionsPrev7d, &o.SessionsLast30d); err != nil {
		return nil, err
	}

	typeRows, err := s.pool.Query(ctx, `SELECT type, count(*) FROM sessions GROUP BY type`)
	if err != nil {
		return nil, err
	}
	defer typeRows.Close()
	for typeRows.Next() {
		var typ string
		var cnt int
		if err := typeRows.Scan(&typ, &cnt); err != nil {
			return nil, err
		}
		o.SessionsByType[typ] = cnt
	}

	if err := s.pool.QueryRow(ctx,
		`SELECT count(DISTINCT user_id) FROM demo_usage WHERE use_count >= 3`,
	).Scan(&o.DemoAccountsAtLimit); err != nil {
		return nil, err
	}

	return o, nil
}

// GetUsageTimeseries returns one point per day for the last `days` days
// (oldest first, zero-filled for days with no sessions) — a single
// DATE_TRUNC/GROUP BY query, not a per-day loop.
func (s *Store) GetUsageTimeseries(ctx context.Context, days int) ([]DayUsage, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT date_trunc('day', created_at)::date AS day, type, count(*)
		FROM sessions
		WHERE created_at >= now() - make_interval(days => $1::int)
		GROUP BY day, type`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byDay := map[string]map[string]int{}
	for rows.Next() {
		var day time.Time
		var typ string
		var cnt int
		if err := rows.Scan(&day, &typ, &cnt); err != nil {
			return nil, err
		}
		key := day.Format("2006-01-02")
		if byDay[key] == nil {
			byDay[key] = map[string]int{}
		}
		byDay[key][typ] = cnt
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	out := make([]DayUsage, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := today.AddDate(0, 0, -i)
		key := d.Format("2006-01-02")
		m := byDay[key]
		out = append(out, DayUsage{
			Date:      key,
			Ambient:   m["ambient"],
			File:      m["file-transcription"],
			Dictation: m["dictation"],
		})
	}
	return out, nil
}

// GetDemoFunnel returns every demo ("user" role) account with its lifetime
// per-feature trial usage, newest signup first — the Command Center sorts
// this client-side by closest-to-limit.
func (s *Store) GetDemoFunnel(ctx context.Context) ([]*DemoFunnelUser, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, username, requested_role, status, created_at
		FROM users
		WHERE 'user' = ANY(roles)
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*DemoFunnelUser
	for rows.Next() {
		u := &DemoFunnelUser{FeatureUsage: map[string]int{}}
		var createdAt time.Time
		if err := rows.Scan(&u.UserID, &u.Username, &u.RequestedRole, &u.ApprovalStatus, &createdAt); err != nil {
			return nil, err
		}
		u.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	usageRows, err := s.pool.Query(ctx, `SELECT user_id, feature, use_count FROM demo_usage`)
	if err != nil {
		return nil, err
	}
	defer usageRows.Close()
	byUser := map[string]map[string]int{}
	for usageRows.Next() {
		var uid, feature string
		var cnt int
		if err := usageRows.Scan(&uid, &feature, &cnt); err != nil {
			return nil, err
		}
		if byUser[uid] == nil {
			byUser[uid] = map[string]int{}
		}
		byUser[uid][feature] = cnt
	}
	for _, u := range out {
		if m, ok := byUser[u.UserID]; ok {
			u.FeatureUsage = m
		}
	}
	return out, nil
}

// GetRecentSessions returns the most recent sessions across every user
// (newest first), joined with the owning user's display name — the Command
// Center's cross-user activity feed.
func (s *Store) GetRecentSessions(ctx context.Context, limit int) ([]*RecentSessionEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.id, s.user_id, u.name, u.username, ('user' = ANY(u.roles)),
		       s.type, s.title, left(s.transcript, 160), (s.document IS NOT NULL), s.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		ORDER BY s.created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*RecentSessionEntry{}
	for rows.Next() {
		e := &RecentSessionEntry{}
		var createdAt time.Time
		if err := rows.Scan(&e.ID, &e.UserID, &e.UserName, &e.Username, &e.IsDemo,
			&e.Type, &e.Title, &e.Transcript, &e.Document, &createdAt); err != nil {
			return nil, err
		}
		e.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		out = append(out, e)
	}
	return out, rows.Err()
}
