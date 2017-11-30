//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-30
//

package main

import (
	"log"
	"os/exec"
)

// Open URL in specified app or in default.
func doOpen() error {
	wf.TextErrors = true
	args := []string{}
	if openApp != "" {
		log.Printf("[open] opening \"%s\" in \"%s\"…", calURL, openApp)
		args = append(args, "-a", openApp)
	} else {
		log.Printf("[open] opening \"%s\" in default browser…", calURL)
	}
	args = append(args, calURL)

	cmd := exec.Command("/usr/bin/open", args...)
	return cmd.Run()
}
