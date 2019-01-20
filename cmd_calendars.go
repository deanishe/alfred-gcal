//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-25
//

package main

import (
	"errors"
	"os/exec"
)

var (
	errNoActive = errors.New("no active calendars")
)

// Active calendars
type Active []string

func allCalendars() ([]*Calendar, error) {
	var (
		cals    []*Calendar
		name    = "calendars.json"
		jobName = "update-calendars"
	)

	if wf.Cache.Expired(name, maxAgeCals) {
		if !wf.IsRunning(jobName) {
			wf.Rerun(0.3)
			cmd := exec.Command("./gcal", "update", "calendars")
			if err := wf.RunInBackground(jobName, cmd); err != nil {
				return nil, err
			}
		}
	}

	if wf.Cache.Exists(name) {
		if err := wf.Cache.LoadJSON("calendars.json", &cals); err != nil {
			return nil, err
		}
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
		return IDMap, errNoActive
	}

	if err := wf.Cache.LoadJSON(name, &IDs); err != nil {
		return nil, err
	}
	for _, id := range IDs {
		IDMap[id] = true
	}
	return IDMap, nil
}

func activeCalendars() ([]*Calendar, error) {
	var cals []*Calendar
	IDs, err := activeCalendarIDs()
	if err != nil {
		return nil, err
	}

	all, err := allCalendars()
	if err != nil {
		return nil, err
	}

	for _, c := range all {
		if IDs[c.ID] {
			cals = append(cals, c)
		}
	}
	return cals, nil
}

// doListCalendars shows a list of available calendars in Alfred.
func doListCalendars() error {
	cals, err := allCalendars()
	if err != nil {
		return err
	}

	if len(cals) == 0 && wf.IsRunning("update-calendars") {
		wf.NewItem("Fetching List of Calendars…").
			Subtitle("List will reload shortly").
			Valid(false).
			Icon(reloadIcon())
		wf.Rerun(0.3)
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
		wf.NewItem(c.Title).
			Subtitle(c.Description).
			Icon(icon).
			Arg(c.ID).
			Match(c.Title).
			Valid(true)
	}
	if query != "" {
		wf.Filter(query)
	}
	wf.WarnEmpty("No Calendars", "Did you log in with the right account?")
	wf.SendFeedback()
	return nil
}
