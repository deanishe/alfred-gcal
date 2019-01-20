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
)

// Check if a new version of the workflow is available.
func doUpdateWorkflow() error {

	wf.Configure(aw.TextErrors(true))

	log.Print("[update] checking for new version of workflow…")

	return wf.CheckForUpdate()
}

// Fetch and cache list of calendars.
func doUpdateCalendars() error {

	wf.Configure(aw.TextErrors(true))

	log.Print("[update] reloading calendars…")

	cals, err := FetchCalendars(auth)
	if err != nil {
		log.Printf("[update] ERR: retrieve calendars: %v", err)
		return err
	}

	return wf.Cache.StoreJSON("calendars.json", cals)
}

// Fetch events for a specified date.
func doUpdateEvents() error {

	wf.Configure(aw.TextErrors(true))

	var (
		events = []*Event{}
		name   = fmt.Sprintf("events-%s.json", startTime.Format(timeFormat))
	)

	log.Printf("[update] fetching events for %s ...", startTime.Format(timeFormat))

	if err := clearOldFiles(); err != nil {
		log.Printf("[update] ERR: delete old cache files: %v", err)
	}

	cals, err := activeCalendars()
	if err != nil {
		log.Printf("[update] ERR: load active calendars: %v", err)
		return err
	}
	log.Printf("[update] %d active calendar(s)", len(cals))

	// Fetch events in parallel
	var (
		ch = make(chan *Event)
		wg sync.WaitGroup
	)

	wg.Add(len(cals))

	for _, c := range cals {

		go func(c *Calendar) {

			defer wg.Done()

			evs, err := FetchEvents(auth, c, startTime)
			if err != nil {
				log.Printf("[update] ERR: fetching events for calendar %q: %v", c.Title, err)
				return
			}

			log.Printf("[update] %d event(s) in calendar %q", len(evs), c.Title)

			for _, e := range evs {
				ch <- e
				// events = append(events, e)
			}
		}(c)
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
		err    error
	)

	err = filepath.Walk(wf.CacheDir(), func(path string, fi os.FileInfo, err error) error {

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
