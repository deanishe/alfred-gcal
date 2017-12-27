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
	var (
		dates  = []time.Time{}
		parsed bool // whether date was parsed from user input
	)

	if dateFormat == "" { // show default list
		for i := -3; i < 4; i++ {
			dates = append(dates, midnight(today.Add(oneDay*time.Duration(i))))
		}
	} else {
		t, err := parseDate(dateFormat)
		if err != nil {
			wf.Warn("Invalid date", "Format is YYYY-MM-DD, YYYMMDD or [+|-]NN[d|w]")
			return nil
		}
		parsed = true
		dates = append(dates, t)
	}

	for _, t := range dates {
		var sub, title string
		dateStr := t.Format(timeFormat)
		longDate := t.Format(timeFormatLong)
		title = relativeDays(t, !parsed)
		sub = dateStr
		if parsed {
			title, sub = longDate, title
		}
		icon := iconDefault
		if t.Equal(today) {
			icon = iconCalToday
		}
		wf.NewItem(title).
			Subtitle(sub).
			Arg(dateStr).
			Autocomplete(dateStr).
			Valid(true).
			Icon(icon)
	}

	wf.SendFeedback()
	return nil
}

// Return midnight in local timezone for given Time.
func midnight(t time.Time) time.Time {
	s := t.In(time.Local).Format(timeFormat)
	m, err := time.ParseInLocation(timeFormat, s, time.Local)
	if err != nil {
		panic(err)
	}
	return m
}

func parseDate(s string) (time.Time, error) {
	if t, err := time.ParseInLocation(timeFormat, s, time.Local); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("20060102", s, time.Local); err == nil {
		return t, nil
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
		return time.Time{}, fmt.Errorf("invalid format: %s", s)
	}

	// Sign
	if m[1] == "-" {
		add = false
	}
	// Count
	n, err := strconv.Atoi(m[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid number: %s", m[2])
	}

	if n == 0 {
		return today, nil
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

	return midnight(t), nil
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
