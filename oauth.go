package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"net/url"
	"os/exec"
)

var (
	conf        *oauth2.Config
	ctx         context.Context
	tokenOutput = make(chan string)
)

func callbackHandler(writer http.ResponseWriter, req *http.Request) {
	queryParts, _ := url.ParseQuery(req.URL.RawQuery)
	code := queryParts["code"][0]
	token, err := conf.Exchange(ctx, code)
	if err != nil {
		log.Fatal(err)
	}
	tokenOutput <- token.AccessToken

	client := conf.Client(ctx, token)
	resp, err := client.Get("https://api.spotify.com/")
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Authentication successful")
	}
	defer resp.Body.Close()

	msg := "<p><strong>Success!</strong></p>"
	msg = msg + "<p>You are authenticated and can now return to the CLI.</p>"
	fmt.Fprintf(writer, msg)
}

func fetchAccessToken() {
	ctx = context.Background()
	conf = &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Scopes:       []string{"user-read-private", "user-read-email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
		RedirectURL: "http://localhost:8080/callback",
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	tlsClient := &http.Client{Transport: transport}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, tlsClient)

	url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	log.Println("You will now be taken to your browser for authentication")
	exec.Command("xdg-open", url).Start()

	http.HandleFunc("/callback", callbackHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getAccessToken() string {
	go fetchAccessToken()
	token := <-tokenOutput
	return token
}
