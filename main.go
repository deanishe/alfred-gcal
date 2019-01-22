//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-25
//

// Command gcal is an Alfred 3 workflow for viewing Google Calendar events.
package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	aw "github.com/deanishe/awgo"
	"github.com/deanishe/awgo/update"
	"github.com/deanishe/awgo/util"
	docopt "github.com/docopt/docopt-go"
	"github.com/pkg/errors"
)

const (
	timeFormat     = "2006-01-02"
	timeFormatLong = "Monday, 2 January 2006"
)

var (
	repo      = "deanishe/alfred-gcal"
	helpURL   = "https://github.com/deanishe/alfred-gcal/issues"
	readmeURL = "https://github.com/deanishe/alfred-gcal/blob/master/README.md"
	forumURL  = "https://www.alfredforum.com/topic/11016-google-calendar-view/"

	usage = `
gcal [<command>] [options] [<query>]

Usage:
    gcal dates [--] [<format>]
    gcal events [--date=<date>] [--] [<query>]
    gcal calendars [<query>]
    gcal toggle <calID>
    gcal update (workflow|calendars|events) [<date>]
    gcal config [<query>]
    gcal clear
    gcal open [--app=<app>] <url>
    gcal server
    gcal reload
    gcal -h

Options:
    -a --app <app>     Application to open URLs in.
    -d --date <date>   Date to show events for (format YYYY-MM-DD).
    -h --help          Show this message and exit.
`
	auth *Authenticator
	wf   *aw.Workflow

	tokenFile     string // Google credentials
	cacheDirIcons string // directory generated icons are stored in

	useAppleMaps bool
	schedule     bool // show in schedule mode

	// Cache ages
	maxAgeCals   = time.Hour * 3
	maxAgeEvents time.Duration

	// CLI args
	opts             *options
	startTime        time.Time
	scheduleDuration time.Duration
	endTime          time.Time

	// Workflow icon colours
	green  = "03ae03"
	blue   = "5484f3"
	red    = "b00000"
	yellow = "f8ac30"
)

// CLI flags
type options struct {
	// commands
	Calendars bool
	Clear     bool
	Config    bool
	Dates     bool
	Events    bool
	Open      bool
	Reload    bool
	Server    bool
	Toggle    bool
	Update    bool

	// sub-commands
	Workflow bool

	// flags
	App        string
	CalendarID string `docopt:"<calID>"`
	Date       string `docopt:"<date>,--date"`
	DateFormat string `docopt:"<format>"`
	Query      string
	URL        string `docopt:"<url>"`

	// needed to make '--' work
	EndOfOptions bool `docopt:"--"`
}

func init() {

	wf = aw.New(update.GitHub(repo), aw.HelpURL(helpURL))
	wf.MagicActions.Register(&calendarMagic{})

	tokenFile = filepath.Join(wf.CacheDir(), "gapi-token.json")
	cacheDirIcons = filepath.Join(wf.CacheDir(), "icons")
	util.MustExist(cacheDirIcons)

	auth = NewAuthenticator(tokenFile, []byte(secret))

	// Workflow settings from Alfred's configuration sheet.
	useAppleMaps = wf.Config.GetBool("APPLE_MAPS")
	scheduleDuration = time.Hour * time.Duration(wf.Config.GetInt("SCHEDULE_DAYS", 3)*24)
	maxAgeEvents = time.Minute * time.Duration(wf.Config.GetInt("EVENT_CACHE_MINS", 30))
}

// Parse command-line flags.
func parseFlags() error {

	opts = &options{}

	args, err := docopt.ParseArgs(usage, wf.Args(), wf.Version())
	if err != nil {
		return errors.Wrap(err, "docopt parse")
	}

	if err := args.Bind(opts); err != nil {
		return errors.Wrap(err, "docopts bind")
	}

	// We don't need to be fussy about the default start and end times:
	// The default startTime is only used in schedule mode, and it (and endTime)
	// will be set to midnight if user specifies a date.
	startTime = time.Now().Local()
	schedule = true

	if opts.Date != "" {
		startTime, err = time.ParseInLocation(timeFormat, opts.Date, time.Local)
		if err != nil {
			return err
		}
		schedule = false
	}

	endTime = startTime.Add(time.Hour * 24)

	log.Printf("query=%q, startTime=%v, endTime=%v", opts.Query, startTime, endTime)

	return nil
}

// Main program entry point.
func run() {
	var err error

	if err := parseFlags(); err != nil {
		wf.FatalError(err)
	}

	if !wf.IsRunning("server") {
		cmd := exec.Command(os.Args[0], "server")
		if err := wf.RunInBackground("server", cmd); err != nil {
			wf.FatalError(err)
		}
	}

	switch {
	// check for Update first as Calendars and Events are also
	// set by the corresponding top-level commands.
	case opts.Update:
		switch {
		case opts.Calendars:
			err = doUpdateCalendars()
		case opts.Events:
			err = doUpdateEvents()
		case opts.Workflow:
			err = doUpdateWorkflow()
		}
		break
	case opts.Calendars:
		err = doListCalendars()
	case opts.Clear:
		err = doClear()
	case opts.Config:
		err = doConfig()
	case opts.Dates:
		err = doDates()
	case opts.Events:
		err = doEvents()
	case opts.Open:
		err = doOpen()
	case opts.Server:
		err = doStartServer()
	case opts.Toggle:
		err = doToggle()
	case opts.Reload:
		err = doReload()
	}

	if err != nil {
		if err == errNoActive {
			wf.NewItem("No active calendars").
				Subtitle("↩ or ⇥ to choose calendars").
				Autocomplete("workflow:calendars").
				Valid(false).
				Icon(aw.IconWarning)

			wf.SendFeedback()
			return
		}
		wf.FatalError(err)
	}
}

// Call via Workflow.Run() to rescue panics and show an error message
// in Alfred.
func main() {
	wf.Run(run)
}
