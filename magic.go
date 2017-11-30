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
	"os/exec"
)

type calendarMagic struct{}

func (cm *calendarMagic) Keyword() string     { return "calendars" }
func (cm *calendarMagic) Description() string { return "Activate/deactivate calendars" }
func (cm *calendarMagic) RunText() string     { return "Opening calendar list…" }
func (cm *calendarMagic) Run() error {
	script := fmt.Sprintf(`tell application "Alfred 3" to run trigger "calendars" in workflow "%s"`, wf.BundleID())
	cmd := exec.Command("/usr/bin/osascript", "-e", script)
	return cmd.Run()
}
