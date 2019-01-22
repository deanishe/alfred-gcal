//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-26
//

package main

import (
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	aw "github.com/deanishe/awgo"
	"github.com/deanishe/awgo/util"
	"github.com/pkg/errors"
)

var (
	apiURL = "http://icons.deanishe.net/icon"
	// Static icons
	iconDefault         = &aw.Icon{Value: "icon.png"} // Workflow icon
	iconCalendar        = &aw.Icon{Value: "icons/calendar.png"}
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
	iconLoading         = &aw.Icon{Value: "icons/loading.png"}
	iconUpdateOK        = &aw.Icon{Value: "icons/update-ok.png"}
	iconUpdateAvailable = &aw.Icon{Value: "icons/update-available.png"}
	iconWarning         = &aw.Icon{Value: "icons/warning.png"}

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

func init() {
	aw.IconWarning = iconWarning
}

// ColouredIcon returns a version of icon in the given colour. If no colour
// is specified or something goes wrong, icon is simply returned.
func ColouredIcon(icon *aw.Icon, colour string) *aw.Icon {

	var (
		c    color.RGBA
		path string
		err  error
	)

	if c, err = ParseHexColour(colour); err != nil {
		log.Printf("[icons] ERR: %s", err)
		return icon
	}

	path = iconCachePath(icon, c)

	if util.PathExists(path) {
		return &aw.Icon{Value: path}
	}

	if err = generateIcon(icon.Value, path, c); err != nil {
		log.Printf("[icons] ERR: generate icon: %v", err)
		return icon
	}

	return &aw.Icon{Value: path}
}

func generateIcon(src, dest string, c color.RGBA) error {

	// defer util.Timed(time.Now(), "generate icon")

	var (
		f    *os.File
		mask image.Image
		err  error
	)

	if f, err = os.Open(src); err != nil {
		return errors.Wrap(err, "open file")
	}
	defer f.Close()

	if mask, _, err = image.Decode(f); err != nil {
		return errors.Wrap(err, "decode image")
	}

	img := image.NewRGBA(mask.Bounds())
	draw.DrawMask(img, img.Bounds(), &image.Uniform{c}, image.ZP, mask, image.ZP, draw.Src)

	if f, err = os.Create(dest); err != nil {
		return errors.Wrap(err, "create file")
	}
	defer f.Close()

	if err = png.Encode(f, img); err != nil {
		return errors.Wrap(err, "write PNG data")
	}

	rel, _ := filepath.Rel(wf.CacheDir(), dest)
	log.Printf("[icons] new icon: %s", rel)

	return nil
}

func iconCachePath(i *aw.Icon, c color.RGBA) string {
	name := filepath.Base(i.Value)
	dir := fmt.Sprintf("%02X/%02X/%02X/%02X", uint8(c.R), uint8(c.G), uint8(c.B), uint8(c.A))
	dir = filepath.Join(cacheDirIcons, dir)

	util.MustExist(dir)

	return filepath.Join(dir, name)
}

// ParseHexColour parses a CSS hex (e.g. #ffffff) to RGBA.
//
// Input must be with preceding # and have 6 (RGB) or 8 (RGBA) characters.
func ParseHexColour(s string) (color.RGBA, error) {

	var (
		// default to 100% opaque
		c   = color.RGBA{A: 0xff}
		err error
	)

	if s[0] == '#' {
		s = s[1:]
	}

	b, err := hex.DecodeString(s)
	if err != nil {
		return c, fmt.Errorf("invalid colour: %q", s)
	}

	switch len(b) {
	case 3:
		c.R = b[0]
		c.G = b[1]
		c.B = b[2]
	case 4:
		c.R = b[0]
		c.G = b[1]
		c.B = b[2]
		c.A = b[3]
	default:
		err = fmt.Errorf("invalid colour: %q", s)
	}

	return c, err
}

// ReloadIcon returns a spinner icon. It rotates by 15 deg on every
// subsequent call. Use with wf.Reload(0.1) to implement an animated
// spinner.
func ReloadIcon() *aw.Icon {
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
		return iconLoading
	}

	return &aw.Icon{Value: fmt.Sprintf("icons/loading-%d.png", current*step)}
}
