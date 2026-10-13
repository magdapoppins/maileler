package calendar

import (
	"time"
)

type Event struct {
	Title     string
	StartTime time.Time
	AllDay    bool
}

func GetCurrentWeekRange(now time.Time) (time.Time, time.Time) {
	daysSinceMonday := (int(now.Weekday()) + 6) % 7 // Go's week starts on Sunday=0
	start := time.Date(now.Year(), now.Month(), now.Day()-daysSinceMonday, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, 7)
	return start, end
}

// Working hours that meeting slots are offered in: 9 AM to 5 PM.
const (
	WorkdayStart = 9
	WorkdayEnd   = 17
)

// IsWeekend reports whether t falls on a Saturday or Sunday.
func IsWeekend(t time.Time) bool {
	return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday
}

// FindAvailableTimeSlots returns the free one-hour slots in the week starting at start.
// Weekends and slots that start before now are never offered.
func FindAvailableTimeSlots(events []Event, start, end, now time.Time) []time.Time {
	availableSlots := []time.Time{}
	// For each day in the week, check for available slots.
	for d := 0; d < 7; d++ {
		dayStart := start.AddDate(0, 0, d)
		dayEnd := dayStart.AddDate(0, 0, 1)
		if IsWeekend(dayStart) {
			continue
		}

		// Check each hour in the day for availability.
		for hour := WorkdayStart; hour < WorkdayEnd; hour++ {
			slotStart := time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(), hour, 0, 0, 0, dayStart.Location())
			slotEnd := slotStart.Add(time.Duration(60) * time.Minute)

			if slotEnd.After(dayEnd) {
				break // No more slots available for this day
			}
			if slotStart.Before(now) {
				continue // Already started or passed
			}

			conflict := false
			for _, e := range events {
				if (e.StartTime.Before(slotEnd) && e.StartTime.Add(time.Duration(60)*time.Minute).After(slotStart)) || e.AllDay {
					conflict = true
					break
				}
			}

			if !conflict {
				availableSlots = append(availableSlots, slotStart)
			}
		}
	}
	return availableSlots
}
