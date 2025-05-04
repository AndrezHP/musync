package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	spotifyApi, err := NewSpotifyApi()
	check(err)

	var userId = spotifyApi.getCurrentUserId()
	var playlists = spotifyApi.getUserPlaylists(userId)
	fmt.Printf("%s", playlists)
}

func makeHttpRequest(method string, api string, endpoint string, token string) []byte {
	client := &http.Client{}
	req, err := http.NewRequest(method, api+endpoint, nil)
	check(err)

	req.Header.Add("Authorization", "Bearer "+token)
	res, err := client.Do(req)
	check(err)

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode > 299 {
		log.Fatalf("Response failed with status code: %d and\nbody: %s\n", res.StatusCode, body)
	}
	check(err)
	return body
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
