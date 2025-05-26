package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type TidalApi struct {
	Client *http.Client
	Url    string
}

func NewTidalApi() TidalApi {
	clientParams := readClientParams(".tidalParams.json")
	scopes := []string{"user.read", "collection.read", "collection.write", "playlists.read", "playlists.write"}
	apiUrl := "https://openapi.tidal.com/v2"
	config := &oauth2.Config{
		ClientID:     clientParams.ClientId,
		ClientSecret: clientParams.ClientSecret,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.tidal.com/authorize",
			TokenURL: "https://auth.tidal.com/v1/oauth2/token",
		},
		RedirectURL: "http://localhost:8081/callback",
	}

	token := getToken(config, context.Background(), apiUrl, ".tidalToken.json", "8081", true)
	client := config.Client(context.Background(), token)
	return TidalApi{
		client,
		apiUrl,
	}
}

func (api TidalApi) getCurrentUserId() string {
	req, err := http.NewRequest("GET", api.Url+"/users/me", nil)
	check(err)
	result, _ := doRequestWithRetry(api.Client, req, false)
	data, _ := result["data"].(map[string]any)
	if id, ok := data["id"].(string); ok {
		return id
	} else {
		panic("Id of current user not found")
	}
}

func (api TidalApi) getUserPlaylists(userId string, next string) []Playlist {
	var request *http.Request
	if next == "" {
		endpoint := api.Url + fmt.Sprintf("/playlists?filter[r.owners.id]=%s", userId)
		req, err := http.NewRequest("GET", endpoint+"countryCode=DK", nil)
		check(err)
		request = req
	} else {
		req, err := http.NewRequest("GET", api.Url+next, nil)
		check(err)
		request = req
	}

	result, _ := doRequestWithRetry(api.Client, request, false)
	var playlists []Playlist
	if lists, ok := result["data"].([]any); ok && len(lists) > 0 {
		for i := range len(lists) {
			item, _ := lists[i].(map[string]any)
			attributes, _ := item["attributes"].(map[string]any)

			id, _ := item["id"].(string)
			name, _ := attributes["name"].(string)
			length, _ := attributes["numberOfItems"].(float64)
			playlists = append(playlists, Playlist{id, name, int(length)})
		}
	}

	links, _ := result["links"].(map[string]any)
	newNext, ok := links["next"].(string)
	if ok {
		log.Println("Fetching next playlists from: ", newNext)
		return append(playlists, api.getUserPlaylists(userId, newNext)...)
	} else {
		return playlists
	}
}

func (api TidalApi) getPlaylistTracks(playlistId string, next string) []Track {
	var request *http.Request
	if next == "" {
		endpoint := api.Url + fmt.Sprintf("/playlists/%s/relationships/items", playlistId)
		req, err := http.NewRequest("GET", endpoint+"countryCode=DK", nil)
		check(err)
		request = req
	} else {
		req, err := http.NewRequest("GET", api.Url+next, nil)
		check(err)
		request = req
	}

	result, _ := doRequestWithRetry(api.Client, request, false)
	var trackIds []string
	if data, ok := result["data"].([]any); ok && len(data) > 0 {
		for i := range len(data) {
			item, _ := data[i].(map[string]any)
			id, _ := item["id"].(string)
			trackIds = append(trackIds, id)
		}
	}

	tracks := api.getTracks(trackIds, nil)
	links, _ := result["links"].(map[string]any)
	newNext, ok := links["next"].(string)
	if ok {
		log.Println("Fetching next tracks from: ", newNext)
		return append(tracks, api.getPlaylistTracks(playlistId, newNext)...)
	} else {
		return tracks
	}
}

func (api TidalApi) getTracks(trackIds []string, trackIndexMap map[string]DiscIndex) []Track {
	var tracks []Track
	batchSize := 20
	for i := 0; i < len(trackIds); i += batchSize {
		batch := trackIds[i:min(i+batchSize, len(trackIds))]
		tracks = append(tracks, api.getTracksBatch(batch, trackIndexMap)...)
	}
	return tracks
}

func (api TidalApi) getTracksBatch(trackIds []string, trackIndexMap map[string]DiscIndex) []Track {
	req, err := http.NewRequest("GET", api.Url+"/tracks", nil)
	check(err)
	params := req.URL.Query()
	params.Set("countryCode", "DK")
	params.Set("include", "albums,artists,tracks")
	var combinedIds string
	for i, id := range trackIds {
		if i == 0 {
			combinedIds = combinedIds + id
		} else {
			combinedIds = combinedIds + "," + id
		}
	}
	params.Set("filter[id]", combinedIds)
	req.URL.RawQuery = params.Encode()

	result, _ := doRequestWithRetry(api.Client, req, false)
	albumMap := make(map[string]string)
	artistMap := make(map[string]string)
	if included, ok := result["included"].([]any); ok && len(included) > 0 {
		for i := range len(included) {
			item, _ := included[i].(map[string]any)
			itemId, _ := item["id"].(string)
			itemType, _ := item["type"].(string)
			attributes, _ := item["attributes"].(map[string]any)

			if itemType == "albums" {
				albumTitle, _ := attributes["title"].(string)
				albumMap[itemId] = albumTitle
			} else if itemType == "artists" {
				name, _ := attributes["name"].(string)
				artistMap[itemId] = name
			} else {
				panic("Unknown item type: " + itemType)
			}
		}
	}

	var tracks []Track
	if data, ok := result["data"].([]any); ok && len(data) > 0 {
		for i := range len(data) {
			item, _ := data[i].(map[string]any)

			trackId, _ := item["id"].(string)
			attributes, _ := item["attributes"].(map[string]any)
			relationships, _ := item["relationships"].(map[string]any)

			trackTitle, _ := attributes["title"].(string)

			albums, _ := relationships["albums"].(map[string]any)
			firstAlbum, _ := albums["data"].([]any)[0].(map[string]any)
			albumId, _ := firstAlbum["id"].(string)

			artists, _ := relationships["artists"].(map[string]any)
			firstArtist, _ := artists["data"].([]any)[0].(map[string]any)
			artistId, _ := firstArtist["id"].(string)

			var trackNumber = 0
			var discNumber = 0
			discIndex, ok := trackIndexMap[trackId]
			if ok {
				trackNumber = discIndex.TrackNumber
				discNumber = discIndex.DiscNumber
			}

			track := Track{trackId, trackTitle, artistMap[artistId], albumMap[albumId], albumId, trackNumber, discNumber}
			tracks = append(tracks, track)
		}
	}

	return tracks
}

func (api TidalApi) addTracks(playlistId string, trackIds []string) {
	endpoint := api.Url + fmt.Sprintf("/playlists/%s/relationships/items", playlistId)
	var tracksToAdd []any
	for _, id := range trackIds {
		trackToAdd := map[string]any{
			"id":   id,
			"type": "tracks",
		}
		tracksToAdd = append(tracksToAdd, trackToAdd)
	}
	payload := map[string]any{
		"data": tracksToAdd,
	}
	jsonData, err := json.Marshal(payload)
	check(err)

	req, err := http.NewRequest("POST", endpoint+"?countryCode=DK", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	check(err)

	_, response := doRequestWithRetry(api.Client, req, false)
	if response.StatusCode != 201 {
		log.Println("Could not add tracks: ", response)
	}
}

func (api TidalApi) createPlaylist(name string) string {
	endpoint := api.Url + "/playlists"
	payload := map[string]any{
		"data": map[string]any{
			"type": "playlists",
			"attributes": map[string]any{
				"name":        name,
				"description": "",
				"privacy":     "PRIVATE",
			},
		},
	}
	jsonData, err := json.Marshal(payload)
	check(err)

	req, err := http.NewRequest("POST", endpoint+"?countryCode=DK", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	check(err)

	result, _ := doRequestWithRetry(api.Client, req, false)
	data, _ := result["data"].(map[string]any)
	if id, ok := data["id"].(string); ok {
		return id
	} else {
		panic("Id of new playlist not found")
	}
}

// TODO Maybe use this for comparison also (with the addition of removing anything remaster related and -)
func cleanSearchString(input string) string {
	regex := regexp.MustCompile(`\[.+\]|[@#$%^&*\[\]:;,?/~\\|]|\b[Tt]rio\b|"`)
	return regex.ReplaceAllString(input, " ")
}

func (api TidalApi) searchTrack(track Track) string {
	searchString := cleanSearchString(track.Name + " " + track.Artist)
	endpoint := api.Url + fmt.Sprintf("/searchResults/%s/relationships/topHits", searchString)

	req, err := http.NewRequest("GET", endpoint+"?countryCode=DK", nil)
	check(err)

	result, _ := doRequestWithRetry(api.Client, req, false)
	data, _ := result["data"].([]any)
	for i := range len(data) {
		item, _ := data[i].(map[string]any)
		if item["type"].(string) == "tracks" {
			return item["id"].(string)
		}
	}
	if len(data) > 0 {
		trackId, _ := data[0].(map[string]any)["id"].(string)
		return trackId
	} else {
		return ""
	}
}

func (api TidalApi) searchAlbum(track Track) string {
	searchString := cleanSearchString(track.Album + " " + track.Artist)
	endpoint := api.Url + fmt.Sprintf("/searchResults/%s/relationships/albums", searchString)
	req, err := http.NewRequest("GET", endpoint+"?countryCode=DK", nil)
	check(err)

	log.Println("Searching album for: ", track)
	result, _ := doRequestWithRetry(api.Client, req, false)
	data, _ := result["data"].([]any)
	if len(data) > 0 {
		trackId, _ := data[0].(map[string]any)["id"].(string)
		return trackId
	} else {
		return ""
	}
}

type DiscIndex struct {
	TrackNumber int
	DiscNumber  int
}

func stringMatch(str1 string, str2 string) bool {
	return strings.ToLower(str1) == strings.ToLower(str2)
}

func (api TidalApi) getAlbum(albumId string, searchTrack Track, next string, trackIndexMap map[string]DiscIndex) []string {
	var req *http.Request
	var err error
	if next == "" {
		req, err = http.NewRequest("GET", api.Url+"/albums/"+albumId, nil)
		params := req.URL.Query()
		params.Set("countryCode", "DK")
		params.Set("include", "items,albums,artists")
		req.URL.RawQuery = params.Encode()
	} else {
		req, err = http.NewRequest("GET", api.Url+next+"&include=albums", nil)
	}

	check(err)
	result, _ := doRequestWithRetry(api.Client, req, false)

	if next == "" {
		albumName, _ := result["data"].(map[string]any)["attributes"].(map[string]any)["title"].(string)
		var albumArtist = ""
		included, _ := result["included"].([]any)
		for i := range len(included) {
			inc, _ := included[i].(map[string]any)
			if inc["type"].(string) == "artists" {
				albumArtist = inc["attributes"].(map[string]any)["name"].(string)
			}
		}

		if !stringMatch(albumArtist, searchTrack.Artist) || !stringMatch(albumName, searchTrack.Album) {
			log.Println("Album or artist did not match!", albumArtist, albumName)
			return nil
		}
	}

	var links map[string]any
	var data []any
	if next == "" {
		relationships := result["data"].(map[string]any)["relationships"].(map[string]any)
		data = relationships["items"].(map[string]any)["data"].([]any)
		links = relationships["items"].(map[string]any)["links"].(map[string]any)
	} else {
		data = result["data"].([]any)
		links = result["links"].(map[string]any)
	}

	var trackIds []string
	for i := range len(data) {
		item, _ := data[i].(map[string]any)
		id, _ := item["id"].(string)

		trackNumber, _ := item["meta"].(map[string]any)["trackNumber"].(float64)
		discNumber, _ := item["meta"].(map[string]any)["volumeNumber"].(float64)
		trackIndexMap[id] = DiscIndex{int(trackNumber), int(discNumber)}
		trackIds = append(trackIds, id)
	}

	newNext, ok := links["next"].(string)
	if ok {
		fmt.Println("Fetching next tracks on album")
		nextTrackIds := api.getAlbum(albumId, searchTrack, newNext, trackIndexMap)
		return append(trackIds, nextTrackIds...)
	} else {
		return trackIds
	}
}
