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
	"net/url"
	"time"
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

	AccountName string // Name of account this calendar belongs to
}

// CalsByTitle sorts a slice of Calendars by title
type CalsByTitle []*Calendar

func (s CalsByTitle) Len() int           { return len(s) }
func (s CalsByTitle) Less(i, j int) bool { return s[i].Title < s[j].Title }
func (s CalsByTitle) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// Event is a calendar event
type Event struct {
	ID            string    // Event ID
	IcalUID       string    // Cross-platform UID
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

// URL that points to location on Google Maps or Apple Maps.
func mapURL(location string) string {
	if location == "" {
		return ""
	}
	if opts.UseAppleMaps {
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
