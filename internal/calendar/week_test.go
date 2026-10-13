package calendar

import (
	"testing"
	"time"
)

var testNow = []struct {
	name string
	now  time.Time
}{
	{"Monday", time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC)},
	{"Sunday", time.Date(2024, 6, 17, 12, 0, 0, 0, time.UTC)},
}

func TestGetCurrentWeekRange(t *testing.T) {
	for _, tt := range testNow {
		t.Run(tt.name, func(t *testing.T) {
			start, end := GetCurrentWeekRange(tt.now)
			expectedStart := time.Date(tt.now.Year(), tt.now.Month(), tt.now.Day()-int(tt.now.Weekday())+1, 0, 0, 0, 0, tt.now.Location())
			expectedEnd := expectedStart.AddDate(0, 0, 7)
			if !start.Equal(expectedStart) || !end.Equal(expectedEnd) {
				t.Errorf("GetCurrentWeekRange(%v) = (%v, %v), want (%v, %v)", tt, start, end, expectedStart, expectedEnd)
			}
		})
	}
}

func TestFindAvailableTimeSlots(t *testing.T) {
	// Wednesday 14 Oct 2026, 11:30.
	now := time.Date(2026, 10, 14, 11, 30, 0, 0, time.UTC)
	start, end := GetCurrentWeekRange(now)
	events := []Event{{Title: "Lunch", StartTime: time.Date(2026, 10, 14, 13, 0, 0, 0, time.UTC)}}

	slots := FindAvailableTimeSlots(events, start, end, now)

	for _, s := range slots {
		if IsWeekend(s) {
			t.Errorf("slot on a weekend: %v", s)
		}
		if s.Before(now) {
			t.Errorf("slot in the past: %v", s)
		}
	}
	// Wed: 12, 14, 15, 16 (13 is lunch). Thu and Fri: 8 each.
	if len(slots) != 4+8+8 {
		t.Errorf("got %d slots, want 20: %v", len(slots), slots)
	}
	if want := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC); len(slots) > 0 && !slots[0].Equal(want) {
		t.Errorf("first slot = %v, want %v", slots[0], want)
	}
}
