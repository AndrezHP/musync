package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
)

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

	token := getToken(config, context.Background(), apiUrl, ".spotifyToken.json", "8080")
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

	var result map[string]any
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

	result, _ := doRequestWithRetry(api.Client, req, false)
	var playlists []Playlist
	if lists, ok := result["items"].([]any); ok && len(lists) > 0 {
		for i := range len(lists) {
			item, _ := lists[i].(map[string]any)

			name, _ := item["name"].(string)
			id, _ := item["id"].(string)
			tracks, _ := item["tracks"].(map[string]any)
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

	result, _ := doRequestWithRetry(api.Client, req, false)
	var tracks []Track
	if lists, ok := result["items"].([]any); ok && len(lists) > 0 {
		for i := range len(lists) {
			item, _ := lists[i].(map[string]any)
			track, _ := item["track"].(map[string]any)
			tracks = append(tracks, trackFromMap(track))
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

func trackFromMap(trackMap map[string]any) Track {
	id, _ := trackMap["id"].(string)
	trackName, _ := trackMap["name"].(string)
	trackNumber, _ := trackMap["track_number"].(float64)
	discNumber, _ := trackMap["disc_number"].(float64)

	trackAlbum, _ := trackMap["album"].(map[string]any)
	albumName, _ := trackAlbum["name"].(string)
	albumId, _ := trackAlbum["id"].(string)

	artists, _ := trackMap["artists"].([]any)
	firstArtist, _ := artists[0].(map[string]any)
	artistName, _ := firstArtist["name"].(string)

	return Track{id, trackName, artistName, albumName, albumId, int(trackNumber), int(discNumber)}
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
	var result map[string]any
	err = json.Unmarshal(responseBody, &result)
	check(err)

	// Parse result
	tracks, _ := result["tracks"].(map[string]any)["items"].([]any)
	firstTrack, _ := tracks[0].(map[string]any)
	return trackFromMap(firstTrack)
}
