//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2019-04-03
//

package main

import (
	aw "github.com/deanishe/awgo"
	"log"
)

// quickAdd check if there are configured accounts and pass data to create an event.
func quickAdd() error {

	if len(accounts) == 0 {
		wf.NewItem("No Accounts Configured").
			Subtitle("Action this item to add a Google account").
			Autocomplete("workflow:login").
			Icon(aw.IconWarning)

		wf.SendFeedback()
		return nil
	}

	log.Println("Creating event", opts.Quick, opts.CalendarID)

	if err := createEvent(opts.Quick, opts.CalendarID); err != nil {
		return err
	}

	if err := doUpdateEvents(); err != nil {
		return err
	}

	return nil
}

// createEvent looks for account by calendar ID and create new event in that account.
func createEvent(quick string, calendarID string) error {

	for _, acc := range accounts {
		for _, c := range acc.Calendars {
			if c.ID == calendarID {
				return acc.QuickAdd(calendarID, quick)
			}
		}
	}

	return nil
}
