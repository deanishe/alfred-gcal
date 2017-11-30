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
	"strconv"
	"time"

	"github.com/deanishe/awgo"
	"github.com/deanishe/awgo/update"
	"github.com/deanishe/awgo/util"
	"github.com/docopt/docopt-go"
)

const (
	timeFormat     = "2006-01-02"
	timeFormatLong = "Monday, 2 January 2006"
)

var (
	repo      = "deanishe/alfred-gcal"
	helpURL   = "https://github.com/deanishe/alfred-gcal/issues"
	readmeURL = "https://github.com/deanishe/alfred-gcal/blob/master/README.md"
	forumURL  = "https://alfredforum.com"

	usage = `
gcal (events|calendars|toggle) [options] [<query>]

Usage:
    gcal dates [--] [<format>]
    gcal events [--date=<date>] [<query>]
    gcal calendars [<query>]
    gcal toggle <calID>
    gcal update (workflow|calendars|events|icons) [<date>]
    gcal config [<query>]
    gcal clear
    gcal open [--app=<app>] <url>
    gcal server
    gcal -h

Options:
    -a --app <app>     Application to open URLs in.
    -d --date <date>   Date to show events for (format YYYY-MM-DD).
    -h --help          Show this message and exit.
`
	auth          *Authenticator
	wf            *aw.Workflow
	tokenFile     string
	cacheDirIcons string
	useAppleMaps  bool
	schedule      bool

	// Cache ages
	maxAgeCals   = time.Hour * 3
	maxAgeEvents time.Duration

	// CLI args
	query            string
	calendarID       string
	command          Cmd
	dateFormat       string
	updateWhat       string
	openApp          string
	calURL           string
	startTime        time.Time
	scheduleDuration time.Duration
	endTime          time.Time
)

// Cmd is a program sub-command
type Cmd int

// String returns the name of the command
func (c Cmd) String() string {
	commands := map[Cmd]string{
		cmdCalendars:       "calendars",
		cmdClear:           "clear",
		cmdConfig:          "config",
		cmdDates:           "dates",
		cmdEvents:          "events",
		cmdOpen:            "open",
		cmdServer:          "server",
		cmdToggle:          "toggle",
		cmdUpdateCalendars: "updateCalendars",
		cmdUpdateEvents:    "updateEvents",
		cmdUpdateIcons:     "updateIcons",
		cmdUpdateWorkflow:  "updateWorkflow",
	}
	return commands[c]
}

const (
	cmdCalendars Cmd = iota
	cmdClear
	cmdConfig
	cmdDates
	cmdEvents
	cmdOpen
	cmdServer
	cmdToggle
	cmdUpdateCalendars
	cmdUpdateEvents
	cmdUpdateIcons
	cmdUpdateWorkflow
)

func init() {
	wf = aw.New(update.GitHub(repo), aw.HelpURL(helpURL))
	wf.MagicActions.Register(&calendarMagic{})

	tokenFile = filepath.Join(wf.CacheDir(), "gapi-token.json")
	cacheDirIcons = filepath.Join(wf.CacheDir(), "icons")
	util.MustExist(cacheDirIcons)

	auth = NewAuthenticator(tokenFile, []byte(secret))

	v := os.Getenv("APPLE_MAPS")
	if v == "1" || v == "yes" || v == "true" {
		useAppleMaps = true
	}

	n := envInt("SCHEDULE_DAYS", 3)
	scheduleDuration = time.Hour * time.Duration(n*24)
	n = envInt("EVENT_CACHE_MINS", 30)
	maxAgeEvents = time.Minute * time.Duration(n)
}

// Parse command-line flags
func parseFlags() error {
	args, err := docopt.Parse(usage, wf.Args(), true, wf.Version(), false, true)
	if err != nil {
		return err
	}
	// log.Printf("args=%#v", args)

	// Default start and end times
	s := time.Now().In(time.Local).Format(timeFormat)
	startTime, err = time.ParseInLocation(timeFormat, s, time.Local)
	if err != nil {
		return err
	}
	schedule = true

	if args["calendars"] == true {
		command = cmdCalendars
	}
	if args["clear"] == true {
		command = cmdClear
	}
	if args["config"] == true {
		command = cmdConfig
	}
	if args["dates"] == true {
		command = cmdDates
	}
	if args["events"] == true {
		command = cmdEvents
	}
	if args["open"] == true {
		command = cmdOpen
	}
	if args["server"] == true {
		command = cmdServer
	}
	if args["toggle"] == true {
		command = cmdToggle
	}
	if args["update"] == true {
		if args["calendars"] == true {
			command = cmdUpdateCalendars
		}
		if args["events"] == true {
			command = cmdUpdateEvents
		}
		if args["icons"] == true {
			command = cmdUpdateIcons
		}
		if args["workflow"] == true {
			command = cmdUpdateWorkflow
		}
	}

	if s, ok := args["<date>"].(string); ok {
		startTime, err = time.ParseInLocation(timeFormat, s, time.Local)
		if err != nil {
			return err
		}
	}
	if s, ok := args["<query>"].(string); ok {
		query = s
	}
	if s, ok := args["--app"].(string); ok {
		openApp = s
	}
	if s, ok := args["<url>"].(string); ok {
		calURL = s
	}
	if s, ok := args["<calID>"].(string); ok {
		calendarID = s
	}
	if s, ok := args["<format>"].(string); ok {
		dateFormat = s
	}
	if s, ok := args["--date"].(string); ok {
		startTime, err = time.ParseInLocation(timeFormat, s, time.Local)
		if err != nil {
			return err
		}
		schedule = false
	}
	endTime = startTime.Add(time.Hour * 24)
	return nil
}

func run() {
	var err error

	if err := parseFlags(); err != nil {
		wf.FatalError(err)
	}

	log.Printf("command=%v, calendarID=%v, query=%v, startTime=%v, endTime=%v, dateFormat=%v",
		command, calendarID, query, startTime, endTime, dateFormat)

	if !aw.IsRunning("server") {
		cmd := exec.Command("./gcal", "server")
		if err := aw.RunInBackground("server", cmd); err != nil {
			wf.FatalError(err)
		}
	}

	switch command {
	case cmdCalendars:
		err = doListCalendars()
	case cmdClear:
		err = doClear()
	case cmdConfig:
		err = doConfig()
	case cmdDates:
		err = doDates()
	case cmdEvents:
		err = doEvents()
	case cmdOpen:
		err = doOpen()
	case cmdServer:
		err = doStartServer()
	case cmdToggle:
		err = doToggle()
	case cmdUpdateCalendars:
		err = doUpdateCalendars()
	case cmdUpdateEvents:
		err = doUpdateEvents()
	case cmdUpdateIcons:
		err = doUpdateIcons()
	case cmdUpdateWorkflow:
		err = doUpdateWorkflow()
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

func main() {
	wf.Run(run)
}

// Get an environment variable as an int.
func envInt(name string, fallback int) int {
	s := os.Getenv(name)
	if s == "" {
		log.Printf("[ERROR] environment variable \"%s\" isn't set", name)
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("[ERROR] environment variable \"%s\" is not a number: %s", name, s)
		return fallback
	}
	log.Printf("[env] %s=%d", name, n)
	return n
}
