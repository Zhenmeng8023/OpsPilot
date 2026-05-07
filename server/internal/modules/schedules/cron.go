package schedules

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

type cronSpec struct {
	minutes    map[int]bool
	hours      map[int]bool
	monthDays  map[int]bool
	months     map[int]bool
	weekdays   map[int]bool
	anyDay     bool
	anyWeekday bool
}

func nextCronTime(expr string, loc *time.Location, after time.Time) (time.Time, error) {
	spec, err := parseCron(expr)
	if err != nil {
		return time.Time{}, err
	}
	cursor := after.In(loc).Truncate(time.Minute).Add(time.Minute)
	deadline := cursor.AddDate(1, 0, 0)
	for cursor.Before(deadline) {
		if spec.matches(cursor) {
			return cursor, nil
		}
		cursor = cursor.Add(time.Minute)
	}
	return time.Time{}, errors.New("cron expression has no fire time within one year")
}

func parseCron(expr string) (cronSpec, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return cronSpec{}, errors.New("cron expression must contain 5 fields")
	}
	minutes, _, err := parseCronField(fields[0], 0, 59, false)
	if err != nil {
		return cronSpec{}, err
	}
	hours, _, err := parseCronField(fields[1], 0, 23, false)
	if err != nil {
		return cronSpec{}, err
	}
	monthDays, anyDay, err := parseCronField(fields[2], 1, 31, false)
	if err != nil {
		return cronSpec{}, err
	}
	months, _, err := parseCronField(fields[3], 1, 12, false)
	if err != nil {
		return cronSpec{}, err
	}
	weekdays, anyWeekday, err := parseCronField(fields[4], 0, 7, true)
	if err != nil {
		return cronSpec{}, err
	}
	return cronSpec{
		minutes:    minutes,
		hours:      hours,
		monthDays:  monthDays,
		months:     months,
		weekdays:   weekdays,
		anyDay:     anyDay,
		anyWeekday: anyWeekday,
	}, nil
}

func parseCronField(field string, minValue, maxValue int, normalizeSunday bool) (map[int]bool, bool, error) {
	values := map[int]bool{}
	any := false
	parts := strings.Split(field, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false, errors.New("empty cron field part")
		}
		base, step := part, 1
		if before, after, ok := strings.Cut(part, "/"); ok {
			base = before
			parsedStep, err := strconv.Atoi(after)
			if err != nil || parsedStep <= 0 {
				return nil, false, errors.New("invalid cron step")
			}
			step = parsedStep
		}
		start, end := minValue, maxValue
		if base == "*" {
			any = true
		} else if before, after, ok := strings.Cut(base, "-"); ok {
			var err error
			start, err = strconv.Atoi(before)
			if err != nil {
				return nil, false, errors.New("invalid cron range start")
			}
			end, err = strconv.Atoi(after)
			if err != nil {
				return nil, false, errors.New("invalid cron range end")
			}
		} else {
			parsed, err := strconv.Atoi(base)
			if err != nil {
				return nil, false, errors.New("invalid cron value")
			}
			start, end = parsed, parsed
		}
		if start < minValue || end > maxValue || start > end {
			return nil, false, errors.New("cron value out of range")
		}
		for value := start; value <= end; value += step {
			if normalizeSunday && value == 7 {
				values[0] = true
				continue
			}
			values[value] = true
		}
	}
	return values, any, nil
}

func (spec cronSpec) matches(value time.Time) bool {
	weekday := int(value.Weekday())
	return spec.minutes[value.Minute()] &&
		spec.hours[value.Hour()] &&
		spec.months[int(value.Month())] &&
		(spec.anyDay || spec.monthDays[value.Day()]) &&
		(spec.anyWeekday || spec.weekdays[weekday])
}
