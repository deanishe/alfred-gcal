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
	"os/exec"
	"time"

	aw "github.com/deanishe/awgo"
)

// doEvents shows a list of events in Alfred.
func doEvents() error {
	var (
		last = today
		cur  = today
	)
	gen, err := NewIconGenerator(cacheDirIcons, aw.IconWorkflow)
	if err != nil {
		return err
	}

	cals, err := activeCalendars()
	if err != nil {
		if err == errNoActive {
			wf.NewItem("No Active Calendars").
				Subtitle("Action this item to choose calendars").
				Autocomplete("workflow:calendars").
				Icon(aw.IconWarning)
			wf.SendFeedback()
			return nil
		}
		return err
	}
	log.Printf("%d active calendar(s)", len(cals))

	if len(cals) == 0 && wf.IsRunning("update-calendars") {
		wf.NewItem("Fetching List of Calendars…").
			Subtitle("List will reload shortly").
			Valid(false).
			Icon(reloadIcon())

		wf.Rerun(0.1)
		wf.SendFeedback()

		return nil
	}

	all, err := loadEvents(startTime, cals...)
	if err != nil {
		return err
	}

	events := []*Event{}

	// Filter out events after cutoff
	for _, e := range all {
		if !schedule && e.Start.After(endTime) {
			break
		}
		events = append(events, e)
		log.Printf("%s", e.Title)
	}

	if len(all) == 0 && wf.IsRunning("update-events") {
		wf.NewItem("Fetching Events…").
			Subtitle("Results will refresh shortly").
			Icon(reloadIcon()).
			Valid(false)
	}

	log.Printf("%d event(s) for %s", len(events), startTime.Format(timeFormat))

	if len(events) == 0 && opts.Query == "" {
		wf.NewItem(fmt.Sprintf("No Events on %s", startTime.Format(timeFormatLong))).
			Icon(aw.IconWorkflow)
	}

	for i, e := range events {

		if schedule { // Show day indicator

			cur = midnight(e.Start)

			// Show current date if this is the first item or the first of
			// a new day.
			if cur.After(last) || i == 0 {
				last = cur
				wf.NewItem(cur.Format(timeFormatLong)).
					Arg(cur.Format(timeFormat)).
					Valid(true).
					Icon(iconDay)
			}
		}

		icon := gen.Icon(eventIconFont, eventIconName, e.Colour)
		sub := fmt.Sprintf("%s – %s / %s", e.Start.Local().Format("15:04"), e.End.Local().Format("15:04"), e.CalendarTitle)
		it := wf.NewItem(e.Title).
			Subtitle(sub).
			Icon(icon).
			Arg(e.URL).
			Quicklook(previewURL(startTime, e.ID)).
			Valid(true).
			Var("action", "open")

		if e.Location != "" {
			app := "Google Maps"
			if useAppleMaps {
				app = "Apple Maps"
			}
			icon := gen.Icon(mapIconFont, mapIconName, e.Colour)
			it.NewModifier("cmd").
				Subtitle("Open in "+app).
				Arg(mapURL(e.Location)).
				Valid(true).
				Icon(icon).
				Var("CALENDAR_APP", "") // Don't open Maps URLs in CALENDAR_APP
		}
		// log.Printf(`"%s" (%s)`, e.Title, e.IcalUID)
	}

	if !schedule {
		// Navigation items
		prev := startTime.AddDate(0, 0, -1)
		wf.NewItem("Previous: "+relativeDate(prev)).
			Icon(iconPrevious).
			Arg(prev.Format(timeFormat)).
			Valid(true).
			Var("action", "date")

		next := startTime.AddDate(0, 0, 1)
		wf.NewItem("Next: "+relativeDate(next)).
			Icon(iconNext).
			Arg(next.Format(timeFormat)).
			Valid(true).
			Var("action", "date")
	}

	if opts.Query != "" {
		wf.Filter(opts.Query)
	}

	if gen.HasQueue() {
		wf.Rerun(0.1)
		if err := gen.Save(); err != nil {
			return err
		}
		if !wf.IsRunning("icons") {
			cmd := exec.Command("./gcal", "update", "icons")
			if err := wf.RunInBackground("icons", cmd); err != nil {
				return err
			}
		}
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

	if wf.Cache.Expired(name, maxAgeEvents) {
		wf.Rerun(0.1)
		if !wf.IsRunning(jobName) {
			cmd := exec.Command("./gcal", "update", "events", dateStr)
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
