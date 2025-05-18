package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"math/rand"
	"net/http"
	"time"
)

type TidalApi struct {
	Client *http.Client
	Url    string
}

func NewTidalApi() TidalApi {
	clientParams := readClientParams(".tidalParams.json")
	scopes := []string{"user.read", "collection.read", "collection.write", "playlists.read", "playlists.write"}
	apiUrl := "https://openapi.tidal.com/"
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

	token := getToken(config, context.Background(), apiUrl, ".tidalToken.json", "8081")
	client := config.Client(context.Background(), token)
	return TidalApi{
		client,
		apiUrl,
	}
}

func (api TidalApi) getCurrentUserId() string {
	res, err := api.Client.Get(api.Url + "v2/users/me")
	check(err)
	responseBody := getBody(res)

	var result map[string]interface{}
	err = json.Unmarshal(responseBody, &result)
	check(err)

	data, _ := result["data"].(map[string]interface{})
	if id, ok := data["id"].(string); ok {
		return id
	} else {
		panic("Id of current user not found")
	}
}

func (api TidalApi) getUserPlaylists(userId string, next string) []Playlist {
	var request *http.Request
	if next == "" {
		endpoint := api.Url + fmt.Sprintf("v2/playlists?filter[r.owners.id]=%s", userId)
		req, err := http.NewRequest("GET", endpoint, nil)
		check(err)

		params := req.URL.Query()
		params.Set("countryCode", "DK")
		req.URL.RawQuery = params.Encode()
		request = req
	} else {
		req, err := http.NewRequest("GET", next, nil)
		check(err)
		request = req
	}

	res, err := api.Client.Do(request)
	check(err)

	responseBody := getBody(res)
	var result map[string]interface{}
	err = json.Unmarshal(responseBody, &result)
	check(err)

	var playlists []Playlist
	if lists, ok := result["data"].([]interface{}); ok && len(lists) > 0 {
		for i := 0; i < len(lists); i++ {
			item, _ := lists[i].(map[string]interface{})
			attributes, _ := item["attributes"].(map[string]interface{})

			id, _ := item["id"].(string)
			name, _ := attributes["name"].(string)
			length, _ := attributes["numberOfItems"].(float64)
			playlists = append(playlists, Playlist{id, name, int(length)})
		}
	}

	links, _ := result["links"].(map[string]interface{})
	newNext, ok := links["next"].(string)
	if ok {
		fmt.Println("Fetching next playlists")
		return append(playlists, api.getUserPlaylists(userId, newNext)...)
	} else {
		return playlists
	}
}

func (api TidalApi) getPlaylistTracks(playlistId string, next string) []Track {
	var request *http.Request
	if next == "" {
		endpoint := api.Url + fmt.Sprintf("v2/playlists/%s/relationships/items", playlistId)
		req, err := http.NewRequest("GET", endpoint, nil)
		check(err)

		params := req.URL.Query()
		params.Set("countryCode", "DK")
		params.Set("include", "items")
		req.URL.RawQuery = params.Encode()

		request = req
	} else {
		req, err := http.NewRequest("GET", api.Url+"v2"+next, nil)
		check(err)
		request = req
	}

	time.Sleep(time.Duration(2000+rand.Intn(1000)) * time.Millisecond)
	res, err := api.Client.Do(request)
	check(err)

	responseBody := getBody(res)
	var result map[string]interface{}
	err = json.Unmarshal(responseBody, &result)
	check(err)

	var trackIds []string
	if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
		for i := 0; i < len(data); i++ {
			item, _ := data[i].(map[string]interface{})
			id, _ := item["id"].(string)
			trackIds = append(trackIds, id)
		}
	}

	tracks := api.getTracks(trackIds)
	links, _ := result["links"].(map[string]interface{})
	newNext, ok := links["next"].(string)
	if ok {
		fmt.Println("Fetching next tracks")
		return append(tracks, api.getPlaylistTracks(playlistId, newNext)...)
	} else {
		return tracks
	}
}

func (api TidalApi) getTracks(trackIds []string) []Track {
	req, err := http.NewRequest("GET", api.Url+"v2/tracks", nil)
	check(err)

	params := req.URL.Query()
	params.Set("countryCode", "DK")
	params.Set("include", "albums,artists")
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

	// Trying to not hit the rate limit
	time.Sleep(time.Duration(2000+rand.Intn(1000)) * time.Millisecond)
	res, err := api.Client.Do(req)
	if res.StatusCode != 200 {
		fmt.Println("Rate limit hit!")
		time.Sleep(5 * time.Second)
		return api.getTracks(trackIds)
	}

	responseBody := getBody(res)
	var result map[string]interface{}
	err = json.Unmarshal(responseBody, &result)
	check(err)

	albumMap := make(map[string]string)
	artistMap := make(map[string]string)
	if included, ok := result["included"].([]interface{}); ok && len(included) > 0 {
		for i := 0; i < len(included); i++ {
			item, _ := included[i].(map[string]interface{})
			itemId, _ := item["id"].(string)
			itemType, _ := item["type"].(string)
			attributes, _ := item["attributes"].(map[string]interface{})

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
	if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
		for i := 0; i < len(data); i++ {
			item, _ := data[i].(map[string]interface{})
			trackId, _ := item["id"].(string)
			attributes, _ := item["attributes"].(map[string]interface{})
			relationships, _ := item["relationships"].(map[string]interface{})

			trackTitle, _ := attributes["title"].(string)

			albums, _ := relationships["albums"].(map[string]interface{})
			firstAlbum, _ := albums["data"].([]interface{})[0].(map[string]interface{})
			albumId, _ := firstAlbum["id"].(string)

			artists, _ := relationships["artists"].(map[string]interface{})
			firstArtist, _ := artists["data"].([]interface{})[0].(map[string]interface{})
			artistId, _ := firstArtist["id"].(string)

			tracks = append(tracks, Track{trackId, trackTitle, albumMap[albumId], artistMap[artistId]})
		}
	}

	return tracks
}

func (api TidalApi) addTrack(playlistId string, trackId string) {
	endpoint := api.Url + fmt.Sprintf("v2/playlists/%s/relationships/items", playlistId)
	payload := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{
				"id":   trackId,
				"type": "tracks",
			}},
	}

	jsonData, err := json.Marshal(payload)
	check(err)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	check(err)

	params := req.URL.Query()
	params.Set("countryCode", "DK")
	req.URL.RawQuery = params.Encode()

	res, err := api.Client.Do(req)
	if res.StatusCode != 201 {
		panic(fmt.Sprintf("Could not create resource: %s", res))
	}
}

func (api TidalApi) createPlaylist(name string) {
	endpoint := api.Url + "v2/playlists"
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "playlists",
			"attributes": map[string]interface{}{
				"name":        name,
				"description": "",
				"privacy":     "PRIVATE",
			},
		},
	}
	jsonData, err := json.Marshal(payload)
	check(err)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	check(err)

	params := req.URL.Query()
	params.Set("countryCode", "DK")
	req.URL.RawQuery = params.Encode()

	res, err := api.Client.Do(req)
	if res.StatusCode != 201 {
		panic(fmt.Sprintf("Could not create resource: %s", res))
	}
}

func (api TidalApi) searchTrack() string {
	return "Yeah man"
}

func printJson(body []byte) {
	var prettyJSON bytes.Buffer
	json.Indent(&prettyJSON, body, "", "  ")
	fmt.Println("Json: ", string(prettyJSON.Bytes()))
}
