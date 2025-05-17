package main

import (
	"context"
	"encoding/json"
	"golang.org/x/oauth2"
	"net/http"
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

func (api TidalApi) getUserPlaylists(userId string) []Playlist {
	return nil
}

func (api TidalApi) getPlaylistTracks(listId string) string {
	return "Yeah man"
}

func (api TidalApi) searchTrack() string {
	return "Yeah man"
}
