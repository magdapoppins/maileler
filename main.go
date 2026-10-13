package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	calendarutils "github.com/magdapoppins/mailerer/internal/calendar"
	"github.com/magdapoppins/mailerer/internal/gcloud"
)

// state is the screen the app is currently showing.
type state int

const (
	stateLoading state = iota // fetching this week's events
	stateEvents               // showing events, asking whether to suggest times
	stateSlots                // picking which free slots to include in the email
	stateEmail                // typing the recipient's address
	stateSending              // waiting for Gmail
	stateSent                 // email sent, asking whether to start over
	stateFailed               // events couldn't be loaded; any key quits
)

// Messages delivered back to Update when the API calls finish.
type eventsMsg struct {
	events []calendarutils.Event
	err    error
}

type sentMsg struct{ err error }

type model struct {
	client     *gcloud.Client
	now        time.Time // slots before this are never offered
	start, end time.Time

	state   state
	events  []calendarutils.Event
	input   textinput.Model
	spinner spinner.Model
	to      string
	err     error

	// The slot picker is a week grid: one column per day, one row per hour.
	// grid[row][col] is an index into slots, or -1 when that hour is busy.
	slots      []time.Time
	selected   []bool
	days       []time.Time
	hours      []int
	grid       [][]int
	row, col   int
	pickerHint string // shown when enter is pressed with nothing selected
}

func newModel(client *gcloud.Client, now time.Time) model {
	start, end := calendarutils.GetCurrentWeekRange(now)
	input := textinput.New()
	input.Placeholder = "name@example.com"
	input.SetWidth(32)
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(accentStyle))
	return model{client: client, now: now, start: start, end: end, state: stateLoading, input: input, spinner: sp}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.loadEvents, m.spinner.Tick)
}

func (m model) loadEvents() tea.Msg {
	events, err := m.client.ListEvents(context.Background(), m.start, m.end)
	return eventsMsg{events: events, err: err}
}

func (m model) sendEmail() tea.Msg {
	var body strings.Builder
	body.WriteString("Hi!\n\nHere are some times I'm available to meet this week:\n\n")
	for _, line := range formatSlots(m.selectedSlots()) {
		fmt.Fprintf(&body, "  %s\n", line)
	}
	return sentMsg{err: m.client.SendEmail(context.Background(), m.to, "Available Meeting Times", body.String())}
}

// openPicker lays out the free slots as a Mon–Fri grid with nothing selected.
func (m *model) openPicker() {
	m.slots = calendarutils.FindAvailableTimeSlots(m.events, m.start, m.end, m.now)
	m.selected = make([]bool, len(m.slots))
	m.pickerHint = ""
	m.state = stateSlots

	m.days, m.hours, m.grid = nil, nil, nil
	for d := 0; d < 7; d++ {
		if day := m.start.AddDate(0, 0, d); !calendarutils.IsWeekend(day) {
			m.days = append(m.days, day)
		}
	}
	if len(m.slots) == 0 {
		return
	}

	for h := calendarutils.WorkdayStart; h < calendarutils.WorkdayEnd; h++ {
		m.hours = append(m.hours, h)
		row := make([]int, len(m.days))
		for d := range row {
			row[d] = -1
		}
		m.grid = append(m.grid, row)
	}
	for i, s := range m.slots {
		for d, day := range m.days {
			if sameDay(s, day) {
				m.grid[s.Hour()-calendarutils.WorkdayStart][d] = i
			}
		}
	}

	// Start the cursor on the first free slot.
	m.row, m.col = 0, 0
	for c := range m.days {
		for r := range m.hours {
			if m.grid[r][c] >= 0 {
				m.row, m.col = r, c
				return
			}
		}
	}
}

func (m model) selectedSlots() []time.Time {
	var out []time.Time
	for i, s := range m.slots {
		if m.selected[i] {
			out = append(out, s)
		}
	}
	return out
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case eventsMsg:
		if msg.err != nil {
			m.state, m.err = stateFailed, msg.err
			return m, nil
		}
		m.state, m.events = stateEvents, msg.events
		return m, nil

	case sentMsg:
		if msg.err != nil {
			// Back to the address prompt so the user can fix a typo and retry.
			m.state, m.err = stateEmail, msg.err
			return m, m.input.Focus()
		}
		m.state, m.err = stateSent, nil
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	if m.state == stateEmail {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	// The address field gets every other key, so "q" and "n" can be typed.
	if m.state == stateEmail {
		switch key {
		case "esc":
			m.state, m.err = stateSlots, nil
			m.input.Blur()
			return m, nil
		case "enter":
			m.to = strings.TrimSpace(m.input.Value())
			m.state, m.err = stateSending, nil
			m.input.Blur()
			return m, m.sendEmail
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	if key == "q" || m.state == stateFailed {
		return m, tea.Quit
	}

	switch m.state {
	case stateEvents:
		switch key {
		case "y", "Y", "enter":
			m.openPicker()
		case "n", "N":
			return m, tea.Quit
		}
	case stateSlots:
		return m.handlePickerKey(key)
	case stateSent:
		switch key {
		case "y", "Y", "enter":
			m.state = stateEvents
		case "n", "N":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) handlePickerKey(key string) (tea.Model, tea.Cmd) {
	if key == "esc" || key == "b" {
		m.state = stateEvents
		return m, nil
	}
	if len(m.slots) == 0 {
		return m, nil
	}

	switch key {
	case "up", "k":
		m.row = max(m.row-1, 0)
	case "down", "j":
		m.row = min(m.row+1, len(m.hours)-1)
	case "left", "h":
		m.col = max(m.col-1, 0)
	case "right", "l":
		m.col = min(m.col+1, len(m.days)-1)
	case "space", "x":
		if i := m.grid[m.row][m.col]; i >= 0 {
			m.selected[i] = !m.selected[i]
			m.pickerHint = ""
		}
	case "a":
		// Select everything, or clear everything if it's all selected already.
		all := len(m.selectedSlots()) == len(m.slots)
		for i := range m.selected {
			m.selected[i] = !all
		}
		m.pickerHint = ""
	case "enter":
		if len(m.selectedSlots()) == 0 {
			m.pickerHint = "Pick at least one time first."
			return m, nil
		}
		m.state = stateEmail
		m.input.Reset()
		return m, m.input.Focus()
	}
	return m, nil
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func main() {
	// Log in before the UI starts: the first-run login flow reads from stdin.
	client, err := gcloud.NewClient(context.Background(), "credentials.json")
	if err != nil {
		log.Fatal(err)
	}

	if _, err := tea.NewProgram(newModel(client, time.Now())).Run(); err != nil {
		log.Fatal(err)
	}
}
