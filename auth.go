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
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"

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

	client *http.Client
	mu     sync.Mutex

	Failed bool
}

// NewAuthenticator creates a new Authenticator
func NewAuthenticator(tokenFile string, secret []byte) *Authenticator {
	return &Authenticator{Secret: secret, TokenFile: tokenFile}
}

// GetClient returns an authenticated Google API client
func (a *Authenticator) GetClient() (*http.Client, error) {

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.Failed {
		return nil, errors.New("authentication failed")
	}

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
		return nil, fmt.Errorf("load config: %v", err)
	}

	tok, err := a.tokenFromFile()
	if err != nil {
		tok, err = a.tokenFromWeb(cfg)
		if err != nil {
			a.Failed = true
			return nil, fmt.Errorf("token from web: %v", err)
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
		return nil, fmt.Errorf("open token file: %v", err)
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
		return fmt.Errorf("open token file: %v", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(tok)
}

// tokenFromWeb initiates web-based authentication and retrieves the oauth2 token
func (a *Authenticator) tokenFromWeb(cfg *oauth2.Config) (*oauth2.Token, error) {
	if err := a.openAuthURL(cfg); err != nil {
		return nil, fmt.Errorf("open auth URL: %v", err)
	}

	code, err := a.codeFromLocalServer()
	if err != nil {
		return nil, fmt.Errorf("get token from local server: %v", err)
	}

	tok, err := cfg.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, fmt.Errorf("token from web: %v", err)
	}
	return tok, nil
}

// openAuthURL opens the Google API authentication URL in the default browser
func (a *Authenticator) openAuthURL(cfg *oauth2.Config) error {
	authURL := cfg.AuthCodeURL(a.state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	cmd := exec.Command("/usr/bin/open", authURL)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open auth URL: %v", err)
	}
	return nil
}

// codeFromLocalServer starts a local webserver to receive the oauth2 token
// from Google
func (a *Authenticator) codeFromLocalServer() (string, error) {

	var (
		c   = make(chan response)
		mux = http.NewServeMux()
		srv = &http.Server{
			Addr:    authServerURL,
			Handler: mux,
		}
	)

	go func() {
		log.Printf("[auth] starting local webserver on %s ...", authServerURL)
		if err := srv.ListenAndServe(); err != nil {
			c <- response{err: err}
		}
	}()

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		vars := req.URL.Query()
		code := vars.Get("code")
		state := vars.Get("state")
		errMsg := vars.Get("error")
		log.Printf("[auth] oauth2 state=%v", state)
		log.Printf("[auth] oauth2 code=%s", code)
		log.Printf("[auth] oauth2 error=%s", errMsg)

		// Verify state to prevent CSRF
		if state != a.state {
			c <- response{err: fmt.Errorf("state mismatch: expected=%s, got=%s", a.state, state)}
			io.WriteString(w, "bad state\n")
			return
		}

		// authentication failed
		if errMsg != "" {
			c <- response{err: errors.New(errMsg)}
			io.WriteString(w, errMsg+"\n")
			return
		}

		// user rejected
		if code == "" {
			c <- response{err: errors.New("user rejected access")}
			io.WriteString(w, "access denied by user\n")
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
			return "", fmt.Errorf("auth webserver: %v", err)
		}
	}

	log.Printf("[auth] local webserver stopped")

	return r.code, r.err
}
