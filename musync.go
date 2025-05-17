package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	spotifyApi, err := NewSpotifyApi()
	check(err)

	var userId = spotifyApi.getCurrentUserId()
	var playlists = spotifyApi.getUserPlaylists(userId)
	fmt.Printf("%s", playlists)
}

func makeHttpRequest(method string, client http.Client, endpoint string) []byte {
	return nil
}

func getBody(response *http.Response) []byte {
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode > 299 {
		log.Fatalf("Response failed with status code: %d and\nbody: %s\n", response.StatusCode, body)
	}
	check(err)
	return body
}

type ClientParams struct {
	ClientId     string
	ClientSecret string
}

func readClientParams(filePath string) ClientParams {
	var clientParams ClientParams
	file, err := os.Open(filePath)
	defer file.Close()
	check(err)

	json.NewDecoder(file).Decode(&clientParams)
	return clientParams
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
