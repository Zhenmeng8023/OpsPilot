package tasks

import "testing"

func TestStatusTransitions(t *testing.T) {
	allowed := []struct {
		from Status
		to   Status
	}{
		{StatusPending, StatusQueued},
		{StatusQueued, StatusRunning},
		{StatusRunning, StatusSuccess},
		{StatusRunning, StatusFailed},
		{StatusRunning, StatusTimeout},
		{StatusPending, StatusCanceled},
		{StatusQueued, StatusCanceled},
	}
	for _, item := range allowed {
		if !CanTransition(item.from, item.to) {
			t.Fatalf("expected %s -> %s to be allowed", item.from, item.to)
		}
	}

	disallowed := []struct {
		from Status
		to   Status
	}{
		{StatusSuccess, StatusRunning},
		{StatusQueued, StatusSuccess},
		{StatusFailed, StatusQueued},
		{StatusCanceled, StatusRunning},
		{StatusRunning, StatusCanceled},
	}
	for _, item := range disallowed {
		if CanTransition(item.from, item.to) {
			t.Fatalf("expected %s -> %s to be rejected", item.from, item.to)
		}
	}
}

func TestIsTerminal(t *testing.T) {
	for _, status := range []Status{StatusSuccess, StatusFailed, StatusTimeout, StatusCanceled} {
		if !IsTerminal(status) {
			t.Fatalf("expected %s to be terminal", status)
		}
	}
	for _, status := range []Status{StatusPending, StatusQueued, StatusRunning} {
		if IsTerminal(status) {
			t.Fatalf("expected %s to be non-terminal", status)
		}
	}
}
