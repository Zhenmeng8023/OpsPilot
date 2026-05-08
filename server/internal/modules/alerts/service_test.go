package alerts

import (
	"database/sql"
	"testing"
	"time"
)

func TestRuleCooldownSeconds(t *testing.T) {
	got := ruleCooldownSeconds(sql.NullString{String: `{"cooldownSeconds":300}`, Valid: true})
	if got != 300 {
		t.Fatalf("ruleCooldownSeconds() = %d", got)
	}
}

func TestParseAlertMetadata(t *testing.T) {
	meta := parseAlertMetadata(sql.NullString{String: `{"acknowledgedAt":"2026-05-07 10:00:00","acknowledgedBy":"admin","silencedUntil":"2026-05-07 11:00:00","cooldownUntil":"2026-05-07 12:00:00"}`, Valid: true})
	if meta.AcknowledgedBy != "admin" || meta.SilencedUntil == "" || meta.CooldownUntil == "" {
		t.Fatalf("unexpected metadata: %#v", meta)
	}
}

func TestSilenceExpired(t *testing.T) {
	past := time.Now().Add(-time.Minute).Format("2006-01-02 15:04:05")
	future := time.Now().Add(time.Minute).Format("2006-01-02 15:04:05")
	if !silenceExpired(alertMetadata{SilencedUntil: past}) {
		t.Fatal("expected past silence to expire")
	}
	if silenceExpired(alertMetadata{SilencedUntil: future}) {
		t.Fatal("expected future silence to remain active")
	}
}

func TestSustainedViolationRequiresContinuousWindow(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	points := []metricSeriesPoint{
		{Value: 91, CollectedAt: now.Format("2006-01-02 15:04:05")},
		{Value: 90, CollectedAt: now.Add(-40 * time.Second).Format("2006-01-02 15:04:05")},
		{Value: 89, CollectedAt: now.Add(-80 * time.Second).Format("2006-01-02 15:04:05")},
	}
	ok, start := sustainedViolation(points, ">=", 85, time.Minute)
	if !ok {
		t.Fatal("expected sustained violation to pass")
	}
	if start.Format("2006-01-02 15:04:05") != now.Add(-80*time.Second).Format("2006-01-02 15:04:05") {
		t.Fatalf("unexpected streak start: %v", start)
	}
}

func TestSustainedViolationResetsOnRecoveredSample(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	points := []metricSeriesPoint{
		{Value: 91, CollectedAt: now.Format("2006-01-02 15:04:05")},
		{Value: 90, CollectedAt: now.Add(-20 * time.Second).Format("2006-01-02 15:04:05")},
		{Value: 70, CollectedAt: now.Add(-40 * time.Second).Format("2006-01-02 15:04:05")},
		{Value: 92, CollectedAt: now.Add(-80 * time.Second).Format("2006-01-02 15:04:05")},
	}
	ok, _ := sustainedViolation(points, ">=", 85, 30*time.Second)
	if ok {
		t.Fatal("expected recovered sample to reset the streak")
	}
}

func TestMaxMetricSeriesPoints(t *testing.T) {
	if got := maxMetricSeriesPoints(0); got != 24 {
		t.Fatalf("unexpected points for zero duration: %d", got)
	}
	if got := maxMetricSeriesPoints(3600); got <= 24 {
		t.Fatalf("expected larger window for long duration, got %d", got)
	}
	if got := maxMetricSeriesPoints(24 * 3600); got != 480 {
		t.Fatalf("expected upper clamp, got %d", got)
	}
}

func TestNormalizeHistoryHours(t *testing.T) {
	if got := normalizeHistoryHours(0); got != 24 {
		t.Fatalf("expected default hours, got %d", got)
	}
	if got := normalizeHistoryHours(24 * 30); got != 24*14 {
		t.Fatalf("expected clamped hours, got %d", got)
	}
}

func TestNormalizeHistoryBucketMinutes(t *testing.T) {
	if got := normalizeHistoryBucketMinutes(0); got != 60 {
		t.Fatalf("expected default bucket, got %d", got)
	}
	if got := normalizeHistoryBucketMinutes(1); got != 5 {
		t.Fatalf("expected minimum bucket clamp, got %d", got)
	}
	if got := normalizeHistoryBucketMinutes(600); got != 240 {
		t.Fatalf("expected maximum bucket clamp, got %d", got)
	}
}

func TestAggregateAlertGroups(t *testing.T) {
	rows := []alertGroupRecord{
		{
			AlertUID:      "al-2",
			RuleUID:       sql.NullString{String: "rule-1", Valid: true},
			RuleName:      sql.NullString{String: "CPU Hot", Valid: true},
			HostUID:       sql.NullString{String: "host-2", Valid: true},
			HostName:      sql.NullString{String: "db-02", Valid: true},
			HostGroupUID:  sql.NullString{String: "group-1", Valid: true},
			HostGroupName: sql.NullString{String: "Database", Valid: true},
			Title:         "CPU above 90%",
			Severity:      "critical",
			Status:        "acknowledged",
			Fingerprint:   "fp-1",
			FirstSeenAt:   "2026-05-08 10:05:00",
			LastSeenAt:    "2026-05-08 10:10:00",
		},
		{
			AlertUID:      "al-1",
			RuleUID:       sql.NullString{String: "rule-1", Valid: true},
			RuleName:      sql.NullString{String: "CPU Hot", Valid: true},
			HostUID:       sql.NullString{String: "host-1", Valid: true},
			HostName:      sql.NullString{String: "db-01", Valid: true},
			HostGroupUID:  sql.NullString{String: "group-1", Valid: true},
			HostGroupName: sql.NullString{String: "Database", Valid: true},
			Title:         "CPU above 90%",
			Severity:      "critical",
			Status:        "firing",
			Fingerprint:   "fp-1",
			FirstSeenAt:   "2026-05-08 10:00:00",
			LastSeenAt:    "2026-05-08 10:12:00",
		},
		{
			AlertUID:    "al-3",
			RuleUID:     sql.NullString{String: "rule-1", Valid: true},
			RuleName:    sql.NullString{String: "CPU Hot", Valid: true},
			HostUID:     sql.NullString{String: "host-3", Valid: true},
			HostName:    sql.NullString{String: "edge-01", Valid: true},
			Title:       "CPU above 90%",
			Severity:    "critical",
			Status:      "resolved",
			Fingerprint: "fp-1",
			FirstSeenAt: "2026-05-08 09:00:00",
			LastSeenAt:  "2026-05-08 09:10:00",
			ResolvedAt:  sql.NullString{String: "2026-05-08 09:12:00", Valid: true},
		},
	}

	groups := aggregateAlertGroups(rows)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].HostGroupID != "group-1" || groups[0].AlertCount != 2 || groups[0].HostCount != 2 {
		t.Fatalf("unexpected grouped counts: %#v", groups[0])
	}
	if groups[0].Status != "firing" || groups[0].ActiveCount != 2 || groups[0].FiringCount != 1 || groups[0].AcknowledgedCount != 1 {
		t.Fatalf("unexpected grouped status: %#v", groups[0])
	}
	if len(groups[0].Hosts) != 2 || groups[0].FirstSeenAt != "2026-05-08 10:00:00" || groups[0].LastSeenAt != "2026-05-08 10:12:00" {
		t.Fatalf("unexpected grouped host/time summary: %#v", groups[0])
	}
	if groups[1].HostGroupID != "" || groups[1].Status != "resolved" || groups[1].ResolvedAt == "" {
		t.Fatalf("unexpected ungrouped summary: %#v", groups[1])
	}
}
