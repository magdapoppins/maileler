package gcloud

import (
	"context"
	"fmt"
	"time"

	calendarutils "github.com/magdapoppins/mailerer/internal/calendar"
	"google.golang.org/api/calendar/v3"
)

// ListEvents returns the events in the user's primary calendar between start and end.
func (c *Client) ListEvents(ctx context.Context, start, end time.Time) ([]calendarutils.Event, error) {
	events, err := c.calendar.Events.List("primary").
		TimeMin(start.Format(time.RFC3339)).
		TimeMax(end.Format(time.RFC3339)).
		SingleEvents(true). // expand recurring events into individual occurrences
		OrderBy("startTime").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return extractEvents(events), nil
}

// extractEvents converts the Calendar API response into the app's own Event type.
func extractEvents(events *calendar.Events) []calendarutils.Event {
	simplifiedEvents := []calendarutils.Event{}
	for _, e := range events.Items {
		if e.Start.DateTime != "" {
			t, _ := time.Parse(time.RFC3339, e.Start.DateTime)
			simplifiedEvents = append(simplifiedEvents, calendarutils.Event{Title: e.Summary, StartTime: t, AllDay: false})
		} else {
			// All-day events only have a Date, no DateTime.
			t, _ := time.Parse("2006-01-02", e.Start.Date)
			simplifiedEvents = append(simplifiedEvents, calendarutils.Event{Title: e.Summary, StartTime: t, AllDay: true})
		}
	}
	return simplifiedEvents
}
