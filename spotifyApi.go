package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"os"
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
	OAuthHandler OAuthHandler
	Url          string
}

func readClientParams() (string, string) {
	file, err := os.Open(".spotify")
	check(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var clientParams []string
	for i := 0; i < 2 && scanner.Scan(); i++ {
		clientParams = append(clientParams, scanner.Text())
	}

	check(scanner.Err())
	return clientParams[0], clientParams[1]
}

func NewSpotifyApi() (SpotifyApi, error) {
	clientId, clientSecret := readClientParams()
	scopes := []string{"user-read-private", "user-read-email", "playlist-read-private", "playlist-read-collaborative"}
	apiUrl := "https://api.spotify.com/"
	conf := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.spotify.com/authorize",
			TokenURL: "https://accounts.spotify.com/api/token",
		},
		RedirectURL: "http://localhost:8080/callback",
	}

	oauthSpotify, err := NewOAuthHandler(conf, context.Background(), make(chan ApiToken), apiUrl)
	check(err)
	api := SpotifyApi{
		oauthSpotify,
		apiUrl,
	}
	return api, nil
}

func (api SpotifyApi) getCurrentUserId() string {
	accessToken := api.OAuthHandler.getAccessToken()
	var response = makeHttpRequest("GET", api.Url, "v1/me", accessToken)
	var result map[string]interface{}
	err := json.Unmarshal(response, &result)
	check(err)

	if id, ok := result["id"]; ok {
		return fmt.Sprint(id)
	} else {
		panic("Id of current user not found")
	}
}

func (api SpotifyApi) getUserPlaylists(userId string) []Playlist {
	accessToken := api.OAuthHandler.getAccessToken()
	var userPlaylists = fmt.Sprintf("v1/users/%s/playlists", userId)
	var res = makeHttpRequest("GET", api.Url, userPlaylists, accessToken)

	var result map[string]interface{}
	err := json.Unmarshal(res, &result)
	check(err)

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

func (api SpotifyApi) getPlaylistSongs(listId string) string {
	return "Yeah man"
}

func (api SpotifyApi) searchSong() string {
	return "Yeah man"
}
