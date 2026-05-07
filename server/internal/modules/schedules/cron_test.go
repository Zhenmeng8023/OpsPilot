package schedules

import (
	"testing"
	"time"
)

func TestNextCronTimeEveryFiveMinutes(t *testing.T) {
	loc := time.FixedZone("test", 8*60*60)
	after := time.Date(2026, 5, 7, 10, 2, 10, 0, loc)
	next, err := nextCronTime("*/5 * * * *", loc, after)
	if err != nil {
		t.Fatalf("nextCronTime returned error: %v", err)
	}
	want := time.Date(2026, 5, 7, 10, 5, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next, want)
	}
}

func TestNextCronTimeSpecificWeekday(t *testing.T) {
	loc := time.UTC
	after := time.Date(2026, 5, 7, 10, 0, 0, 0, loc)
	next, err := nextCronTime("30 9 * * 1", loc, after)
	if err != nil {
		t.Fatalf("nextCronTime returned error: %v", err)
	}
	want := time.Date(2026, 5, 11, 9, 30, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next, want)
	}
}

func TestNextCronTimeRejectsInvalidExpression(t *testing.T) {
	if _, err := nextCronTime("* * *", time.UTC, time.Now()); err == nil {
		t.Fatal("expected invalid expression error")
	}
}
