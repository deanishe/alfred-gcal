//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-25
//

package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

const (
	authServerURL = "localhost:61432"
)

type response struct {
	code string
	err  error
}

// Authenticator creates an authenticated Google API client
type Authenticator struct {
	Secret    []byte
	TokenFile string
	state     string
	client    *http.Client
}

// NewAuthenticator creates a new Authenticator
func NewAuthenticator(tokenFile string, secret []byte) *Authenticator {
	return &Authenticator{Secret: secret, TokenFile: tokenFile}
}

// GetClient returns an authenticated Google API client
func (a *Authenticator) GetClient() (*http.Client, error) {
	if a.client != nil {
		return a.client, nil
	}

	// generate CSRF token
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("couldn't read random bytes: %v", err)
	}
	a.state = fmt.Sprintf("%x", b)

	ctx := context.Background()
	cfg, err := google.ConfigFromJSON(a.Secret, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("couldn't load config: %v", err)
	}

	tok, err := a.tokenFromFile()
	if err != nil {
		tok, err = a.tokenFromWeb(cfg)
		if err != nil {
			return nil, fmt.Errorf("couldn't get token from web: %v", err)
		}
		a.saveToken(tok)
	}

	a.client = cfg.Client(ctx, tok)
	return a.client, nil
}

// tokenFromFile loads the oauth2 token from a file
func (a *Authenticator) tokenFromFile() (*oauth2.Token, error) {
	f, err := os.Open(a.TokenFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't open token file: %v", err)
	}
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	defer f.Close()
	return tok, err
}

// saveToken saves an oauth2 token to a file
func (a *Authenticator) saveToken(tok *oauth2.Token) error {
	f, err := os.OpenFile(a.TokenFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("couldn't open token file: %v", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(tok)
}

// tokenFromWeb initiates web-based authentication and retrieves the oauth2 token
func (a *Authenticator) tokenFromWeb(cfg *oauth2.Config) (*oauth2.Token, error) {
	if err := a.openAuthURL(cfg); err != nil {
		return nil, fmt.Errorf("couldn't open auth URL: %v", err)
	}

	code, err := a.codeFromLocalServer()
	if err != nil {
		return nil, fmt.Errorf("couldn't get token from local server: %v", err)
	}

	tok, err := cfg.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, fmt.Errorf("couldn't retrieve token from web: %v", err)
	}
	return tok, nil
}

// openAuthURL opens the Google API authentication URL in the default browser
func (a *Authenticator) openAuthURL(cfg *oauth2.Config) error {
	authURL := cfg.AuthCodeURL(a.state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	cmd := exec.Command("/usr/bin/open", authURL)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("couldn't open auth URL: %v", err)
	}
	return nil
}

// codeFromLocalServer starts a local webserver to receive the oauth2 token
// from Google
func (a *Authenticator) codeFromLocalServer() (string, error) {
	c := make(chan response)
	srv := &http.Server{Addr: authServerURL}

	go func() {
		log.Printf("local webserver started")
		if err := srv.ListenAndServe(); err != nil {
			c <- response{err: err}
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		vars := req.URL.Query()
		code := vars.Get("code")
		state := vars.Get("state")
		log.Printf("oauth2 state=%v", state)
		log.Printf("oauth2 code=%s", code)

		// Verify state to prevent CSRF
		if state != a.state {
			c <- response{err: fmt.Errorf("state mismatch: expected=%s, got=%s", a.state, state)}
			io.WriteString(w, "bad state\n")
			return
		}

		c <- response{code: code}
		io.WriteString(w, "ok\n")
	})

	r := <-c

	// log.Printf("srv=%+v, response=%+v", srv, r)
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("shutdown error: %v", err)
		if err != http.ErrServerClosed {
			return "", fmt.Errorf("local webserver error: %v", err)
		}
	}
	log.Printf("local webserver stopped")

	return r.code, r.err
}
