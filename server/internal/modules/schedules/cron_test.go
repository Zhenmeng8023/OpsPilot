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

func TestNextCronTimeRangeAndList(t *testing.T) {
	loc := time.UTC
	after := time.Date(2026, 5, 7, 9, 10, 0, 0, loc)
	next, err := nextCronTime("10,20 9-10 * * *", loc, after)
	if err != nil {
		t.Fatalf("nextCronTime returned error: %v", err)
	}
	want := time.Date(2026, 5, 7, 9, 20, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next, want)
	}

	next, err = nextCronTime("10,20 9-10 * * *", loc, want)
	if err != nil {
		t.Fatalf("nextCronTime returned error: %v", err)
	}
	want = time.Date(2026, 5, 7, 10, 10, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next, want)
	}
}

func TestNextCronTimeSundaySeven(t *testing.T) {
	loc := time.UTC
	after := time.Date(2026, 5, 9, 10, 0, 0, 0, loc)
	next, err := nextCronTime("0 9 * * 7", loc, after)
	if err != nil {
		t.Fatalf("nextCronTime returned error: %v", err)
	}
	want := time.Date(2026, 5, 10, 9, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next, want)
	}
}

func TestNextCronTimeUsesTimezone(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	after := time.Date(2026, 5, 7, 1, 58, 30, 0, time.UTC)
	next, err := nextCronTime("0 10 * * *", loc, after)
	if err != nil {
		t.Fatalf("nextCronTime returned error: %v", err)
	}
	want := time.Date(2026, 5, 7, 10, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next, want)
	}
	if next.Location().String() != "Asia/Shanghai" {
		t.Fatalf("next location = %s, want Asia/Shanghai", next.Location())
	}
}

func TestNextCronTimeRejectsInvalidFields(t *testing.T) {
	cases := []string{
		"*/0 * * * *",
		"70 * * * *",
		"10-5 * * * *",
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) {
			if _, err := nextCronTime(expr, time.UTC, time.Now()); err == nil {
				t.Fatal("expected invalid expression error")
			}
		})
	}
}
