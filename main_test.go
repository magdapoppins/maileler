package main

import (
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	calendarutils "github.com/magdapoppins/mailerer/internal/calendar"
)

func press(t *testing.T, m tea.Model, keys ...string) tea.Model {
	t.Helper()
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "esc":
			msg = tea.KeyPressMsg{Code: tea.KeyEscape}
		case "space":
			msg = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
		case "left":
			msg = tea.KeyPressMsg{Code: tea.KeyLeft}
		case "right":
			msg = tea.KeyPressMsg{Code: tea.KeyRight}
		case "down":
			msg = tea.KeyPressMsg{Code: tea.KeyDown}
		default:
			msg = tea.KeyPressMsg{Code: []rune(k)[0], Text: k}
		}
		m, _ = m.Update(msg)
	}
	return m
}

func TestFlow(t *testing.T) {
	now := time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC)
	var m tea.Model = newModel(nil, now)

	m, _ = m.Update(eventsMsg{events: []calendarutils.Event{
		{Title: "Standup", StartTime: time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)},
	}})
	if got := m.(model).state; got != stateEvents {
		t.Fatalf("after load: state = %v, want stateEvents", got)
	}

	m = press(t, m, "y")
	if got := m.(model); got.state != stateSlots || len(got.slots) == 0 {
		t.Fatalf("after y: state = %v, %d slots", got.state, len(got.slots))
	}
	if got := len(m.(model).days); got != 5 {
		t.Fatalf("grid has %d day columns, want 5 (Mon–Fri)", got)
	}

	// Enter with nothing selected stays on the picker.
	m = press(t, m, "enter")
	if got := m.(model); got.state != stateSlots || got.pickerHint == "" {
		t.Fatalf("enter with no selection: state = %v, hint = %q", got.state, got.pickerHint)
	}

	// It's Tue 13 Oct 08:00, so Monday has passed and Tue 09:00 is busy (Standup):
	// the cursor starts on Tue 10:00. Select it, try the passed Mon 10:00 (no effect),
	// then move to Wed 10:00 and select that too.
	m = press(t, m, "space", "left", "space", "right", "right", "space")
	got := m.(model).selectedSlots()
	want := []time.Time{
		time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC),
	}
	if len(got) != len(want) || !got[0].Equal(want[0]) || !got[1].Equal(want[1]) {
		t.Fatalf("selected = %v, want %v", got, want)
	}
	if lines := formatSlots(got); lines[0] != "Tue 13 Oct: 10:00" || lines[1] != "Wed 14 Oct: 10:00" {
		t.Fatalf("formatSlots = %q", lines)
	}

	// "q" and "n" must be typeable in the address field rather than quitting.
	m = press(t, m, "enter", "q", "n", "@", "x", "enter")
	if got := m.(model); got.state != stateSending || got.to != "qn@x" {
		t.Fatalf("after typing address: state = %v, to = %q", got.state, got.to)
	}

	m, _ = m.Update(sentMsg{err: errors.New("invalid recipient")})
	if got := m.(model); got.state != stateEmail || got.err == nil {
		t.Fatalf("after send error: state = %v, err = %v", got.state, got.err)
	}

	m, _ = m.Update(sentMsg{})
	m = press(t, m, "y")
	if got := m.(model).state; got != stateEvents {
		t.Fatalf("after sent + y: state = %v, want stateEvents", got)
	}
}

func TestLoadError(t *testing.T) {
	var m tea.Model = newModel(nil, time.Now())
	m, _ = m.Update(eventsMsg{err: errors.New("403")})
	if got := m.(model).state; got != stateFailed {
		t.Fatalf("state = %v, want stateFailed", got)
	}
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"}); cmd == nil {
		t.Fatal("any key should quit after a load error")
	}
}
