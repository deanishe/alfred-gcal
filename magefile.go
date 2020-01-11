// Copyright (c) 2019 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT

// +build mage

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar"
	"github.com/deanishe/awgo/util/build"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Default target to run when none is specified
// If not set, running mage will list available targets
// var Default = Build

var (
	info     *build.Info
	buildDir = "./build"
	distDir  = "./dist"
	env      map[string]string
)

func init() {
	var err error
	if info, err = build.NewInfo(); err != nil {
		panic(err)
	}
	env = info.Env()
	env["TAGS"] = "private"
}

func mod(args ...string) error {
	argv := append([]string{"mod"}, args...)
	return sh.RunWith(info.Env(), "go", argv...)
}

// Aliases are mage command aliases.
var Aliases = map[string]interface{}{
	"b": Build,
	"c": Clean,
	"d": Dist,
	"l": Link,
}

// Build builds workflow in ./build
func Build() error {
	mg.Deps(cleanBuild, Icons)
	// mg.Deps(Deps)
	fmt.Println("building ...")

	if err := sh.RunWith(env, "go", "build", "-tags", "$TAGS", "-o", buildDir+"/gcal", "."); err != nil {
		return err
	}

	globs := build.Globs(
		"*.png",
		"info.plist",
		"*.html",
		"README.md",
		"LICENCE.txt",
		"icons/*.png",
	)

	return build.SymlinkGlobs(buildDir, globs...)
}

// Run run workflow
func Run() error {
	mg.Deps(Build)
	fmt.Println("running ...")
	return sh.RunWith(info.Env(), buildDir+"/gcal", "-h")
}

// Dist build an .alfredworkflow file in ./dist
func Dist() error {
	mg.SerialDeps(Clean, Build)
	p, err := build.Export(buildDir, distDir)
	if err != nil {
		return err
	}
	fmt.Printf("exported %q\n", p)
	return nil
}

// Link symlinks ./build directory to Alfred's workflow directory.
func Link() error {
	mg.Deps(Build)

	dir := filepath.Join(info.AlfredWorkflowDir, info.BundleID)
	fmt.Printf("linking %q to %q ...\n", buildDir, dir)
	return build.Symlink(dir, buildDir, true)
}

// Icons generate icons
func Icons() error {
	var (
		green = "03ae03"
		blue  = "5484f3"
		// red    = "b00000"
		// yellow = "f8ac30"
	)

	copies := []struct {
		src, dest, colour string
	}{
		{"calendar.png", "icon.png", blue},
		{"calendar.png", "calendars.png", blue},
		{"calendar.png", "calendar-today.png", green},
		{"docs.png", "help.png", green},
	}

	for i, cfg := range copies {
		var (
			src  = filepath.Join("icons", cfg.src)
			dest = filepath.Join("icons", cfg.dest)
		)

		if exists(dest) {
			fmt.Printf("[%d/%d] skipped existing: %s\n", i+1, len(copies), dest)
			continue
		}

		if err := copyImage(src, dest, cfg.colour); err != nil {
			return err
		}
	}

	return rotateIcon("./icons/loading.png", []int{15, 30})
}

// Deps ensure dependencies
func Deps() error {
	mg.Deps(cleanDeps)
	fmt.Println("downloading deps ...")
	return mod("download")
}

// Clean remove build files
func Clean() {
	fmt.Println("cleaning ...")
	mg.Deps(cleanBuild, cleanMage)
}

func cleanDeps() error { return mod("tidy", "-v") }

func cleanDir(name string, exclude ...string) error {
	if _, err := os.Stat(name); err != nil {
		return nil
	}

	infos, err := ioutil.ReadDir(name)
	if err != nil {
		return err
	}

	for _, fi := range infos {
		var match bool
		for _, glob := range exclude {
			if match, err = doublestar.Match(glob, fi.Name()); err != nil {
				return err
			} else if match {
				break
			}
		}

		if match {
			fmt.Printf("excluded: %s\n", fi.Name())
			continue
		}

		p := filepath.Join(name, fi.Name())
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}

func cleanBuild() error { return cleanDir("./build") }
func CleanIcons() error { return cleanDir("./icons") }
func cleanMage() error  { return sh.Run("mage", "-clean") }

// expand ~ and variables in path.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		path = "${HOME}" + path[1:]
	}

	return os.ExpandEnv(path)
}

func exists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		panic(err)
	}

	return true
}
