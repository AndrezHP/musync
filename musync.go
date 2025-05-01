package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

var api = "https://api.spotify.com/"

func main() {
	var token = getAccessToken()
	fmt.Println("Super cool token: " + token)
	var userId = getCurrentUserId(token)
	var playlists = getUserPlaylists(userId, token)
	fmt.Printf("%s", playlists)
}

func getCurrentUserId(token string) string {
	var _ = makeHttpRequest("GET", api, "v1/me", token)
	// TODO Implement json parsing to get id
	var userId = "1121551335"
	return userId
}

func getUserPlaylists(userId string, token string) string {
	var userPlaylists = fmt.Sprintf("v1/users/%s/playlists", userId)
	return makeHttpRequest("GET", api, userPlaylists, token)
}

func getPlaylistSongs(listId string) string {
	// TODO Implement this
	return "Yeah man"
}

func makeHttpRequest(method string, api string, endpoint string, token string) string {
	client := &http.Client{}
	req, err := http.NewRequest(method, api+endpoint, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", "Bearer "+token)
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode > 299 {
		log.Fatalf("Response failed with status code: %d and\nbody: %s\n", res.StatusCode, body)
	}

	if err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%s", body)
}
