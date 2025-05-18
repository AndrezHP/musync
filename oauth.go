package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
)

type OAuthHandler struct {
	Config       *oauth2.Config
	Ctx          context.Context
	TokenChannel chan *oauth2.Token
	ApiUrl       string
	Port         string
	VerifierCode string
}

func getToken(config *oauth2.Config, ctx context.Context, apiUrl string, tokenPath string, port string) *oauth2.Token {
	_, err := os.Stat(tokenPath)
	tokenFileExists := !errors.Is(err, os.ErrNotExist)
	var token *oauth2.Token
	if tokenFileExists {
		token, err = readTokenFromFile(tokenPath)
		check(err)
	} else {
		verifierCode := oauth2.GenerateVerifier()
		handler := OAuthHandler{
			config,
			ctx,
			make(chan *oauth2.Token),
			apiUrl,
			port,
			verifierCode,
		}
		token = handler.getInitToken()
		saveTokenToFile(token, tokenPath)
	}
	return token
}

func (oauth OAuthHandler) callbackHandler(writer http.ResponseWriter, req *http.Request) {
	queryParts, _ := url.ParseQuery(req.URL.RawQuery)
	code := queryParts["code"][0]
	token, err := oauth.Config.Exchange(oauth.Ctx, code, oauth2.SetAuthURLParam("code_verifier", oauth.VerifierCode))
	check(err)

	oauth.TokenChannel <- token

	client := oauth.Config.Client(oauth.Ctx, token)
	resp, err := client.Get(oauth.ApiUrl)
	check(err)

	log.Println("Authentication successful")
	defer resp.Body.Close()

	msg := "<p><strong>Success!</strong></p>"
	msg = msg + "<p>You are authenticated and can now return to the CLI.</p>"
	fmt.Fprintf(writer, msg)
}

func (oauth OAuthHandler) getInitToken() *oauth2.Token {
	go func() {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		tlsClient := &http.Client{Transport: transport}
		oauth.Ctx = context.WithValue(oauth.Ctx, oauth2.HTTPClient, tlsClient)

		url := oauth.Config.AuthCodeURL("state", oauth2.S256ChallengeOption(oauth.VerifierCode))

		log.Println("You will now be taken to your browser for authentication")
		exec.Command("xdg-open", url).Start()

		http.HandleFunc("/callback", oauth.callbackHandler)

		log.Fatal(http.ListenAndServe(":"+oauth.Port, nil))
	}()
	token := <-oauth.TokenChannel
	return token
}

func readTokenFromFile(filePath string) (*oauth2.Token, error) {
	var token oauth2.Token
	file, err := os.Open(filePath)
	defer file.Close()
	check(err)

	json.NewDecoder(file).Decode(&token)
	return &token, err
}

func saveTokenToFile(token *oauth2.Token, filePath string) error {
	file, err := os.Create(filePath)
	defer file.Close()
	json.NewEncoder(file).Encode(*token)
	return err
}
