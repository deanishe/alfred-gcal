//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-26
//

package main

import (
	"context"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"time"
)

const (
	previewServerURL = "localhost:61433"
	quitAfter        = 60 * time.Second
	// quitAfter = 20 * time.Second
)

// previewURL returns a preview server URL.
func previewURL(t time.Time, eventID string) string {
	u, _ := url.Parse("http://" + previewServerURL)
	v := u.Query()
	v.Set("date", midnight(t).Format(timeFormat))
	v.Set("event", eventID)
	u.RawQuery = v.Encode()
	log.Printf("[preview] url=%s", u.String())
	return u.String()
}

// doStartServer starts the preview server.
func doStartServer() error {
	log.Printf("[preview] starting preview server on %s ...", previewServerURL)
	var (
		lastRequest = time.Now()
		mu          = sync.Mutex{}
		c           = make(chan struct{})
		templates   = template.Must(template.ParseFiles(filepath.Join(wf.Dir(), "preview.html")))
		mux         = http.NewServeMux()
		srv         = &http.Server{
			Addr:    previewServerURL,
			Handler: mux,
		}
	)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				log.Print("[preview] server stopped")
			} else {
				log.Printf("[preview] ERR: server failed: %v", err)
			}
		}
		c <- struct{}{}
	}()

	go func() {
		c := time.Tick(10 * time.Second)
		for now := range c {
			mu.Lock()
			d := now.Sub(lastRequest)
			mu.Unlock()
			log.Printf("[preview] %0.0fs since last request", d.Seconds())
			if d >= quitAfter {
				log.Print("[preview] stopping server ...")
				if err := srv.Shutdown(context.Background()); err != nil {
					log.Printf("[preview] server shutdown error: %v", err)
				}
				// log.Printf("[preview] server stopped")
			}
		}
	}()

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			mu.Lock()
			lastRequest = time.Now()
			mu.Unlock()
		}()

		var (
			v       = req.URL.Query()
			dateStr = v.Get("date")
			eventID = v.Get("event")
			event   *Event
		)
		log.Printf("[preview] date=%s, event=%s", dateStr, eventID)

		// Load events
		t, err := time.Parse(timeFormat, dateStr)
		if err != nil {
			io.WriteString(w, "bad date\n")
			return
		}
		cals, err := activeCalendars()
		if err != nil {
			log.Printf("[preview] ERR: load active calendars: %v", err)
			return
		}
		log.Printf("[preview] %d active calendar(s)", len(cals))
		events, err := loadEvents(t, cals...)
		if err != nil {
			log.Printf("[preview] ERR: load events: %v", err)
			return
		}
		for _, e := range events {
			if e.ID == eventID {
				event = e
				event.MapURL = mapURL(event.Location)
				break
			}
		}

		if event == nil {
			if err := templates.ExecuteTemplate(w, "fail", eventID); err != nil {
				log.Printf(`[preview] ERR: execute template "fail": %v`, err)
			}
			return
		}

		if err := templates.ExecuteTemplate(w, "event", event); err != nil {
			log.Printf(`[preview] ERR: execute template "event": %v`, err)
		}
	})

	<-c
	return nil
}
