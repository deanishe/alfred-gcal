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
	"log"
	"net/url"
	"sort"
	"time"

	calendar "google.golang.org/api/calendar/v3"
)

const (
	gMapsURL = "https://www.google.com/maps/search/?api=1"
	aMapsURL = "http://maps.apple.com/"
)

// Calendar is a Google Calendar
type Calendar struct {
	ID          string // Calendar ID
	Title       string // Calendar title
	Description string // Calendar description
	Colour      string // CSS hex colour of calendar
}

// CalsByTitle sorts a slice of Calendars by title
type CalsByTitle []*Calendar

func (s CalsByTitle) Len() int           { return len(s) }
func (s CalsByTitle) Less(i, j int) bool { return s[i].Title < s[j].Title }
func (s CalsByTitle) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// FetchCalendars retrieves a list of the user's calendars.
func FetchCalendars(auth *Authenticator) ([]*Calendar, error) {

	srv, err := calendarService(auth)
	if err != nil {
		return nil, err
	}

	ls, err := srv.CalendarList.List().Do()
	if err != nil {
		return nil, err
	}

	var cals []*Calendar
	for _, entry := range ls.Items {
		if entry.Hidden {
			log.Printf("[events] ignoring hidden calendar \"%s\"", entry.Summary)
			continue
		}

		c := &Calendar{
			ID:          entry.Id,
			Title:       entry.Summary,
			Description: entry.Description,
			Colour:      entry.BackgroundColor,
		}
		if entry.SummaryOverride != "" {
			c.Title = entry.SummaryOverride
		}
		cals = append(cals, c)
	}
	sort.Sort(CalsByTitle(cals))
	return cals, nil
}

// Event is a calendar event
type Event struct {
	ID            string    // Calendar ID
	Title         string    // Event title
	Description   string    // Event summary/description
	URL           string    // Event URL
	MapURL        string    // Google Maps URL
	Location      string    // Where the event takes place
	Start         time.Time // Time event started
	End           time.Time // Time event finished
	Colour        string    // CSS hex colour of event
	CalendarID    string    // Calendar event belongs to
	CalendarTitle string    // Title of calendar event belongs to
}

// Duration returns the duration of the Event
func (e *Event) Duration() time.Duration { return e.End.Sub(e.Start) }

func (e *Event) String() string {
	date := e.Start.Format("2/1 at 15:04")
	return fmt.Sprintf("\"%s\" on %s for %0.0fm", e.Title, date, e.Duration().Minutes())
}

// EventsByStart sorts a slice of Events by start time.
type EventsByStart []*Event

func (s EventsByStart) Len() int           { return len(s) }
func (s EventsByStart) Less(i, j int) bool { return s[i].Start.Before(s[j].Start) }
func (s EventsByStart) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// FetchEvents retrieves events for given date
func FetchEvents(auth *Authenticator, cal *Calendar, start time.Time) ([]*Event, error) {
	var (
		end       = start.Add(scheduleDuration)
		events    = []*Event{}
		startTime = start.Format(time.RFC3339)
		endTime   = end.Format(time.RFC3339)
	)

	log.Printf("[events] cal=\"%s\", start=%s, end=%s", cal.Title, start, end)

	srv, err := calendarService(auth)
	if err != nil {
		return nil, err
	}

	evs, err := srv.Events.List(cal.ID).
		SingleEvents(true).
		MaxResults(2500).
		TimeMin(startTime).
		TimeMax(endTime).
		OrderBy("startTime").Do()

	if err != nil {
		return nil, fmt.Errorf("couldn't retrieve events for calendar %s: %v", cal.ID, err)
	}

	for _, e := range evs.Items {
		if e.Start.DateTime == "" { // all-day event
			continue
		}
		start, err := time.Parse(time.RFC3339, e.Start.DateTime)
		if err != nil {
			log.Printf("[events/error] couldn't parse start time (%s): %v", e.Start.DateTime, err)
			continue
		}
		end, err := time.Parse(time.RFC3339, e.End.DateTime)
		if err != nil {
			log.Printf("[events/error] couldn't parse end time (%s): %v", e.End.DateTime, err)
			continue
		}

		events = append(events, &Event{
			ID:            e.Id,
			Title:         e.Summary,
			Description:   e.Description,
			URL:           e.HtmlLink,
			Location:      e.Location,
			Start:         start,
			End:           end,
			Colour:        cal.Colour,
			CalendarID:    cal.ID,
			CalendarTitle: cal.Title,
		})
	}
	return events, nil
}

func calendarService(auth *Authenticator) (*calendar.Service, error) {
	client, err := auth.GetClient()
	if err != nil {
		return nil, fmt.Errorf("couldn't get API client: %v", err)
	}

	srv, err := calendar.New(client)
	if err != nil {
		return nil, fmt.Errorf("couldn't create Calendar Service: %v", err)
	}
	return srv, nil
}

// URL that points to location on Google Maps or Apple Maps.
func mapURL(location string) string {
	if location == "" {
		return ""
	}
	if useAppleMaps {
		return appleMapsURL(location)
	}
	return googleMapsURL(location)
}

func googleMapsURL(location string) string {
	u, _ := url.Parse(gMapsURL)
	v := u.Query()
	v.Set("query", location)
	u.RawQuery = v.Encode()
	return u.String()
}

func appleMapsURL(location string) string {
	u, _ := url.Parse(aMapsURL)
	v := u.Query()
	v.Set("address", location)
	u.RawQuery = v.Encode()
	return u.String()
}
