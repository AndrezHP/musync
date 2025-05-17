package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"net/http"
)

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
	Href  string
	Total int
}

type SpotifyApi struct {
	Client *http.Client
	Url    string
}

func NewSpotifyApi() (SpotifyApi, error) {
	clientParams := readClientParams(".spotifyParams.json")
	scopes := []string{"user-read-private", "user-read-email", "playlist-read-private", "playlist-read-collaborative"}
	apiUrl := "https://api.spotify.com/"
	config := &oauth2.Config{
		ClientID:     clientParams.ClientId,
		ClientSecret: clientParams.ClientSecret,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
		RedirectURL: "http://localhost:8080/callback",
	}

	token := getToken(config, context.Background(), apiUrl, ".spotifyToken.json")
	client := config.Client(context.Background(), token)
	return SpotifyApi{
		client,
		apiUrl,
	}, nil
}

func (api SpotifyApi) getCurrentUserId() string {
	res, err := api.Client.Get(api.Url + "v1/me")
	check(err)
	responseBody := getBody(res)

	var result map[string]interface{}
	err = json.Unmarshal(responseBody, &result)
	check(err)

	if id, ok := result["id"]; ok {
		return fmt.Sprint(id)
	} else {
		panic("Id of current user not found")
	}
}

func (api SpotifyApi) getUserPlaylists(userId string, offset int) []Playlist {
	res, err := api.Client.Get(api.Url + fmt.Sprintf("v1/users/%s/playlists", userId))
	check(err)
	responseBody := getBody(res)

	var result map[string]interface{}
	err = json.Unmarshal(responseBody, &result)
	check(err)

	var playlists []Playlist
	if lists, ok := result["items"].([]interface{}); ok && len(lists) > 0 {
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

	newOffset := offset + 50
	var total int
	if totalFloat, ok := result["total"].(float64); ok {
		total = int(totalFloat)
	} else {
		panic("Could not find total number of playlists")
	}

	if newOffset < total { // TODO There might be an off by one here
		return append(playlists, api.getUserPlaylists(userId, newOffset)...)
	} else {
		return playlists
	}
}

func (api SpotifyApi) getPlaylistSongs(listId string) string {
	return "Yeah man"
}

func (api SpotifyApi) searchSong() string {
	return "Yeah man"
}
