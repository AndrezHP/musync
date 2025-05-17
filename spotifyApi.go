package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
)

type Playlist struct {
	Id     string
	Name   string
	Length int
}

type Track struct {
	Id     string
	Name   string
	Album  string
	Artist string
}

type SpotifyApi struct {
	Client *http.Client
	Url    string
}

func NewSpotifyApi() SpotifyApi {
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
	}
}

func (api SpotifyApi) getCurrentUserId() string {
	res, err := api.Client.Get(api.Url + "v1/me")
	check(err)
	responseBody := getBody(res)

	var result map[string]interface{}
	err = json.Unmarshal(responseBody, &result)
	check(err)

	if id, ok := result["id"].(string); ok {
		return id
	} else {
		panic("Id of current user not found")
	}
}

func (api SpotifyApi) getUserPlaylists(userId string, offset int) []Playlist {
	endpoint := api.Url + fmt.Sprintf("v1/users/%s/playlists", userId)
	req, err := http.NewRequest("GET", endpoint, nil)
	check(err)

	params := req.URL.Query()
	params.Set("limit", "50")
	params.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = params.Encode()

	res, err := api.Client.Do(req)
	check(err)

	responseBody := getBody(res)
	var result map[string]interface{}
	err = json.Unmarshal(responseBody, &result)
	check(err)

	var playlists []Playlist
	if lists, ok := result["items"].([]interface{}); ok && len(lists) > 0 {
		for i := 0; i < len(lists); i++ {
			item, _ := lists[i].(map[string]interface{})

			name, _ := item["name"].(string)
			id, _ := item["id"].(string)
			tracks, _ := item["tracks"].(map[string]interface{})
			total, _ := tracks["total"].(float64)
			playlists = append(playlists, Playlist{id, name, int(total)})
		}
	}

	newOffset := offset + 50
	var total int
	if totalFloat, ok := result["total"].(float64); ok {
		total = int(totalFloat)
	} else {
		panic("Could not find total number of playlists")
	}

	if newOffset < total {
		return append(playlists, api.getUserPlaylists(userId, newOffset)...)
	} else {
		return playlists
	}
}

func (api SpotifyApi) getPlaylistTracks(playlistId string, offset int) []Track {
	endpoint := api.Url + fmt.Sprintf("v1/playlists/%s/tracks", playlistId)
	req, err := http.NewRequest("GET", endpoint, nil)
	check(err)

	params := req.URL.Query()
	params.Set("limit", "50")
	params.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = params.Encode()

	res, err := api.Client.Do(req)
	check(err)

	responseBody := getBody(res)
	var result map[string]interface{}
	err = json.Unmarshal(responseBody, &result)
	check(err)

	var tracks []Track
	if lists, ok := result["items"].([]interface{}); ok && len(lists) > 0 {
		for i := 0; i < len(lists); i++ {
			item, _ := lists[i].(map[string]interface{})
			track, _ := item["track"].(map[string]interface{})

			id, _ := track["id"].(string)
			name, _ := track["name"].(string)

			album, _ := track["album"].(map[string]interface{})
			albumName, _ := album["name"].(string)

			artists, _ := track["artists"].([]interface{})
			firstArtist, _ := artists[0].(map[string]interface{})
			artistName, _ := firstArtist["name"].(string)

			tracks = append(tracks, Track{id, name, albumName, artistName})
		}
	}

	newOffset := offset + 50
	var total int
	if totalFloat, ok := result["total"].(float64); ok {
		total = int(totalFloat)
	} else {
		panic("Could not find total number of songs on playlist")
	}

	if newOffset < total {
		return append(tracks, api.getPlaylistTracks(playlistId, newOffset)...)
	} else {
		return tracks
	}
}

func (api SpotifyApi) searchTrack(name string, artist string, album string) Track {
	req, err := http.NewRequest("GET", api.Url+"v1/search", nil)
	check(err)

	searchString := fmt.Sprintf("track:%s artist:%s album:%s", name, artist, album)

	params := req.URL.Query()
	params.Set("q", searchString)
	params.Set("type", "track")
	req.URL.RawQuery = params.Encode()

	res, err := api.Client.Do(req)
	check(err)

	responseBody := getBody(res)
	var result map[string]interface{}
	err = json.Unmarshal(responseBody, &result)
	check(err)

	// Parse result
	tracks, _ := result["tracks"].(map[string]interface{})["items"].([]interface{})
	firstTrack, _ := tracks[0].(map[string]interface{})

	id, _ := firstTrack["id"].(string)
	trackName, _ := firstTrack["name"].(string)

	trackAlbum, _ := firstTrack["album"].(map[string]interface{})
	albumName, _ := trackAlbum["name"].(string)

	artists, _ := firstTrack["artists"].([]interface{})
	firstArtist, _ := artists[0].(map[string]interface{})
	artistName, _ := firstArtist["name"].(string)

	return Track{id, trackName, albumName, artistName}
}
