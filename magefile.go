// Copyright (c) 2019 Dean Jackson <deanishe@deanishe.net>
// MIT Licence applies http://opensource.org/licenses/MIT

// +build mage

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"unicode"

	"github.com/bmatcuk/doublestar"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Default target to run when none is specified
// If not set, running mage will list available targets
// var Default = Build

var (
	workDir string
)

func init() {
	var err error
	if workDir, err = os.Getwd(); err != nil {
		panic(err)
	}
}

func mod(args ...string) error {
	argv := append([]string{"mod"}, args...)
	return sh.RunWith(alfredEnv(), "go", argv...)
}

// Aliases are mage command aliases.
var Aliases = map[string]interface{}{
	"b": Build,
	"d": Dist,
	"l": Link,
}

// Build builds workflow in ./build
func Build() error {
	mg.Deps(cleanBuild, Icons)
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

	if err := sh.RunWith(alfredEnv(), "go", "build", "-o", "./build/gcal", "."); err != nil {
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

		fmt.Printf("    %s (%v)\n", name, fi.Mode().Perm())

		var (
			f  *os.File
			zf io.Writer
			fh *zip.FileHeader
		)

		fh = &zip.FileHeader{
			Name:   name,
			Method: zip.Deflate,
		}

		// fh.SetModTime(fi.ModTime())
		fh.SetMode(fi.Mode().Perm())

		if f, err = os.Open(path); err != nil {
			return err
		}
		defer f.Close()

		if zf, err = w.CreateHeader(fh); err != nil {
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
	}
	// try to remove it anyway, as dangling symlinks register as existing
	if err := os.RemoveAll(target); err != nil && !os.IsNotExist(err) {
		return err
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

		src := filepath.Join("icons", cfg.src)
		dest := filepath.Join("icons", cfg.dest)

		if exists(dest) {
			fmt.Printf("[%d/%d] skipped existing: %s\n", i+1, len(copies), dest)
			continue
		}

		if err := copyImage(src, dest, cfg.colour); err != nil {
			return err
		}
	}

	return rotateIcon("./icons/loading.png", []int{15, 30})

	return nil
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

// CleanIcons delete all generated icons from ./icons
func CleanIcons() error {
	return cleanDir("./icons")
}
