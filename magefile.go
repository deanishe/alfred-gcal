// Copyright (c) 2019 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT

// +build mage

package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/bmatcuk/doublestar"
	"github.com/magefile/mage/mg" // mg contains helpful utility functions, like Deps
	"github.com/magefile/mage/sh"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Default target to run when none is specified
// If not set, running mage will list available targets
// var Default = Build

var (
	mod     = sh.RunCmd("go", "mod")
	workDir string
)

func init() {
	var err error
	if workDir, err = os.Getwd(); err != nil {
		panic(err)
	}
}

// Aliases are mage command aliases.
var Aliases = map[string]interface{}{
	"b": Build,
	"d": Dist,
	"l": Link,
}

// Build builds workflow in ./build
func Build() error {
	mg.Deps(cleanBuild)
	// mg.Deps(Deps)
	fmt.Println("building ...")
	if err := sh.Run("mv", "-v", "secret.go", "secret.go.tmp"); err != nil {
		return err
	}
	if err := sh.Run("mv", "-v", "secret.go.private", "secret.go"); err != nil {
		return err
	}

	defer func() {
		if err := sh.Run("mv", "-v", "secret.go", "secret.go.private"); err != nil {
			fmt.Printf("ERR: %v\n", err)
		}
		if err := sh.Run("mv", "-v", "secret.go.tmp", "secret.go"); err != nil {
			fmt.Printf("ERR: %v\n", err)
		}
	}()

	if err := sh.Run("go", "build", "-o", "./build/gcal", "."); err != nil {
		return err
	}

	// link files to ./build
	globs := []struct {
		glob, dest string
	}{
		// {"../ical", ""},
		{"*.png", ""},
		// {"../mask.png", ""},
		{"info.plist", ""},
		{"*.html", ""},
		{"README.md", ""},
		{"LICENCE.txt", ""},
		{"icons/*.png", ""},
	}

	pairs := []struct {
		src, dest string
	}{}

	for _, cfg := range globs {
		files, err := doublestar.Glob(cfg.glob)
		if err != nil {
			return err
		}

		for _, p := range files {
			dest := filepath.Join("./build", cfg.dest, p)
			pairs = append(pairs, struct{ src, dest string }{p, dest})
		}
	}

	for _, p := range pairs {

		var (
			relPath string
			dir     = filepath.Dir(p.dest)
			err     error
		)

		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		if relPath, err = filepath.Rel(filepath.Dir(p.dest), p.src); err != nil {
			return err
		}
		fmt.Printf("%s  -->  %s\n", p.dest, relPath)
		if err := os.Symlink(relPath, p.dest); err != nil {
			return err
		}
	}

	return nil
}

// Run run workflow
func Run() error {
	mg.Deps(Build)
	fmt.Println("running ...")
	if err := os.Chdir("./build"); err != nil {
		return err
	}
	defer os.Chdir(workDir)

	return sh.RunWith(alfredEnv(), "./gcal", "-h")
}

// Dist build an .alfred-workflow file in ./dist
func Dist() error {
	mg.SerialDeps(Clean, Build)
	if err := os.MkdirAll("./dist", 0700); err != nil {
		return err
	}

	var (
		name = slugify(fmt.Sprintf("%s-%s.alfredworkflow", Name, Version))
		path = filepath.Join("./dist", name)
		f    *os.File
		w    *zip.Writer
		err  error
	)

	fmt.Println("building .alfredworkflow file ...")

	if _, err = os.Stat(path); err == nil {
		if err = os.Remove(path); err != nil {
			return err
		}
		fmt.Println("deleted old .alfredworkflow file")
	}

	if f, err = os.Create(path); err != nil {
		return err
	}
	defer f.Close()

	w = zip.NewWriter(f)

	err = filepath.Walk("./build", func(path string, fi os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		var name string
		if name, err = filepath.Rel("./build", path); err != nil {
			return err
		}

		fmt.Printf("    %s\n", name)

		var (
			f  *os.File
			zf io.Writer
		)
		if f, err = os.Open(path); err != nil {
			return err
		}
		defer f.Close()

		if zf, err = w.Create(name); err != nil {
			return err
		}
		if _, err = io.Copy(zf, f); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	if err = w.Close(); err != nil {
		return err
	}

	fmt.Printf("wrote %s\n", path)

	return nil
}

var (
	rxAlphaNum  = regexp.MustCompile(`[^a-zA-Z0-9.-]+`)
	rxMultiDash = regexp.MustCompile(`-+`)
)

// make s filesystem- and URL-safe.
func slugify(s string) string {
	s = fold(s)
	s = rxAlphaNum.ReplaceAllString(s, "-")
	s = rxMultiDash.ReplaceAllString(s, "-")
	return s
}

var stripper = transform.Chain(norm.NFD, transform.RemoveFunc(isMn))

// isMn returns true if rune is a non-spacing mark
func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: non-spacing mark
}

// fold strips diacritics from string.
func fold(s string) string {
	ascii, _, err := transform.String(stripper, s)
	if err != nil {
		panic(err)
	}
	return ascii
}

// Link symlinks ./build directory to Alfred's workflow directory.
func Link() error {
	mg.Deps(Build)

	fmt.Println("linking ./build to workflow directory ...")
	target := filepath.Join(workflowDirectory(), BundleID)
	// fmt.Printf("target: %s\n", target)

	if exists(target) {
		fmt.Println("removing existing workflow ...")
		if err := os.RemoveAll(target); err != nil {
			return err
		}
	}

	build, err := filepath.Abs("build")
	if err != nil {
		return err
	}
	src, err := filepath.Rel(filepath.Dir(target), build)
	if err != nil {
		return err
	}

	if err := os.Symlink(src, target); err != nil {
		return err
	}

	fmt.Printf("symlinked workflow to %s\n", target)

	return nil
}

type iconCfg struct {
	name, colour, font, icon string
}

// IconsReplace download all icons, replacing any existing ones
func IconsReplace() {
	mg.SerialDeps(cleanIcons, Icons)
}

// Icons download workflow icons
func Icons() error {

	var (
		api   = "http://icons.deanishe.net/icon"
		f     *os.File
		icons []iconCfg
		err   error
	)

	if f, err = os.Open("./icons/icons.txt"); err != nil {
		return err
	}
	defer f.Close()

	var (
		n   int
		scn = bufio.NewScanner(f)
	)

	for scn.Scan() {
		n++
		s := scn.Text()
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}

		row := strings.Fields(s)
		if len(row) != 4 {
			return fmt.Errorf("line %d: need 4 fields not %d: %s", n, len(row), s)
		}

		icons = append(icons, iconCfg{name: row[0] + ".png", font: row[1], colour: row[2], icon: row[3]})
	}
	if err = scn.Err(); err != nil {
		return err
	}

	for i, icon := range icons {
		p := filepath.Join("./icons", icon.name)
		if exists(p) {
			fmt.Printf("[%d/%d] skipped existing: %s\n", i+1, len(icons), icon.name)
			continue
		}

		fmt.Printf("[%d/%d] %s ...\n", i+1, len(icons), icon.name)
		u := fmt.Sprintf("%s/%s/%s/%s", api, icon.font, icon.colour, icon.icon)
		if err := download(u, p); err != nil {
			return err
		}
	}

	return nil
}

var client = http.Client{
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

func download(URL, path string) error {

	r, err := client.Get(URL)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	fmt.Printf("[%d] %s\n", r.StatusCode, URL)
	if r.StatusCode > 299 {
		return fmt.Errorf("bad HTTP response: [%d] %s", r.StatusCode, URL)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, r.Body); err != nil {
		return err
	}

	fmt.Printf("wrote %s\n", path)

	return nil
}

// Deps ensure dependencies
func Deps() error {
	fmt.Println("installing deps ...")
	return mod("download")
}

// Clean remove build files
func Clean() {
	fmt.Println("cleaning ...")
	mg.Deps(cleanBuild, cleanMage)
}

func cleanDeps() error {
	return mod("tidy", "-v")
}

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

func cleanBuild() error {
	return cleanDir("./build")
}

func cleanMage() error {
	return sh.Run("mage", "-clean")
}

func cleanIcons() error {
	return cleanDir("./icons", "*.txt")
}
