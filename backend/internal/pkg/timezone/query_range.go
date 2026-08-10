package timezone

import (
	"fmt"
	"strings"
	"time"
)

// QueryTimeRange is a half-open [Start, End) interval derived from query params.
type QueryTimeRange struct {
	Start *time.Time
	End   *time.Time
	// Precise is true when start_time/end_time were used (second-level bounds).
	Precise bool
}

// ParseRFC3339 parses RFC3339Nano first, then RFC3339.
func ParseRFC3339(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, value)
}

// ParseQueryTimeRange parses optional start_time/end_time (RFC3339) with
// precedence over start_date/end_date (YYYY-MM-DD).
//
// Semantics:
//   - precise path: bounds are used as-is (half-open [start, end) at the repository layer)
//   - date path: start is day 00:00; end is exclusive next calendar day 00:00 (DST-safe AddDate)
func ParseQueryTimeRange(startTimeStr, endTimeStr, startDateStr, endDateStr, userTZ string) (QueryTimeRange, error) {
	startTimeStr = strings.TrimSpace(startTimeStr)
	endTimeStr = strings.TrimSpace(endTimeStr)
	startDateStr = strings.TrimSpace(startDateStr)
	endDateStr = strings.TrimSpace(endDateStr)

	if startTimeStr != "" || endTimeStr != "" {
		var startPtr, endPtr *time.Time
		if startTimeStr != "" {
			t, err := ParseRFC3339(startTimeStr)
			if err != nil {
				return QueryTimeRange{}, fmt.Errorf("Invalid start_time, expect RFC3339")
			}
			startPtr = &t
		}
		if endTimeStr != "" {
			t, err := ParseRFC3339(endTimeStr)
			if err != nil {
				return QueryTimeRange{}, fmt.Errorf("Invalid end_time, expect RFC3339")
			}
			endPtr = &t
		}
		if startPtr != nil && endPtr != nil && startPtr.After(*endPtr) {
			return QueryTimeRange{}, fmt.Errorf("start_time must be <= end_time")
		}
		return QueryTimeRange{Start: startPtr, End: endPtr, Precise: true}, nil
	}

	var startPtr, endPtr *time.Time
	if startDateStr != "" {
		t, err := ParseInUserLocation("2006-01-02", startDateStr, userTZ)
		if err != nil {
			return QueryTimeRange{}, fmt.Errorf("Invalid start_date format, use YYYY-MM-DD")
		}
		startPtr = &t
	}
	if endDateStr != "" {
		t, err := ParseInUserLocation("2006-01-02", endDateStr, userTZ)
		if err != nil {
			return QueryTimeRange{}, fmt.Errorf("Invalid end_date format, use YYYY-MM-DD")
		}
		// Half-open range [start, end): move to next calendar day start (DST-safe).
		t = t.AddDate(0, 0, 1)
		endPtr = &t
	}
	if startPtr != nil && endPtr != nil && startPtr.After(*endPtr) {
		return QueryTimeRange{}, fmt.Errorf("start_date must be <= end_date")
	}
	return QueryTimeRange{Start: startPtr, End: endPtr, Precise: false}, nil
}

// FormatRangeEcho returns start_date/end_date/start_time/end_time strings for API responses.
// end_date is the inclusive calendar day of the half-open end bound when non-precise,
// or the calendar day of End when precise.
func FormatRangeEcho(start, end time.Time, userTZ string, precise bool) (startDate, endDate, startTime, endTime string) {
	loc := Location()
	if userTZ != "" {
		if userLoc, err := time.LoadLocation(userTZ); err == nil {
			loc = userLoc
		}
	}
	if !start.IsZero() {
		startIn := start.In(loc)
		startDate = startIn.Format("2006-01-02")
		startTime = start.UTC().Format(time.RFC3339)
	}
	if !end.IsZero() {
		endIn := end.In(loc)
		if precise {
			endDate = endIn.Format("2006-01-02")
		} else {
			// end is exclusive next-day 00:00 for date ranges
			endDate = endIn.AddDate(0, 0, -1).Format("2006-01-02")
		}
		endTime = end.UTC().Format(time.RFC3339)
	}
	return startDate, endDate, startTime, endTime
}
