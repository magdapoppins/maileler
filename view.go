package main

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	accent = lipgloss.Color("212")
	muted  = lipgloss.Color("241")
	green  = lipgloss.Color("42")
	red    = lipgloss.Color("203")

	accentStyle = lipgloss.NewStyle().Foreground(accent)
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(accent).Padding(0, 1)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	mutedStyle  = lipgloss.NewStyle().Foreground(muted)
	errorStyle  = lipgloss.NewStyle().Foreground(red)
	okStyle     = lipgloss.NewStyle().Foreground(green).Bold(true)
	boxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(muted).Padding(0, 2)
	helpStyle   = mutedStyle.MarginTop(1)

	// Week grid cells.
	cellStyle      = lipgloss.NewStyle().Width(7).Align(lipgloss.Center)
	checkedStyle   = cellStyle.Foreground(green).Bold(true)
	busyStyle      = cellStyle.Foreground(muted)
	cursorStyle    = cellStyle.Background(accent).Foreground(lipgloss.Color("230")).Bold(true)
	hourStyle      = mutedStyle.Width(6)
	dayHeadStyle   = cellStyle.Bold(true)
	todayHeadStyle = dayHeadStyle.Foreground(accent)
)

const (
	dayFormat  = "Mon 02 Jan"
	hourFormat = "15:04"
)

func (m model) View() tea.View {
	title := titleStyle.Render("🪼 mailerer")
	week := mutedStyle.Render(fmt.Sprintf("  week of %s", m.start.Format(dayFormat)))

	var body, help string
	switch m.state {
	case stateLoading:
		body = m.spinner.View() + " Loading your calendar…"

	case stateFailed:
		body = errorStyle.Render("Couldn't load your calendar: " + m.err.Error())
		help = "any key: quit"

	case stateEvents:
		body = m.eventsView()
		help = "y/enter: find free times · q: quit"

	case stateSlots:
		body = m.pickerView()
		if len(m.slots) == 0 {
			help = "esc: back · q: quit"
		} else {
			help = "←↑↓→/hjkl: move · space: toggle · a: all/none · enter: continue · esc: back · q: quit"
		}

	case stateEmail:
		body = m.emailView()
		help = "enter: send · esc: back"

	case stateSending:
		body = fmt.Sprintf("%s Sending to %s…", m.spinner.View(), m.to)

	case stateSent:
		body = okStyle.Render("✓ Email sent to "+m.to) + "\n\nStart over?"
		help = "y/enter: back to calendar · q: quit"
	}

	out := title + week + "\n\n" + body + "\n"
	if help != "" {
		out += helpStyle.Render(help) + "\n"
	}
	return tea.NewView(out)
}

func (m model) eventsView() string {
	if len(m.events) == 0 {
		return boxStyle.Render(mutedStyle.Render("No events this week. 🌴")) + "\n\nFind times to meet?"
	}

	var b strings.Builder
	var lastDay time.Time
	for i, e := range m.events {
		if i == 0 || !sameDay(e.StartTime, lastDay) {
			if i > 0 {
				b.WriteString("\n")
			}
			fmt.Fprintln(&b, headerStyle.Render(e.StartTime.Format(dayFormat)))
			lastDay = e.StartTime
		}
		at := e.StartTime.Format(hourFormat)
		if e.AllDay {
			at = "all day"
		}
		fmt.Fprintf(&b, "  %s  %s\n", mutedStyle.Width(7).Render(at), e.Title)
	}
	return boxStyle.Render(strings.TrimRight(b.String(), "\n")) + "\n\nFind times to meet?"
}

func (m model) pickerView() string {
	if len(m.slots) == 0 {
		return boxStyle.Render(mutedStyle.Render("No available time slots this week."))
	}

	var b strings.Builder
	b.WriteString("Pick the times to offer:\n\n")

	b.WriteString(hourStyle.Render(""))
	today := m.now
	for _, d := range m.days {
		style := dayHeadStyle
		if sameDay(d, today) {
			style = todayHeadStyle
		}
		b.WriteString(style.Render(d.Format("Mon 02")))
	}
	b.WriteString("\n")

	for r, h := range m.hours {
		b.WriteString(hourStyle.Render(fmt.Sprintf("%02d:00", h)))
		for c := range m.days {
			b.WriteString(m.cellView(r, c))
		}
		b.WriteString("\n")
	}

	n := len(m.selectedSlots())
	status := mutedStyle.Render(fmt.Sprintf("%d of %d selected", n, len(m.slots)))
	if n > 0 {
		status = okStyle.Render(fmt.Sprintf("%d selected", n)) + mutedStyle.Render(fmt.Sprintf(" of %d", len(m.slots)))
	}
	fmt.Fprintf(&b, "\n%s", status)
	if m.pickerHint != "" {
		fmt.Fprintf(&b, "  %s", errorStyle.Render(m.pickerHint))
	}
	return boxStyle.Render(b.String())
}

func (m model) cellView(r, c int) string {
	i := m.grid[r][c]
	label, style := "·", busyStyle
	if i >= 0 {
		label, style = "[ ]", cellStyle
		if m.selected[i] {
			label, style = "[✓]", checkedStyle
		}
	}
	if r == m.row && c == m.col {
		style = cursorStyle
	}
	return style.Render(label)
}

func (m model) emailView() string {
	var b strings.Builder
	fmt.Fprintln(&b, headerStyle.Render("Times to offer"))
	for _, line := range formatSlots(m.selectedSlots()) {
		fmt.Fprintf(&b, "  %s\n", line)
	}
	fmt.Fprintf(&b, "\nSend to:\n%s", m.input.View())
	if m.err != nil {
		fmt.Fprintf(&b, "\n\n%s", errorStyle.Render("Couldn't send: "+m.err.Error()))
	}
	return boxStyle.Render(b.String())
}

// formatSlots groups slots by day, e.g. "Tue 13 Oct: 09:00, 10:00, 14:00".
// Used both on screen and in the email body.
func formatSlots(slots []time.Time) []string {
	var lines []string
	for i := 0; i < len(slots); {
		day := slots[i]
		var times []string
		for ; i < len(slots) && sameDay(slots[i], day); i++ {
			times = append(times, slots[i].Format(hourFormat))
		}
		lines = append(lines, day.Format(dayFormat)+": "+strings.Join(times, ", "))
	}
	return lines
}
