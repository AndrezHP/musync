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
	"time"
)

type ApiToken struct {
	AccessToken  string
	TokenType    string
	RefreshToken string
	Expiry       time.Time
}

type OAuthHandler struct {
	Config       *oauth2.Config
	Ctx          context.Context
	TokenChannel chan ApiToken
	ApiUrl       string
	ApiToken     *ApiToken
}

func NewOAuthHandler(config *oauth2.Config, ctx context.Context, tokenChannel chan ApiToken, apiUrl string) (OAuthHandler, error) {
	handler := OAuthHandler{
		config,
		ctx,
		tokenChannel,
		apiUrl,
		nil,
	}
	initToken := handler.getInitToken()
	handler.ApiToken = &initToken
	return handler, nil
}

func (oauth OAuthHandler) callbackHandler(writer http.ResponseWriter, req *http.Request) {
	queryParts, _ := url.ParseQuery(req.URL.RawQuery)
	code := queryParts["code"][0]
	token, err := oauth.Config.Exchange(oauth.Ctx, code)
	check(err)

	oauth.TokenChannel <- ApiToken{
		token.AccessToken,
		token.TokenType,
		token.RefreshToken,
		token.Expiry,
	}

	client := oauth.Config.Client(oauth.Ctx, token)
	resp, err := client.Get(oauth.ApiUrl)
	check(err)

	log.Println("Authentication successful")
	defer resp.Body.Close()

	msg := "<p><strong>Success!</strong></p>"
	msg = msg + "<p>You are authenticated and can now return to the CLI.</p>"
	fmt.Fprintf(writer, msg)
}

func (oauth OAuthHandler) getInitToken() ApiToken {
	go func() {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		tlsClient := &http.Client{Transport: transport}
		oauth.Ctx = context.WithValue(oauth.Ctx, oauth2.HTTPClient, tlsClient)

		url := oauth.Config.AuthCodeURL("state", oauth2.AccessTypeOffline)
		log.Println("You will now be taken to your browser for authentication")
		exec.Command("xdg-open", url).Start()

		http.HandleFunc("/callback", oauth.callbackHandler)
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
	token := <-oauth.TokenChannel
	return token
}

func (oauth OAuthHandler) getAccessToken() string {
	if oauth.ApiToken == nil {
		panic("Token was nil")
	} else if time.Now().Sub(oauth.ApiToken.Expiry) > 0 {
		// TODO Implement token refresh
		return ""
	} else {
		return oauth.ApiToken.AccessToken
	}
}
