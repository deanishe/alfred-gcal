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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	aw "github.com/deanishe/awgo"
	"github.com/deanishe/awgo/util"
	"github.com/pkg/errors"
)

// Check if a new version of the workflow is available.
func doUpdateWorkflow() error {
	wf.Configure(aw.TextErrors(true))

	log.Print("[update] checking for new version of workflow…")

	return wf.CheckForUpdate()
}

// Fetch and cache list of calendars.
func doUpdateCalendars() error {
	var (
		acc *Account
		err error
	)

	wf.Configure(aw.TextErrors(true))

	log.Print("[update] reloading calendars…")

	if len(accounts) == 0 {
		log.Print("[update] no Google accounts configured")
	}

	for _, acc = range accounts {
		if err = acc.FetchCalendars(); err != nil {
			return err
		}

		if !util.PathExists(acc.IconPath()) {
			if err := download(acc.AvatarURL, acc.IconPath()); err != nil {
				return errors.Wrap(err, "fetch account avatar")
			}
		}

		log.Printf("[update] %d calendar(s) in account %q", len(acc.Calendars), acc.Name)
	}

	return nil
}

// Fetch events for a specified date.
func doUpdateEvents() error {
	wf.Configure(aw.TextErrors(true))

	var (
		name   = fmt.Sprintf("events-%s.json", opts.StartTime.Format(timeFormat))
		cals   []*Calendar
		events []*Event
		err    error
	)

	log.Printf("[update] fetching events for %s ...", opts.StartTime.Format(timeFormat))

	if err := clearOldFiles(); err != nil {
		log.Printf("[update] ERR: delete old cache files: %v", err)
	}

	if cals, err = activeCalendars(); err != nil {
		return err
	}

	if len(accounts) == 0 {
		log.Print("[update] no Google accounts configured")
		return nil
	}

	if len(cals) == 0 {
		log.Print("[update] no active calendars")
		return nil
	}

	log.Printf("[update] %d active calendar(s)", len(cals))

	// Fetch events in parallel
	var (
		ch     = make(chan *Event)
		wg     sync.WaitGroup
		wanted = make(map[string]bool, len(cals)) // IDs of calendars to update
	)

	for _, c := range cals {
		wanted[c.ID] = true
	}

	wg.Add(len(cals))

	for _, acc := range accounts {
		for _, c := range acc.Calendars {
			if _, ok := wanted[c.ID]; !ok {
				continue
			}

			go func(c *Calendar, acc *Account) {
				defer wg.Done()

				evs, err := acc.FetchEvents(c, opts.StartTime)
				if err != nil {
					log.Printf("[update] ERR: fetching events for calendar %q: %v", c.Title, err)
					return
				}

				log.Printf("[update] %d event(s) in calendar %q", len(evs), c.Title)

				for _, e := range evs {
					ch <- e
				}
			}(c, acc)
		}
	}

	// Close channel when all goroutines are done
	go func() {
		wg.Wait()
		close(ch)
	}()

	colours := map[string]bool{}
	for e := range ch {
		log.Printf("[update] %s", e)
		events = append(events, e)
		colours[e.Colour] = true
	}

	sort.Sort(EventsByStart(events))

	if err := wf.Cache.StoreJSON(name, events); err != nil {
		return err
	}

	// Ensure icons exist in all colours
	for clr := range colours {
		_ = ColouredIcon(iconCalendar, clr)
		_ = ColouredIcon(iconMap, clr)
	}

	return nil
}

// Remove events-* files and icons older than two weeks.
func clearOldFiles() error {
	var (
		cutoff = time.Now().AddDate(0, 0, -14)
		dirs   = []string{}
	)

	err := filepath.Walk(wf.CacheDir(), func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.Name() == "_aw" && fi.IsDir() {
			return filepath.SkipDir
		}

		if fi.IsDir() {
			dirs = append(dirs, path)
			return nil
		}

		if fi.ModTime().After(cutoff) {
			return nil
		}

		ext := filepath.Ext(path)

		if (strings.HasPrefix(fi.Name(), "events-") && ext == ".json") || ext == ".png" {
			if err := os.Remove(path); err != nil {
				log.Printf("[cache] ERR: delete %q: %v", path, err)
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Remove empty directories. Sort in reverse order so sub-directories are
	// before their parents.
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))

Outer:
	for _, dir := range dirs {
		infos, err := ioutil.ReadDir(dir)
		if err != nil {
			log.Printf("[cache] ERR: open dir %s: %v", dir, err)
			return err
		}

		// ignore dotfiles
		for _, fi := range infos {
			if strings.HasPrefix(fi.Name(), ".") {
				continue
			}
			// rel, _ := filepath.Rel(wf.CacheDir(), dir)
			// log.Printf("[cache] %s -- %d item(s)", rel, len(infos))
			continue Outer
		}

		if err := os.RemoveAll(dir); err != nil {
			log.Printf("[cache] ERR: delete dir %s: %v", dir, err)
			return err
		}
		log.Printf("[cache] deleted dir: %s", util.PrettyPath(dir))
	}

	return nil
}
