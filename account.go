// Copyright (c) 2019 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	aw "github.com/deanishe/awgo"
	"github.com/deanishe/awgo/util"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
)

// Account is a Google account. It contains user's email, avatar URL and OAuth2
// token.
type Account struct {
	Name      string // Directory account data is stored in
	Email     string // User's email address
	AvatarURL string // URL of user's Google avatar
	ReadWrite bool   // Define whether account has write permissions or not

	Calendars []*Calendar // Calendars contained by account

	Token *oauth2.Token // OAuth2 authentication token
	auth  *Authenticator
}

// NewAccount creates a new account or loads an existing one.
func NewAccount(name string) (*Account, error) {

	var (
		a   = &Account{Name: name}
		err error
	)

	if name != "" {
		if err = wf.Cache.LoadJSON(a.CacheName(), a); err != nil {
			return nil, errors.Wrap(err, "load account")
		}
	}

	return a, nil
}

// LoadAccounts reads saved accounts from disk.
func LoadAccounts() ([]*Account, error) {

	var (
		accounts = []*Account{}
		infos    []os.FileInfo
		err      error
	)

	if infos, err = ioutil.ReadDir(wf.CacheDir()); err != nil {
		return nil, errors.Wrap(err, "read accountsDir")
	}

	for _, fi := range infos {
		if fi.IsDir() ||
			!strings.HasSuffix(fi.Name(), ".json") ||
			!strings.HasPrefix(fi.Name(), "account-") {
			continue
		}

		acc := &Account{}
		if err := wf.Cache.LoadJSON(fi.Name(), acc); err != nil {
			return nil, errors.Wrap(err, "load account")
		}
		log.Printf("[account] loaded %+v", acc)

		accounts = append(accounts, acc)
	}

	return accounts, nil
}

// CacheName returns the name of Account's cache file.
func (a *Account) CacheName() string { return "account-" + a.Name + ".json" }

// IconPath returns the path to the cached user avatar.
func (a *Account) IconPath() string {
	return filepath.Join(cacheDirIcons, a.Name+filepath.Ext(a.AvatarURL))
}

// Icon returns Account user avatar.
func (a *Account) Icon() *aw.Icon {
	p := a.IconPath()
	if util.PathExists(p) {
		return &aw.Icon{Value: p}
	}

	return iconAccount
}

// Authenticator creates a new Authenticator for Account.
func (a *Account) Authenticator() *Authenticator {
	if a.auth == nil {
		a.auth = NewAuthenticator(a, []byte(secret))
	}

	return a.auth
}

// Save saves authentication token.
func (a *Account) Save() error {
	if err := wf.Cache.StoreJSON(a.CacheName(), a); err != nil {
		return errors.Wrap(err, "save account")
	}
	log.Printf("[account] saved %q", a.Name)
	return nil
}

// Service returns a Calendar Service for this Account.
func (a *Account) Service() (*calendar.Service, error) {

	var (
		client *http.Client
		srv    *calendar.Service
		err    error
	)

	if client, err = a.Authenticator().GetClient(); err != nil {
		return nil, errors.Wrap(err, "get authenticator client")
	}

	if srv, err = calendar.New(client); err != nil {
		return nil, errors.Wrap(err, "create new calendar client")
	}

	return srv, nil
}

// FetchCalendars retrieves a list of all calendars in Account.
func (a *Account) FetchCalendars() error {

	var (
		srv  *calendar.Service
		ls   *calendar.CalendarList
		cals []*Calendar
		err  error
	)

	if srv, err = a.Service(); err != nil {
		return errors.Wrap(err, "create service")
	}

	if ls, err = srv.CalendarList.List().Do(); err != nil {
		return errors.Wrap(err, "retrieve calendar list")
	}

	for _, entry := range ls.Items {
		if entry.Hidden {
			log.Printf("[account] ignoring hidden calendar %q in %q", entry.Summary, a.Name)
			continue
		}

		c := &Calendar{
			ID:          entry.Id,
			Title:       entry.Summary,
			Description: entry.Description,
			Colour:      entry.BackgroundColor,
			AccountName: a.Name,
		}
		if entry.SummaryOverride != "" {
			c.Title = entry.SummaryOverride
		}
		cals = append(cals, c)
	}

	sort.Sort(CalsByTitle(cals))
	a.Calendars = cals
	return a.Save()
}

// FetchEvents returns events from the specified calendar.
func (a *Account) FetchEvents(cal *Calendar, start time.Time) ([]*Event, error) {

	var (
		end       = start.Add(opts.ScheduleDuration())
		events    = []*Event{}
		startTime = start.Format(time.RFC3339)
		endTime   = end.Format(time.RFC3339)
		srv       *calendar.Service
		err       error
	)

	log.Printf("[account] account=%q, cal=%q, start=%s, end=%s", a.Name, cal.Title, start, end)

	if srv, err = a.Service(); err != nil {
		return nil, a.handleAPIError(err)
	}

	evs, err := srv.Events.List(cal.ID).
		SingleEvents(true).
		MaxResults(2500).
		TimeMin(startTime).
		TimeMax(endTime).
		OrderBy("startTime").Do()

	if err != nil {
		return nil, a.handleAPIError(err)
	}

	for _, e := range evs.Items {
		if e.Start.DateTime == "" { // all-day event
			continue
		}

		var (
			start time.Time
			end   time.Time
			err   error
		)

		if start, err = time.Parse(time.RFC3339, e.Start.DateTime); err != nil {
			log.Printf("[events] ERR: parse start time (%s): %v", e.Start.DateTime, err)
			continue
		}
		if end, err = time.Parse(time.RFC3339, e.End.DateTime); err != nil {
			log.Printf("[events] ERR: parse end time (%s): %v", e.End.DateTime, err)
			continue
		}

		events = append(events, &Event{
			ID:            e.Id,
			IcalUID:       e.ICalUID,
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

// QuickAdd creates a new event in the passed calendar from Account.
func (a *Account) QuickAdd(calendarID string, quick string) error {

	var (
		srv *calendar.Service
		err error
	)

	if srv, err = a.Service(); err != nil {
		return errors.Wrap(err, "create service")
	}

	if _, err = srv.Events.QuickAdd(calendarID, quick).Do(); err != nil {
		return errors.Wrap(err, "create new event error")
	}

	return err
}

// Check for OAuth2 error and  remove tokens if they've expired/been revoked.
func (a *Account) handleAPIError(err error) error {

	if err2, ok := err.(*url.Error); ok {

		if err3, ok := err2.Err.(*oauth2.RetrieveError); ok {
			var resp errorResponse
			if err4 := json.Unmarshal([]byte(err3.Body), &resp); err4 == nil {

				log.Printf("[events] ERR: OAuth: %s (%s)", resp.Name, resp.Description)

				err := errorAuthentication{
					Name:        resp.Name,
					Description: resp.Description,
					Err:         err3,
				}

				if err.Name == "invalid_grant" {

					log.Printf("[account] clearing invalid token for %q", a.Name)

					a.Token = nil
					if err := a.Save(); err != nil {
						log.Printf("[account] ERR: save %q: %v", a.Name, err)
					}
				}

				return err
			}
		}
	}

	return err
}

// for unmarshalling API errors.
type errorResponse struct {
	Name        string `json:"error"`
	Description string `json:"error_description"`
}

type errorAuthentication struct {
	Name        string
	Description string
	Err         error
}

// Error implements error.
func (err errorAuthentication) Error() string {
	return fmt.Sprintf("authentication error: %s (%s)", err.Name, err.Description)
}
