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

	// Workflow icon colours
	yellow = "f8ac30"
	// green  = "03ae03"
	// blue   = "5484f3"
	// red    = "b00000"

	// Workflow settings & URLs
	repo      = "deanishe/alfred-gcal"
	helpURL   = "https://github.com/deanishe/alfred-gcal/issues"
	readmeURL = "https://github.com/deanishe/alfred-gcal/blob/master/README.md"
	forumURL  = "https://www.alfredforum.com/topic/11016-google-calendar-view/"
)

const usage = `
gcal [<command>] [options] [<query>]

Usage:
    gcal dates [--] [<format>]
    gcal events [--date=<date>] [--] [<query>]
    gcal calendars [<query>]
    gcal active [<query>]
    gcal toggle <calID>
    gcal set <key> <value>
    gcal update (workflow|calendars|events) [<date>]
    gcal config [<query>]
    gcal logout <account>
    gcal reauth <account>
    gcal clear
    gcal open [--app=<app>] <url>
    gcal server
    gcal reload
    gcal create <quick> <calID>
    gcal -h

Options:
    -a --app <app>     Application to open URLs in.
    -d --date <date>   Date to show events for (format YYYY-MM-DD).
    -h --help          Show this message and exit.
    --version          Show workflow version and exit.
`

var (
	wf       *aw.Workflow
	accounts []*Account

	cacheDirIcons string // directory generated icons are stored in

	// CLI args
	opts *options

	// display times using 24h clock
	hourFormat = "15:04"
)

// CLI flags
type options struct {
	// commands
	Calendars bool
	Active    bool
	Clear     bool
	Config    bool
	Dates     bool
	Events    bool
	Logout    bool
	Reauth    bool
	Open      bool
	Reload    bool
	Server    bool
	Set       bool
	Toggle    bool
	Update    bool
	Create    bool

	// sub-commands
	Workflow bool

	// flags
	Account    string
	App        string
	CalendarID string `docopt:"<calID>"`
	Date       string `docopt:"<date>,--date"`
	DateFormat string `docopt:"<format>"`
	Query      string
	URL        string `docopt:"<url>"`
	Key        string
	Value      string
	Quick      string `docopt:"<quick>"`

	// options
	UseAppleMaps   bool `env:"APPLE_MAPS"`
	EventCacheMins int  `env:"EVENT_CACHE_MINS"`
	ScheduleDays   int  `env:"SCHEDULE_DAYS"`
	Use12HourTime  bool `env:"TIME_12H"`
	ScheduleMode   bool
	StartTime      time.Time
	EndTime        time.Time

	// needed to make '--' work
	EndOfOptions bool `docopt:"--"`
}

func (opts *options) MaxAgeCalendar() time.Duration { return time.Hour * 3 }

func (opts *options) MaxAgeEvents() time.Duration {
	d := time.Duration(opts.EventCacheMins) * time.Minute
	if d < time.Minute*5 {
		d = time.Minute * 5
	}

	return d
}

func (opts *options) ScheduleDuration() time.Duration {
	return time.Duration(opts.ScheduleDays) * time.Hour * 24
}

func init() {
	opts = &options{}

	wf = aw.New(update.GitHub(repo), aw.HelpURL(helpURL))
	wf.Configure(aw.AddMagic(&calendarMagic{}, &loginMagic{}))

	cacheDirIcons = filepath.Join(wf.CacheDir(), "icons")
}

// Parse command-line flags.
func parseFlags() error {
	args, err := docopt.ParseArgs(usage, wf.Args(), wf.Version())
	if err != nil {
		return errors.Wrap(err, "docopt parse")
	}

	if err := args.Bind(opts); err != nil {
		return errors.Wrap(err, "bind docopt")
	}

	if err := wf.Config.To(opts); err != nil {
		return errors.Wrap(err, "bind config")
	}

	// We don't need to be fussy about the default start and end times:
	// The default startTime is only used in schedule mode, and it (and endTime)
	// will be set to midnight if user specifies a date.
	opts.StartTime = time.Now().Local()
	opts.ScheduleMode = true

	if opts.Date != "" {
		opts.StartTime, err = time.ParseInLocation(timeFormat, opts.Date, time.Local)
		if err != nil {
			return err
		}
		opts.ScheduleMode = false
	}

	if opts.Use12HourTime {
		hourFormat = "3:04"
	}

	opts.EndTime = opts.StartTime.Add(time.Hour * 24)

	log.Printf("[main] query=%q, startTime=%v, endTime=%v",
		opts.Query, opts.StartTime, opts.EndTime)

	return nil
}

// Main program entry point.
func run() {
	var err error

	if err = parseFlags(); err != nil {
		wf.FatalError(err)
	}

	// Ensure required directories exist
	util.MustExist(cacheDirIcons)

	if accounts, err = LoadAccounts(); err != nil {
		panic(err)
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
	case opts.Logout:
		err = doLogout()
	case opts.Open:
		err = doOpen()
	case opts.Set:
		err = doSet()
	case opts.Server:
		err = doStartServer()
	case opts.Toggle:
		err = doToggle()
	case opts.Reauth:
		err = doReauth()
	case opts.Reload:
		err = doReload()
	case opts.Create:
		err = quickAdd()
	case opts.Active:
		err = doListWritableCalendars()
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

// Call via Workflow.Run() to rescue panics and show an error message in Alfred.
func main() {
	wf.Run(run)
}
