package metrics

import "testing"

func TestNormalizeTrendHours(t *testing.T) {
	if got := normalizeTrendHours(0); got != 24 {
		t.Fatalf("normalizeTrendHours(0) = %d", got)
	}
	if got := normalizeTrendHours(24 * 30); got != 24*14 {
		t.Fatalf("normalizeTrendHours(too large) = %d", got)
	}
}

func TestNormalizeTrendPointLimit(t *testing.T) {
	if got := normalizeTrendPointLimit(0); got != 120 {
		t.Fatalf("normalizeTrendPointLimit(0) = %d", got)
	}
	if got := normalizeTrendPointLimit(3); got != 10 {
		t.Fatalf("normalizeTrendPointLimit(3) = %d", got)
	}
	if got := normalizeTrendPointLimit(500); got != 240 {
		t.Fatalf("normalizeTrendPointLimit(500) = %d", got)
	}
}

func TestNormalizeTrendGranularity(t *testing.T) {
	if got := normalizeTrendGranularity("auto", 12); got != "raw" {
		t.Fatalf("expected raw for short auto range, got %s", got)
	}
	if got := normalizeTrendGranularity("auto", 48); got != "5m" {
		t.Fatalf("expected 5m for medium auto range, got %s", got)
	}
	if got := normalizeTrendGranularity("auto", 96); got != "1h" {
		t.Fatalf("expected 1h for long auto range, got %s", got)
	}
	if got := normalizeTrendGranularity("unknown", 96); got != "raw" {
		t.Fatalf("expected raw for unsupported granularity, got %s", got)
	}
}

func TestNormalizeDashboardInput(t *testing.T) {
	normalized, appErr := normalizeDashboardInput(DashboardInput{
		Name:        " Capacity ",
		MetricCode:  "agent.os.disk.used_percent",
		RangeHours:  0,
		PointLimit:  500,
		Granularity: "1h",
	}, true)
	if appErr != nil {
		t.Fatalf("unexpected error: %v", appErr)
	}
	if normalized.Name != "Capacity" || normalized.RangeHours != 24 || normalized.PointLimit != 240 || normalized.Granularity != "1h" || normalized.Status != "active" {
		t.Fatalf("unexpected normalized input: %#v", normalized)
	}
}
