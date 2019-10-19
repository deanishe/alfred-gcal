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
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

const (
	authServerURL  = "localhost:61432"
	userEmailScope = "https://www.googleapis.com/auth/userinfo.email"
)

type response struct {
	code string
	err  error
}

// Authenticator creates an authenticated Google API client
type Authenticator struct {
	Secret  []byte
	Account *Account
	state   string

	client *http.Client
	mu     sync.Mutex

	// set when authentication fails so other goroutines don't
	// repeatedly try to log in
	Failed bool
}

// NewAuthenticator creates a new Authenticator
func NewAuthenticator(acc *Account, secret []byte) *Authenticator {
	return &Authenticator{Account: acc, Secret: secret}
}

// GetClient returns an authenticated Google API client
func (a *Authenticator) GetClient() (*http.Client, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// bail out as previous authentication attempt has failed
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
	cfg, err := google.ConfigFromJSON(a.Secret, calendar.CalendarScope, userEmailScope)
	if err != nil {
		return nil, errors.Wrap(err, "load config")
	}

	var save bool
	if a.Account.Token == nil {
		if err = a.tokenFromWeb(cfg); err != nil {
			a.Failed = true
			return nil, errors.Wrap(err, "token from web")
		}
		a.Account.ReadWrite = cfg.Scopes[0] == calendar.CalendarScope
		save = true
	}

	a.client = cfg.Client(ctx, a.Account.Token)

	// If Account is empty, fetch user info from Google API
	if a.Account.Name == "" {
		if err := a.getUserInfo(); err != nil {
			return nil, err
		}
		save = true
	}

	if save {
		if err = a.Account.Save(); err != nil {
			return nil, errors.Wrap(err, "save account")
		}
	}

	return a.client, nil
}

/*
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
*/

// tokenFromWeb initiates web-based authentication and retrieves the oauth2 token
func (a *Authenticator) tokenFromWeb(cfg *oauth2.Config) error {
	var (
		code  string
		token *oauth2.Token
		err   error
	)

	if err = a.openAuthURL(cfg); err != nil {
		return errors.Wrap(err, "open auth URL")
	}

	if code, err = a.codeFromLocalServer(); err != nil {
		return errors.Wrap(err, "get token from local server")
	}

	if token, err = cfg.Exchange(context.Background(), code); err != nil {
		return errors.Wrap(err, "token from web")
	}

	a.Account.Token = token

	return nil
}

func (a *Authenticator) getUserInfo() error {
	var (
		resp *http.Response
		data []byte
		err  error
	)

	if resp, err = a.client.Get("https://accounts.google.com/.well-known/openid-configuration"); err != nil {
		return fmt.Errorf("get user info: %v", err)
	}
	defer resp.Body.Close()

	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		return fmt.Errorf("read user response: %v", err)
	}

	s := struct {
		Endpoint string `json:"userinfo_endpoint"`
	}{}

	if err = json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parse OpenID JSON: %v", err)
	}

	log.Printf("[auth] fetching user info from %s ...", s.Endpoint)

	if resp, err = a.client.Get(s.Endpoint); err != nil {
		return fmt.Errorf("read userinfo_endpoint: %v", err)
	}
	defer resp.Body.Close()

	if data, err = ioutil.ReadAll(resp.Body); err != nil {
		return fmt.Errorf("read userinfo_endpoint response: %v", err)
	}

	log.Printf("[auth] response=%s", string(data))

	st := struct {
		Email  string `json:"email"`
		Avatar string `json:"picture"`
	}{}

	if err := json.Unmarshal(data, &st); err != nil {
		return errors.Wrap(err, "unmarshal userinfo")
	}

	a.Account.Name = st.Email
	a.Account.Email = st.Email
	a.Account.AvatarURL = st.Avatar

	log.Printf("[auth] fetching user avatar ...")
	if err := download(a.Account.AvatarURL, a.Account.IconPath()); err != nil {
		return errors.Wrap(err, "fetch avatar")
	}

	return nil
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
			if _, err := io.WriteString(w, "bad state\n"); err != nil {
				log.Printf("[error] write server response: %v", err)
			}
			return
		}

		// authentication failed
		if errMsg != "" {
			c <- response{err: errors.New(errMsg)}
			if _, err := io.WriteString(w, errMsg+"\n"); err != nil {
				log.Printf("[error] write server response: %v", err)
			}
			return
		}

		// user rejected
		if code == "" {
			c <- response{err: errors.New("user rejected access")}
			if _, err := io.WriteString(w, "access denied by user\n"); err != nil {
				log.Printf("[error] write server response: %v", err)
			}
			return
		}

		c <- response{code: code}
		if _, err := io.WriteString(w, "ok\n"); err != nil {
			log.Printf("[error] write server response: %v", err)
		}
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
