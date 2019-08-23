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
	"log"
	"os"
	"os/exec"
	"time"

	aw "github.com/deanishe/awgo"
	"github.com/pkg/errors"
)

// doEvents shows a list of events in Alfred.
func doEvents() error {

	if len(accounts) == 0 {
		wf.NewItem("No Accounts Configured").
			Subtitle("Action this item to add a Google account").
			Autocomplete("workflow:login").
			Icon(aw.IconWarning)

		wf.SendFeedback()
		return nil
	}

	var (
		cals []*Calendar
		err  error
	)

	if cals, err = activeCalendars(); err != nil {

		if err == errNoActive {

			wf.NewItem("No Active Calendars").
				Subtitle("Action this item to choose calendars").
				Autocomplete("workflow:calendars").
				Icon(aw.IconWarning)

			wf.SendFeedback()

			return nil
		}

		if err == errNoCalendars {

			if !wf.IsRunning("update-calendars") {
				cmd := exec.Command(os.Args[0], "update", "calendars")
				if err := wf.RunInBackground("update-calendars", cmd); err != nil {
					return errors.Wrap(err, "run calendar update")
				}
			}

			wf.NewItem("Fetching List of Calendars…").
				Subtitle("List will reload shortly").
				Valid(false).
				Icon(ReloadIcon())

			wf.Rerun(0.1)
			wf.SendFeedback()

			return nil
		}

		return err
	}

	log.Printf("%d active calendar(s)", len(cals))

	var (
		all    []*Event
		events []*Event
		parsed time.Time
	)

	if all, err = loadEvents(opts.StartTime, cals...); err != nil {
		return errors.Wrap(err, "load events")
	}

	// Filter out events after cutoff
	for _, e := range all {
		if !opts.ScheduleMode && e.Start.After(opts.EndTime) {
			break
		}
		events = append(events, e)
		log.Printf("%s", e.Title)
	}

	if len(all) == 0 && wf.IsRunning("update-events") {
		wf.NewItem("Fetching Events…").
			Subtitle("Results will refresh shortly").
			Icon(ReloadIcon()).
			Valid(false)

		wf.Rerun(0.1)
	}

	log.Printf("%d event(s) for %s", len(events), opts.StartTime.Format(timeFormat))

	if t, ok := parseDate(opts.Query); ok {
		parsed = t
	}

	if len(events) == 0 && opts.Query == "" {
		wf.NewItem(fmt.Sprintf("No Events on %s", opts.StartTime.Format(timeFormatLong))).
			Icon(ColouredIcon(iconCalendar, yellow))
	}

	var day time.Time

	for _, e := range events {

		// Show day indicator if this is the first event of a given day
		if opts.ScheduleMode && midnight(e.Start).After(day) {

			day = midnight(e.Start)

			wf.NewItem(day.Format(timeFormatLong)).
				Arg(day.Format(timeFormat)).
				Valid(true).
				Icon(iconDay)
		}

		icon := ColouredIcon(iconCalendar, e.Colour)

		sub := fmt.Sprintf("%s – %s / %s",
			e.Start.Local().Format(hourFormat),
			e.End.Local().Format(hourFormat),
			e.CalendarTitle)

		if e.Location != "" {
			sub = sub + " / " + e.Location
		}

		it := wf.NewItem(e.Title).
			Subtitle(sub).
			Icon(icon).
			Arg(e.URL).
			Quicklook(previewURL(opts.StartTime, e.ID)).
			Valid(true).
			Var("action", "open")

		if e.Location != "" {
			app := "Google Maps"
			if opts.UseAppleMaps {
				app = "Apple Maps"
			}

			icon := ColouredIcon(iconMap, e.Colour)
			it.NewModifier("cmd").
				Subtitle("Open in "+app).
				Arg(mapURL(e.Location)).
				Valid(true).
				Icon(icon).
				Var("CALENDAR_APP", "") // Don't open Maps URLs in CALENDAR_APP
		}
	}

	if !opts.ScheduleMode {
		// Navigation items
		prev := opts.StartTime.AddDate(0, 0, -1)
		wf.NewItem("Previous: "+relativeDate(prev)).
			Icon(iconPrevious).
			Arg(prev.Format(timeFormat)).
			Valid(true).
			Var("action", "date")

		next := opts.StartTime.AddDate(0, 0, 1)
		wf.NewItem("Next: "+relativeDate(next)).
			Icon(iconNext).
			Arg(next.Format(timeFormat)).
			Valid(true).
			Var("action", "date")
	}

	if opts.Query != "" {
		wf.Filter(opts.Query)
	}

	if !parsed.IsZero() {

		s := parsed.Format(timeFormat)

		wf.NewItem(parsed.Format(timeFormatLong)).
			Subtitle(relativeDays(parsed, false)).
			Arg(s).
			Autocomplete(s).
			Valid(true).
			Icon(iconDefault)
	}

	wf.WarnEmpty("No Matching Events", "Try a different query?")
	wf.SendFeedback()
	return nil
}

// loadEvents loads events for given date calendar(s) from cache or server.
func loadEvents(t time.Time, cal ...*Calendar) ([]*Event, error) {
	var (
		events  = []*Event{}
		dateStr = t.Format(timeFormat)
		name    = fmt.Sprintf("events-%s.json", dateStr)
		jobName = "update-events"
	)

	if wf.Cache.Expired(name, opts.MaxAgeEvents()) {
		wf.Rerun(0.1)
		if !wf.IsRunning(jobName) {
			cmd := exec.Command(os.Args[0], "update", "events", dateStr)
			if err := wf.RunInBackground(jobName, cmd); err != nil {
				return nil, err
			}
		}
	}

	if wf.Cache.Exists(name) {
		if err := wf.Cache.LoadJSON(name, &events); err != nil {
			return nil, err
		}
	}

	// Set map URL
	for _, e := range events {
		e.MapURL = mapURL(e.Location)
	}
	return events, nil
}
