//
// Copyright (c) 2017 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2017-11-25
//

package main

// Google doesn't allow app keys to be stored in open-source source code.
// The built workflow includes a key.
//
// If you want to hack on the source code, register your own project here:
// https://console.developers.google.com/apis/dashboard
//
// Add the Google Calendar API and create credentials for a web app, with
// http://localhost:61432 as the redirect URI.
//
// The workflow only requires read access.
const secret = `
{
  "web": {
    "redirect_uris": [
      "http://localhost:61432"
    ],
    "auth_uri": "https://accounts.google.com/o/oauth2/auth",
    "client_id": "",
    "project_id": "",
    "client_secret": "",
    "token_uri": "https://accounts.google.com/o/oauth2/token",
    "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs"
  }
}
`
