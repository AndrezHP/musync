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

	token := getToken(config, context.Background(), apiUrl, ".spotifyToken.json", "8080", false)
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
	var endpoint = fmt.Sprintf("v1/users/%s/playlists", userId)
	endpoint += "?limit=50&offset=" + strconv.Itoa(offset)
	req, err := http.NewRequest("GET", api.Url+endpoint, nil)
	check(err)

	result, _ := doRequestWithRetry(api.Client, req, false)
	var playlists []Playlist
	jsonResult := JsonWrapper{result}
	items := jsonResult.getSlice("items")
	for i := range len(items) {
		item := makeJson(items[i])
		playlist := Playlist{
			item.getString("id"),
			item.getString("name"),
			item.getInt("total"),
		}
		playlists = append(playlists, playlist)
	}

	newOffset := offset + 50
	if newOffset < jsonResult.getInt("total") {
		return append(playlists, api.getUserPlaylists(userId, newOffset)...)
	} else {
		return playlists
	}
}

func (api SpotifyApi) getPlaylistTracks(playlistId string, offset int) []Track {
	endpoint := fmt.Sprintf("v1/playlists/%s/tracks", playlistId)
	req, err := http.NewRequest("GET", api.Url+endpoint, nil)
	check(err)

	params := req.URL.Query()
	params.Set("limit", "50")
	params.Set("offset", strconv.Itoa(offset))
	req.URL.RawQuery = params.Encode()

	result, _ := doRequestWithRetry(api.Client, req, false)
	jsonResult := JsonWrapper{result}
	var tracks []Track
	items := jsonResult.getSlice("items")
	for i := range len(items) {
		item := makeJson(items[i])
		tracks = append(tracks, trackFromMap(item.get("track")))
	}

	newOffset := offset + 50
	if newOffset < jsonResult.getInt("total") {
		return append(tracks, api.getPlaylistTracks(playlistId, newOffset)...)
	} else {
		return tracks
	}
}

func trackFromMap(track JsonWrapper) Track {
	id := track.getString("id")

	trackName := track.getString("name")
	trackNumber := track.getInt("track_number")
	discNumber := track.getInt("disc_number")

	albumName := track.get("album").getString("name")
	albumType := track.get("album").getString("album_type")
	albumId := track.get("album").getString("id")

	firstArtist := track.getAt("artists", 0)
	artistName := firstArtist.getString("name")

	return Track{id, trackName, artistName, albumName, albumType, albumId, int(trackNumber), int(discNumber)}
}

func (api SpotifyApi) searchTrack(name string, artist string, album string) Track {
	name = cleanSearchString(name)
	artist = cleanSearchString(artist)
	album = cleanSearchString(album)
	searchString := fmt.Sprintf("track:\"%s\" artist:\"%s\" album:\"%s\"", name, artist, album)
	endpoint := "v1/search" + "?type=track&q=" + searchString
	req, err := http.NewRequest("GET", api.Url+endpoint, nil)
	check(err)

	result, _ := doRequestWithRetry(api.Client, req, false)
	check(err)

	tracks := JsonWrapper{result}.get("tracks")
	track := tracks.getAt("items", 0)
	return trackFromMap(track)
}
