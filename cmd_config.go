//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-25
//

package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	aw "github.com/deanishe/awgo"
	"github.com/deanishe/awgo/util"
)

// doConfig shows configuration options.
func doConfig() error {
	wf.Var("CALENDAR_APP", "") // Open links in default browser, not CALENDAR_APP
	wf.NewItem("Active Calendars").
		Subtitle("Turn calendars on/off").
		Icon(iconCalOn).
		Valid(true).
		Var("action", "calendars")

	if wf.UpdateAvailable() {
		wf.NewItem("An Update is Available").
			Subtitle("A newer version of the workflow is available").
			Autocomplete("workflow:update").
			Icon(iconUpdateAvailable).
			Valid(false)
	} else {
		wf.NewItem("Workflow is up to Date").
			Subtitle("Action to force update check").
			Icon(iconUpdateOK).
			Valid(true).
			Var("action", "update")
	}

	wf.NewItem("Open Documentation").
		Subtitle("Open workflow README in your browser").
		Arg(readmeURL).
		Valid(true).
		Icon(iconDocs).
		Var("action", "open")

	wf.NewItem("Get Help").
		Subtitle("Open alfredforum.com thread in your browser").
		Arg(forumURL).
		Valid(true).
		Icon(iconHelp).
		Var("action", "open")

	wf.NewItem("Report Issue").
		Subtitle("Open GitHub issues in your browser").
		Arg(helpURL).
		Valid(true).
		Icon(iconIssue).
		Var("action", "open")

	wf.NewItem("Clear Cached Calendars & Events").
		Subtitle("Remove cached list of calendars and events").
		Icon(iconDelete).
		Valid(true).
		Var("action", "clear")

	if opts.Query != "" {
		wf.Filter(opts.Query)
	}

	wf.WarnEmpty("No Matches", "Try a different query")
	wf.SendFeedback()
	return nil
}

// doToggle turns a calendar on or off.
func doToggle() error {
	IDs, err := activeCalendarIDs()
	if err != nil && err != errNoActive {
		return err
	}
	if IDs[opts.CalendarID] {
		log.Printf("deactivating calendar %s ...", opts.CalendarID)
		delete(IDs, opts.CalendarID)
	} else {
		log.Printf("activating calendar %s ...", opts.CalendarID)
		IDs[opts.CalendarID] = true
	}

	active := []string{}
	for ID := range IDs {
		active = append(active, ID)
	}
	return wf.Cache.StoreJSON("active.json", active)
}

// doClear removes cached calendars and events.
func doClear() error {
	log.Print("clearing cached calendars and events…")
	wf.Configure(aw.TextErrors(true))

	paths := []string{filepath.Join(wf.CacheDir(), "calendars.json")}

	files, err := ioutil.ReadDir(wf.CacheDir())
	if err != nil {
		return err
	}

	for _, fi := range files {
		fn := fi.Name()
		if strings.HasPrefix(fn, "events-") && strings.HasSuffix(fn, ".json") {
			paths = append(paths, filepath.Join(wf.CacheDir(), fn))
		}
	}

	for _, p := range paths {
		if err := os.Remove(p); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			log.Printf("[ERROR] couldn't delete \"%s\": %v", util.PrettyPath(p), err)
			return err
		}
		log.Printf("deleted \"%s\"", util.PrettyPath(p))
	}
	return nil
}
