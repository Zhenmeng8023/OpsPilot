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
