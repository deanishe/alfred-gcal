// Copyright (c) 2019 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT

package main

import (
	"fmt"
	"log"

	aw "github.com/deanishe/awgo"
)

// Change a setting.
func doSet() error {

	wf.Configure(aw.TextErrors(true))

	log.Printf("[set] key=%q, value=%q", opts.Key, opts.Value)

	switch opts.Key {
	case "maps":
		value := "1"
		if opts.Value == "google" {
			value = "0"
		}
		return wf.Config.Set("APPLE_MAPS", value, true).Do()
	default:
		return fmt.Errorf("unknown config key: %s", opts.Key)
	}
}
