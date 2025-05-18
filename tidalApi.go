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
	var endpoint string
	if next == "" {
		endpoint = api.Url + fmt.Sprintf("v2/playlists?filter[r.owners.id]=%s", userId)
	} else {
		endpoint = next
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	check(err)

	params := req.URL.Query()
	params.Set("countryCode", "DK")
	req.URL.RawQuery = params.Encode()

	res, err := api.Client.Do(req)
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
		return append(playlists, api.getUserPlaylists(userId, newNext)...)
	} else {
		return playlists
	}
}
}

func (api TidalApi) getPlaylistTracks(listId string) string {
	return "Yeah man"
}

func (api TidalApi) searchTrack() string {
	return "Yeah man"
}

func printJson(body []byte) {
	var prettyJSON bytes.Buffer
	json.Indent(&prettyJSON, body, "", "  ")
	fmt.Println("Json: ", string(prettyJSON.Bytes()))
}
