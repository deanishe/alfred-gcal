//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-30
//

package main

// Show reload bar.
func doReload() error {
	wf.Rerun(0.1)

	wf.NewItem("Progress…").
		Icon(ReloadIcon())

	wf.SendFeedback()

	return nil
}
