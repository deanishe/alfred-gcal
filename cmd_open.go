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

	aw "github.com/deanishe/awgo"
)

// Open URL in specified app or in default.
func doOpen() error {
	wf.Configure(aw.TextErrors(true))
	args := []string{}
	if opts.App != "" {
		log.Printf("[open] opening \"%s\" in \"%s\"…", opts.URL, opts.App)
		args = append(args, "-a", opts.App)
	} else {
		log.Printf("[open] opening \"%s\" in default browser…", opts.URL)
	}
	args = append(args, opts.URL)

	cmd := exec.Command("/usr/bin/open", args...)
	return cmd.Run()
}
