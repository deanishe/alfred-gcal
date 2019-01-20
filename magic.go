//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-25
//

package main

import aw "github.com/deanishe/awgo"

type calendarMagic struct{}

func (cm *calendarMagic) Keyword() string     { return "calendars" }
func (cm *calendarMagic) Description() string { return "Activate/deactivate calendars" }
func (cm *calendarMagic) RunText() string     { return "Opening calendar list…" }
func (cm *calendarMagic) Run() error          { return aw.NewAlfred().RunTrigger("calendars", "") }
