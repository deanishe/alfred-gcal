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
	"github.com/pkg/errors"
)

// doConfig shows configuration options.
func doConfig() error {
	wf.Var("CALENDAR_APP", "") // Open links in default browser, not CALENDAR_APP

	if opts.Query == "" {
		wf.Configure(aw.SuppressUIDs(true))
	}

	if len(accounts) > 0 {

		wf.NewItem("Active Calendars…").
			Subtitle("Turn calendars on/off").
			UID("calendars").
			Icon(iconCalendars).
			Valid(true).
			Var("action", "calendars")

		wf.NewItem("Add Account…").
			Subtitle("Action this item to add a Google account").
			UID("add-account").
			Autocomplete("workflow:login").
			Icon(iconAccountAdd)

	} else {

		wf.NewItem("No Accounts Configured").
			Subtitle("Action this item to add a Google account").
			UID("add-account").
			Autocomplete("workflow:login").
			Icon(aw.IconWarning)
	}

	for _, acc := range accounts {

		it := wf.NewItem(acc.Name).
			Subtitle("↩ to remove account / ⌘↩ to re-authenticate").
			UID(acc.Name).
			Arg(acc.Name).
			Valid(true).
			Icon(acc.Icon()).
			Var("action", "logout").
			Var("account", acc.Name)

		it.NewModifier("cmd").
			Subtitle("Re-authenticate account with read-write permission").
			Var("action", "reauth").
			Var("account", acc.Name)

	}

	var (
		name  = "Google Maps"
		other = "Apple Maps"
		icon  = iconGoogleMaps
		arg   = "apple"
	)

	if opts.UseAppleMaps {
		name, other = other, name
		icon = iconAppleMaps
		arg = "google"
	}

	wf.NewItem("Open Locations in "+name).
		Subtitle("Toggle this setting to use "+other).
		UID("location").
		Arg(arg).
		Valid(true).
		Icon(icon).
		Var("action", "set").
		Var("key", "maps").
		Var("value", arg)

	if wf.UpdateAvailable() {
		wf.NewItem("An Update is Available").
			Subtitle("A newer version of the workflow is available").
			UID("update").
			Autocomplete("workflow:update").
			Icon(iconUpdateAvailable).
			Valid(false)
	} else {
		wf.NewItem("Workflow is up to Date").
			Subtitle("Action to force update check").
			UID("update").
			Icon(iconUpdateOK).
			Valid(true).
			Var("action", "update")
	}

	wf.NewItem("Open Documentation").
		Subtitle("Open workflow README in your browser").
		UID("docs").
		Arg(readmeURL).
		Valid(true).
		Icon(iconDocs).
		Var("action", "open")

	wf.NewItem("Get Help").
		Subtitle("Open alfredforum.com thread in your browser").
		UID("forum").
		Arg(forumURL).
		Valid(true).
		Icon(iconHelp).
		Var("action", "open")

	wf.NewItem("Report Issue").
		Subtitle("Open GitHub issues in your browser").
		UID("issues").
		Arg(helpURL).
		Valid(true).
		Icon(iconIssue).
		Var("action", "open")

	wf.NewItem("Clear Cached Calendars & Events").
		Subtitle("Remove cached list of calendars and events").
		UID("clear").
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
	if err == errNoActive {
		IDs = map[string]bool{}
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

	if err := wf.Cache.StoreJSON("active.json", active); err != nil {
		return errors.Wrap(err, "save active calendar list")
	}

	// calendars have changed, so delete cached schedules
	return clearEvents()
}

// Re-authenticate specified account.
func doReauth() error {
	wf.Configure(aw.TextErrors(true))
	log.Printf("[reauth] account=%q", opts.Account)

	for _, acc := range accounts {

		if acc.Name == opts.Account {
			acc.Token = nil
			if err := acc.Save(); err != nil {
				return errors.Wrap(err, "reauth: save account")
			}

			// retrieve calendar list to trigger authentication
			if err := acc.FetchCalendars(); err != nil {
				return errors.Wrap(err, "reauth: fetch calendars")
			}
		}
	}

	return nil
}

// doLogout removes an account.
func doLogout() error {

	wf.Configure(aw.TextErrors(true))

	log.Printf("[logout] account=%q", opts.Account)

	deleteMe := map[string]bool{}

	for _, acc := range accounts {

		if acc.Name == opts.Account {

			for _, cal := range acc.Calendars {
				deleteMe[cal.ID] = true
			}

			if err := wf.Cache.Store(acc.CacheName(), nil); err != nil {
				return errors.Wrap(err, "delete account file")
			}
			if err := os.Remove(acc.IconPath()); err != nil && !os.IsNotExist(err) {
				return errors.Wrap(err, "delete account avatar")
			}

			log.Printf("[logout] removed account %q", opts.Account)
		}
	}

	var (
		active []string
		IDs    map[string]bool
		err    error
	)

	// Remove active calendars belonging to account
	if IDs, err = activeCalendarIDs(); err != nil && err != errNoActive {
		return errors.Wrap(err, "get active calendars")
	}

	// No active calendars to change
	if err == errNoActive || len(deleteMe) == 0 {
		return nil
	}

	for id := range IDs {
		if !deleteMe[id] {
			active = append(active, id)
		}
	}

	if err := wf.Cache.StoreJSON("active.json", active); err != nil {
		return errors.Wrap(err, "save active calendar list")
	}

	// delete cached schedules now calendars have changed
	return clearEvents()
}

// doClear removes cached calendars and events.
func doClear() error {
	log.Print("clearing cached calendars and events…")
	wf.Configure(aw.TextErrors(true))

	if err := clearEvents(); err != nil {
		return errors.Wrap(err, "clear cached data")
	}

	for _, acc := range accounts {
		acc.Calendars = []*Calendar{}
		if err := acc.Save(); err != nil {
			return errors.Wrap(err, "remove account calendars")
		}
	}

	return nil
}

// delete cached events.
func clearEvents() error {

	var (
		infos []os.FileInfo
		err   error
	)

	if infos, err = ioutil.ReadDir(wf.CacheDir()); err != nil {
		return errors.Wrap(err, "read cache directory")
	}

	for _, fi := range infos {
		name := fi.Name()
		if strings.HasPrefix(name, "events-") && strings.HasSuffix(name, ".json") {
			if err = os.Remove(filepath.Join(wf.CacheDir(), name)); err != nil {
				return errors.Wrap(err, "delete events cache file")
			}

			log.Printf("[cache] deleted %q", name)
		}
	}

	return err
}
