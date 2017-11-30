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
)

// Check if a new version of the workflow is available.
func doUpdateWorkflow() error {
	log.Print("[update] checking for new version of workflow…")
	wf.TextErrors = true
	return wf.CheckForUpdate()
}

// Fetch and cache list of calendars.
func doUpdateCalendars() error {
	log.Print("[update] reloading calendars…")
	wf.TextErrors = true
	cals, err := FetchCalendars(auth)
	if err != nil {
		return fmt.Errorf("couldn't load calendars: %v", err)
	}
	return wf.Cache.StoreJSON("calendars.json", cals)
}

// Fetch events for a specified date.
func doUpdateEvents() error {
	log.Printf("[update] fetching events for %s…", startTime.Format(timeFormat))
	wf.TextErrors = true
	var (
		events = []*Event{}
		name   = fmt.Sprintf("events-%s.json", startTime.Format(timeFormat))
	)
	if err := clearOldEvents(); err != nil {
		log.Printf("[update/error] problem deleting old cache files: %v", err)
	}
	cals, err := activeCalendars()
	if err != nil {
		log.Printf("[update/error] couldn't load active calendars: %v", err)
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
				log.Printf("[update/error] fetching events for calendar \"%s\": %v", c.Title, err)
				return
			}

			log.Printf("[update] %d event(s) in calendar \"%s\"", len(evs), c.Title)
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

	for e := range ch {
		log.Printf("[update] %s", e)
		events = append(events, e)
	}

	sort.Sort(EventsByStart(events))

	if err := wf.Cache.StoreJSON(name, events); err != nil {
		return err
	}
	return nil
}

// doUpdateIcons fetches queued icons.
func doUpdateIcons() error {
	gen, err := NewIconGenerator(cacheDirIcons, aw.IconWorkflow)
	if err != nil {
		return err
	}
	return gen.Download()
}

// Remove events-* files that haven't been updated in a week.
func clearOldEvents() error {
	files, err := ioutil.ReadDir(wf.CacheDir())
	if err != nil {
		return err
	}
	for _, fi := range files {
		name := fi.Name()
		cutoff := time.Now().AddDate(0, 0, -7)
		if strings.HasPrefix(name, "events-") && strings.HasSuffix(name, ".json") {
			if fi.ModTime().Before(cutoff) {
				p := filepath.Join(wf.CacheDir(), name)
				if err := os.Remove(p); err != nil {
					log.Printf("[ERROR] couldn't delete file \"%s\": %v", p, err)
				}
			}
		}
	}
	return nil
}
