//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-25
//

package main

import (
	aw "github.com/deanishe/awgo"
	"github.com/pkg/errors"
)

// "magic" action to open list of calendars
type calendarMagic struct{}

func (cm *calendarMagic) Keyword() string     { return "calendars" }
func (cm *calendarMagic) Description() string { return "Activate/deactivate calendars" }
func (cm *calendarMagic) RunText() string     { return "Opening calendar list…" }
func (cm *calendarMagic) Run() error          { return aw.NewAlfred().RunTrigger("calendars", "") }

// magic action to log in to a new account
type loginMagic struct{}

func (cm *loginMagic) Keyword() string     { return "login" }
func (cm *loginMagic) Description() string { return "Add a Google account" }
func (cm *loginMagic) RunText() string     { return "Opening Google signin page;" }
func (cm *loginMagic) Run() error {

	acc, err := NewAccount("")
	if err != nil {
		return errors.Wrap(err, "magic: new account")
	}

	if err := acc.FetchCalendars(); err != nil {
		return errors.Wrap(err, "magic: fetch calendars")
	}

	// clear cached schedules now calendars have changed
	return clearEvents()
}
