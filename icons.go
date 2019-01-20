//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-26
//

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	aw "github.com/deanishe/awgo"
	"github.com/deanishe/awgo/util"
)

var (
	apiURL = "http://icons.deanishe.net/icon"
	// Static icons
	iconDefault         = &aw.Icon{Value: "icon.png"} // Workflow icon
	iconCalOff          = &aw.Icon{Value: "icons/calendar-off.png"}
	iconCalOn           = &aw.Icon{Value: "icons/calendar-on.png"}
	iconCalToday        = &aw.Icon{Value: "icons/calendar-today.png"}
	iconDay             = &aw.Icon{Value: "icons/day.png"}
	iconDelete          = &aw.Icon{Value: "icons/trash.png"}
	iconDocs            = &aw.Icon{Value: "icons/docs.png"}
	iconIssue           = &aw.Icon{Value: "icons/issue.png"}
	iconHelp            = &aw.Icon{Value: "icons/help.png"}
	iconMap             = &aw.Icon{Value: "icons/map.png"}
	iconNext            = &aw.Icon{Value: "icons/next.png"}
	iconPrevious        = &aw.Icon{Value: "icons/previous.png"}
	iconReload          = &aw.Icon{Value: "icons/reload.png"}
	iconUpdateOK        = &aw.Icon{Value: "icons/update-ok.png"}
	iconUpdateAvailable = &aw.Icon{Value: "icons/update-available.png"}

	// Font & name of dynamic icons
	eventIconFont = "material"
	eventIconName = "calendar"
	mapIconFont   = "elusive"
	mapIconName   = "map-marker"

	// HTTP client
	webClient = &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   60 * time.Second,
				KeepAlive: 60 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   30 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 10 * time.Second,
		},
	}
)

func reloadIcon() *aw.Icon {
	var (
		step    = 15
		max     = (45 / step) - 1
		current = wf.Config.GetInt("RELOAD_PROGRESS", 0)
		next    = current + 1
	)
	if next > max {
		next = 0
	}

	log.Printf("progress: current=%d, next=%d", current, next)

	wf.Var("RELOAD_PROGRESS", fmt.Sprintf("%d", next))

	if current == 0 {
		return iconReload
	}

	return &aw.Icon{Value: fmt.Sprintf("icons/reload-%d.png", current*step)}

	// switch current {
	// case 1:
	// 	return &aw.Icon{Value: "icons/reload-60.png"}
	// case 2:
	// 	return &aw.Icon{Value: "icons/reload-120.png"}
	// case 3:
	// 	return &aw.Icon{Value: "icons/reload-180.png"}
	// case 4:
	// 	return &aw.Icon{Value: "icons/reload-240.png"}
	// case 5:
	// 	return &aw.Icon{Value: "icons/reload-300.png"}
	// default:
	// 	return &aw.Icon{Value: "icons/reload.png"}
	// }
}

// IconConfig is an icon from the server
type IconConfig struct {
	Font   string
	Name   string
	Colour string
}

// Filename returns the filename for the retrieved icon.
func (ic *IconConfig) Filename() string {
	return fmt.Sprintf("%s-%s-%s.png", ic.Font, ic.Name, ic.Colour)
}

// String is a synonym for Filename().
func (ic *IconConfig) String() string { return ic.Filename() }

// IconGenerator fetches and caches icons from the icon server
type IconGenerator struct {
	Dir     string          // Directory to store icons in
	Default *aw.Icon        // Icon to return if requested icon isn't cached yet
	Queue   []*IconConfig   // Icons that need to be downloaded
	icMap   map[string]bool // Map of IconConfig filenames to prevent duplicates in Queue
}

// NewIconGenerator creates an initialised IconGenerator
func NewIconGenerator(dir string, def *aw.Icon) (*IconGenerator, error) {
	util.MustExist(dir)
	g := &IconGenerator{
		Dir:     dir,
		Default: def,
		Queue:   []*IconConfig{},
		icMap:   map[string]bool{},
	}
	if err := g.loadQueue(); err != nil {
		return nil, err
	}
	return g, nil
}

// loadQueue loads the Generator's queue from the queuefile.
func (g *IconGenerator) loadQueue() error {
	p := g.Queuefile()
	data, err := ioutil.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &g.Queue)
}

// Queuefile returns the path to Generators queue file.
func (g *IconGenerator) Queuefile() string { return filepath.Join(g.Dir, "queue.json") }

// Icon returns an *aw.Icon from the cache.
func (g *IconGenerator) Icon(font, name, colour string) *aw.Icon {
	if strings.HasPrefix(colour, "#") {
		colour = colour[1:]
	}

	ic := &IconConfig{font, name, colour}
	p := g.path(ic)

	if util.PathExists(p) {
		return &aw.Icon{Value: p}
	}

	if !g.icMap[ic.Filename()] {
		g.Queue = append(g.Queue, ic)
		log.Printf("queued icon for retrieval: %s", ic)
		g.icMap[ic.Filename()] = true
	}
	return g.Default
}

// path returns the local path to the cached icon.
func (g *IconGenerator) path(ic *IconConfig) string {
	return filepath.Join(g.Dir, ic.Filename())
}

// url returns the API URL of the IconConfig.
func (g *IconGenerator) url(ic *IconConfig) string {
	return fmt.Sprintf("%s/%s/%s/%s", apiURL, ic.Font, ic.Colour, ic.Name)
}

// HasQueue returns true if there are icons queued for retrieval.
func (g *IconGenerator) HasQueue() bool { return len(g.Queue) > 0 }

// Save caches the Generator queue to queuefile.
func (g *IconGenerator) Save() error {
	if !g.HasQueue() {
		return nil
	}
	data, err := json.MarshalIndent(g.Queue, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(g.Queuefile(), data, 0600)
}

// Download retrieves queued icons.
func (g *IconGenerator) Download() error {
	var errs []error
	if !g.HasQueue() {
		log.Print("no icons to download")
		return nil
	}
	for _, ic := range g.Queue {
		URL := g.url(ic)
		path := g.path(ic)
		if util.PathExists(path) {
			continue
		}
		if err := download(URL, path); err != nil {
			log.Printf("couldn't download \"%s\": %v", URL, err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d error(s) downloading icons", len(errs))
	}
	return os.Remove(g.Queuefile())
}

// download saves a URL to a filepath.
func download(URL string, path string) error {
	res, err := openURL(URL)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	n, err := io.Copy(out, res.Body)
	if err != nil {
		return err
	}
	log.Printf("wrote \"%s\" (%d bytes)", path, n)
	return nil
}

// openURL returns an http.Response. It will return an error if the
// HTTP status code > 299.
func openURL(URL string) (*http.Response, error) {
	log.Printf("fetching %s ...", URL)
	res, err := webClient.Get(URL)
	if err != nil {
		return nil, err
	}
	log.Printf("[%d] %s", res.StatusCode, URL)
	if res.StatusCode > 299 {
		res.Body.Close()
		return nil, errors.New(res.Status)
	}
	return res, nil
}
