//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-25
//

package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	oneDay     = time.Hour * 24
	oneWeek    = oneDay * 7
	today      = midnight(time.Now())
	tomorrow   = midnight(today.AddDate(0, 0, 1))
	yesterday  = midnight(today.AddDate(0, 0, -1))
	parseRegex = regexp.MustCompile(`^(\+|-)?(\d+)(d|w)?$`)
)

// doDates shows a list of dates in Alfred.
func doDates() error {

	var parsed bool

	if t, ok := parseDate(opts.DateFormat); ok {

		parsed = true

		s := t.Format(timeFormat)

		wf.NewItem(t.Format(timeFormatLong)).
			Subtitle(relativeDays(t, false)).
			Arg(s).
			Autocomplete(s).
			Valid(true).
			Icon(iconDefault)

	} else {
		for i := -3; i < 4; i++ {

			var (
				t    = midnight(today.Add(oneDay * time.Duration(i)))
				s    = t.Format(timeFormat)
				icon = iconDefault
			)

			if t.Equal(today) {
				icon = iconCalToday
			}

			wf.NewItem(relativeDays(t, true)).
				Subtitle(s).
				Arg(s).
				Autocomplete(s).
				Valid(true).
				Icon(icon)

		}
	}

	if !parsed && opts.DateFormat != "" {
		_ = wf.Filter(opts.DateFormat)
	}

	wf.WarnEmpty("Invalid date", "Format is YYYY-MM-DD, YYYMMDD or [+|-]NN[d|w]")

	wf.SendFeedback()
	return nil
}

// Return midnight in local timezone for given Time.
func midnight(t time.Time) time.Time {
	s := t.Local().Format(timeFormat)
	m, err := time.ParseInLocation(timeFormat, s, time.Local)
	if err != nil {
		panic(err)
	}
	return m
}

// parse string into Time. Boolean is true if parsing was successful.
func parseDate(s string) (time.Time, bool) {

	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}

	if t, err := time.ParseInLocation(timeFormat, s, time.Local); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation("20060102", s, time.Local); err == nil {
		return t, true
	}

	// Parse custom format [+|-]NN[d|w]
	var (
		add   = true
		delta time.Duration
		t     time.Time
		unit  = "d"
	)
	m := parseRegex.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}, false
	}

	// Sign
	if m[1] == "-" {
		add = false
	}
	// Count
	n, err := strconv.Atoi(m[2])
	if err != nil {
		return time.Time{}, false
	}

	if n == 0 {
		return today, true
	}

	// Optional unit
	if m[3] != "" {
		unit = m[3]
	}

	// Calculate date
	if unit == "d" {
		delta = oneDay * time.Duration(n)
	} else {
		delta = oneWeek * time.Duration(n)
	}

	if add {
		t = today.Add(delta)
	} else {
		t = today.Add(-delta)
	}

	return midnight(t), true
}

// Return Time as "x day(s) ago" or "in x day(s)"
func relativeDays(t time.Time, names bool) string {
	var (
		d    time.Duration
		days int
	)
	if t.Before(today) {
		d = today.Sub(t)
	} else if t.After(today) {
		d = t.Sub(today)
	} else {
		return "Today"
	}
	days = int(d.Hours() / 24)

	// Return day name
	if names {
		if days == 1 {
			if t.Before(today) {
				return "Yesterday"
			}
			return "Tomorrow"
		}
		return t.Format("Monday")
	}

	var (
		format string
		unit   = "days"
	)

	// Return in N day(s) or N day(s) ago
	format = "%d %s ago"
	if t.After(today) {
		format = "in %d %s"
	}
	if days == 1 {
		unit = "day"
	}
	return fmt.Sprintf(format, days, unit)
}

// relativeDate returns Yesterday, Today, Tomorrow or long date.
func relativeDate(t time.Time) string {
	t = midnight(t)
	if t.Equal(today) {
		return "Today"
	}
	if t.Equal(yesterday) {
		return "Yesterday"
	}
	if t.Equal(tomorrow) {
		return "Tomorrow"
	}
	return t.Format("Monday, 2 Jan 2006")
}
