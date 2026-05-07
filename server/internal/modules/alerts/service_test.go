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
