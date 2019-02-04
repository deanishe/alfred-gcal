//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-25
//

package main

import (
	"log"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

var (
	errNoActive    = errors.New("no active calendars")
	errNoCalendars = errors.New("no calendars")
)

// doListCalendars shows a list of available calendars in Alfred.
func doListCalendars() error {

	var (
		cals []*Calendar
		err  error
	)

	if cals, err = allCalendars(); err != nil {

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

	if len(cals) == 0 && wf.IsRunning("update-calendars") {
		wf.NewItem("Fetching List of Calendars…").
			Subtitle("List will reload shortly").
			Valid(false).
			Icon(ReloadIcon())
		wf.Rerun(0.1)
		wf.SendFeedback()
		return nil
	}

	active, err := activeCalendarIDs()
	if err != nil && err != errNoActive {
		return err
	}

	for _, c := range cals {

		on := active[c.ID]
		icon := iconCalOff
		if on {
			icon = iconCalOn
		}
		sub := c.Description + " / " + c.AccountName
		if c.Description == "" {
			sub = c.AccountName
		}

		wf.NewItem(c.Title).
			Subtitle(sub).
			Icon(icon).
			Arg(c.ID).
			Match(c.Title).
			Valid(true)
	}

	if opts.Query != "" {
		wf.Filter(opts.Query)
	}

	wf.WarnEmpty("No Calendars", "Did you log in with the right account?")
	wf.SendFeedback()

	return nil
}

func allCalendars() ([]*Calendar, error) {
	var (
		jobName = "update-calendars"
		cals    []*Calendar
		expired bool
	)

	for _, acc := range accounts {
		if wf.Cache.Expired(acc.CacheName(), opts.MaxAgeCalendar()) {
			expired = true
		}
		cals = append(cals, acc.Calendars...)
	}

	if expired {

		if !wf.IsRunning(jobName) {

			wf.Rerun(0.1)

			cmd := exec.Command(os.Args[0], "update", "calendars")
			if err := wf.RunInBackground(jobName, cmd); err != nil {
				return nil, err
			}
		}
	}

	log.Printf("[main] %d calendar(s) in %d account(s)", len(cals), len(accounts))

	if len(cals) == 0 {
		return nil, errNoCalendars
	}

	return cals, nil
}

func activeCalendarIDs() (map[string]bool, error) {
	var (
		IDs   []string
		IDMap = map[string]bool{}
		name  = "active.json"
	)

	if !wf.Cache.Exists(name) {
		return nil, errNoActive
	}

	if err := wf.Cache.LoadJSON(name, &IDs); err != nil {
		return nil, err
	}
	for _, id := range IDs {
		IDMap[id] = true
	}

	if len(IDMap) == 0 {
		return nil, errNoActive
	}

	return IDMap, nil
}

func activeCalendars() ([]*Calendar, error) {
	var (
		cals []*Calendar
		all  []*Calendar
		IDs  map[string]bool
		err  error
	)

	if IDs, err = activeCalendarIDs(); err != nil {
		return nil, err
	}

	if all, err = allCalendars(); err != nil {
		return nil, err
	}

	if len(all) == 0 {
		return nil, errNoCalendars
	}

	for _, c := range all {
		if IDs[c.ID] {
			cals = append(cals, c)
		}
	}

	if len(cals) == 0 {
		return nil, errNoActive
	}

	return cals, nil
}
