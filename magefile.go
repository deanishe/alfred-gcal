// Copyright (c) 2019 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT

// +build mage

package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/deanishe/awgo/util"
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

// Build build workflow in ./build
func Build() error {
	mg.Deps(cleanBuild)
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

// run workflow
func Run() error {
	mg.Deps(Build)
	fmt.Println("running ...")
	return sh.RunWith(info.Env(), buildDir+"/gcal", "-h")
}

// build .alfredworkflow file in ./dist
func Dist() error {
	mg.SerialDeps(Clean, Build)
	p, err := build.Export(buildDir, distDir)
	if err != nil {
		return err
	}
	fmt.Printf("exported %q\n", p)
	return nil
}

// symlink ./build directory to Alfred's workflow directory
func Link() error {
	mg.Deps(Build)

	dir := filepath.Join(info.AlfredWorkflowDir, info.BundleID)
	fmt.Printf("linking %q to %q ...\n", buildDir, dir)
	if err := sh.Rm(dir); err != nil {
		return err
	}
	return build.Symlink(dir, buildDir, true)
}

// clean & download dependencies
func Deps() error {
	mg.Deps(cleanDeps)
	fmt.Println("downloading deps ...")
	return mod("download")
}

// remove build files
func Clean() {
	fmt.Println("cleaning ...")
	mg.Deps(cleanBuild, cleanMage)
}

func cleanDeps() error { return mod("tidy", "-v") }

func cleanDir(name string) error {
	if !util.PathExists(name) {
		return nil
	}

	infos, err := ioutil.ReadDir(name)
	if err != nil {
		return err
	}

	for _, fi := range infos {
		if err := sh.Rm(filepath.Join(name, fi.Name())); err != nil {
			return err
		}
	}
	return nil
}

// remove generated icons
func CleanIcons() error { return cleanDir("./icons") }

func cleanBuild() error { return cleanDir("./build") }
func cleanMage() error  { return sh.Run("mage", "-clean") }
