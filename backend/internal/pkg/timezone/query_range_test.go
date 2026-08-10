package timezone

import (
	"testing"
	"time"
)

func TestParseQueryTimeRange_PreciseRFC3339(t *testing.T) {
	start := "2026-08-09T10:39:31Z"
	end := "2026-08-10T10:39:31Z"
	got, err := ParseQueryTimeRange(start, end, "2026-01-01", "2026-01-02", "UTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Precise {
		t.Fatal("expected precise range")
	}
	if got.Start == nil || got.End == nil {
		t.Fatal("expected both bounds")
	}
	wantStart, _ := time.Parse(time.RFC3339, start)
	wantEnd, _ := time.Parse(time.RFC3339, end)
	if !got.Start.Equal(wantStart) || !got.End.Equal(wantEnd) {
		t.Fatalf("bounds mismatch: got [%v, %v) want [%v, %v)", *got.Start, *got.End, wantStart, wantEnd)
	}
	// Date params must be ignored when times are present.
	if got.Start.Day() == 1 {
		t.Fatal("date params should not apply when times are present")
	}
}

func TestParseQueryTimeRange_PreciseRFC3339Nano(t *testing.T) {
	start := "2026-08-09T10:39:31.123456789Z"
	end := "2026-08-10T10:39:31.987654321Z"
	got, err := ParseQueryTimeRange(start, end, "", "", "UTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Precise {
		t.Fatal("expected precise range")
	}
	if got.Start == nil || got.End == nil {
		t.Fatal("expected both bounds")
	}
}

func TestParseQueryTimeRange_DateOnlyHalfOpen(t *testing.T) {
	if err := Init("UTC"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	got, err := ParseQueryTimeRange("", "", "2026-08-09", "2026-08-10", "UTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Precise {
		t.Fatal("expected non-precise date range")
	}
	wantStart := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	if !got.Start.Equal(wantStart) || !got.End.Equal(wantEnd) {
		t.Fatalf("bounds mismatch: got [%v, %v) want [%v, %v)", *got.Start, *got.End, wantStart, wantEnd)
	}
}

func TestParseQueryTimeRange_TimePrecedenceOverDate(t *testing.T) {
	got, err := ParseQueryTimeRange("2026-08-09T12:00:00Z", "", "2026-01-01", "2026-01-02", "UTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Precise {
		t.Fatal("expected precise")
	}
	if got.Start == nil || got.Start.Month() != time.August {
		t.Fatalf("start should come from start_time, got %v", got.Start)
	}
	if got.End != nil {
		t.Fatalf("end should be nil when only start_time provided, got %v", got.End)
	}
}

func TestParseQueryTimeRange_InvalidTime(t *testing.T) {
	_, err := ParseQueryTimeRange("not-a-time", "2026-08-10T10:00:00Z", "", "", "UTC")
	if err == nil {
		t.Fatal("expected error for invalid start_time")
	}
	_, err = ParseQueryTimeRange("2026-08-09T10:00:00Z", "bad", "", "", "UTC")
	if err == nil {
		t.Fatal("expected error for invalid end_time")
	}
}

func TestParseQueryTimeRange_StartAfterEnd(t *testing.T) {
	_, err := ParseQueryTimeRange("2026-08-10T12:00:00Z", "2026-08-09T12:00:00Z", "", "", "UTC")
	if err == nil {
		t.Fatal("expected error when start_time > end_time")
	}
}

func TestParseQueryTimeRange_InvalidDate(t *testing.T) {
	_, err := ParseQueryTimeRange("", "", "bad-date", "2026-08-10", "UTC")
	if err == nil {
		t.Fatal("expected error for invalid start_date")
	}
}

func TestParseQueryTimeRange_Empty(t *testing.T) {
	got, err := ParseQueryTimeRange("", "", "", "", "UTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Start != nil || got.End != nil || got.Precise {
		t.Fatalf("expected empty range, got %+v", got)
	}
}

func TestFormatRangeEcho_DateRange(t *testing.T) {
	if err := Init("UTC"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	start := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	sd, ed, st, et := FormatRangeEcho(start, end, "UTC", false)
	if sd != "2026-08-09" || ed != "2026-08-10" {
		t.Fatalf("date echo mismatch: %s %s", sd, ed)
	}
	if st == "" || et == "" {
		t.Fatal("expected RFC3339 time echoes")
	}
}

func TestFormatRangeEcho_Precise(t *testing.T) {
	if err := Init("UTC"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	start, _ := time.Parse(time.RFC3339, "2026-08-09T10:39:31Z")
	end, _ := time.Parse(time.RFC3339, "2026-08-10T10:39:31Z")
	sd, ed, st, et := FormatRangeEcho(start, end, "UTC", true)
	if sd != "2026-08-09" || ed != "2026-08-10" {
		t.Fatalf("date echo mismatch: %s %s", sd, ed)
	}
	if st != "2026-08-09T10:39:31Z" || et != "2026-08-10T10:39:31Z" {
		t.Fatalf("time echo mismatch: %s %s", st, et)
	}
}
