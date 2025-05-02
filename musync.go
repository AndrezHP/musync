package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const api = "https://api.spotify.com/"

func main() {
	var token = getAccessToken()
	var userId = getCurrentUserId(token)
	var playlists = getUserPlaylists(userId, token)
	fmt.Printf("%s", playlists)
}

func getCurrentUserId(token string) string {
	var res = makeHttpRequest("GET", api, "v1/me", token)

	var result map[string]interface{}
	err := json.Unmarshal(res, &result)
	if err != nil {
		log.Fatal(err)
	}

	if id, ok := result["id"]; ok {
		return fmt.Sprint(id)
	} else {
		return "Oh no"
	}
}

func handlePlaylist(playlist Playlist) {
	// TODO create playlist on other service and add songs
}

type Playlist struct {
	Id     string
	Name   string
	Length int
}

type Song struct {
	Name   string
	Id     string
	Artist string
	Album  string
}

type Track struct {
	href  string
	total int
}

func getUserPlaylists(userId string, token string) []Playlist {
	var userPlaylists = fmt.Sprintf("v1/users/%s/playlists", userId)
	var res = makeHttpRequest("GET", api, userPlaylists, token)

	var result map[string]interface{}
	err := json.Unmarshal(res, &result)
	if err != nil {
		log.Fatal(err)
	}

	var playlists []Playlist
	if lists, ok := result["items"].([]interface{}); ok && len(lists) > 0 {
		fmt.Println(len(lists))
		for i := 0; i < len(lists); i++ {
			if item, ok := lists[i].(map[string]interface{}); ok {
				name, _ := item["name"].(string)
				id, _ := item["id"].(string)

				tracks, _ := item["tracks"].(map[string]interface{})
				total, _ := tracks["total"].(float64)
				playlists = append(playlists, Playlist{id, name, int(total)})
			}
		}
	}
	return playlists
}

func getPlaylistSongs(listId string, token string) string {
	// TODO Implement this
	return "Yeah man"
}

func makeHttpRequest(method string, api string, endpoint string, token string) []byte {
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
	return body
}
