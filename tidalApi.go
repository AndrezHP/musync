package main

import (
	"context"
	"golang.org/x/oauth2"
	"net/http"
)

type TidalApi struct {
	Client *http.Client
	Url    string
}

func NewTidalApi() (TidalApi, error) {
	clientParams := readClientParams(".tidalParams.json")
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

	token := getToken(config, context.Background(), apiUrl, ".tidalToken.json")
	client := config.Client(context.Background(), token)
	return TidalApi{
		client,
		apiUrl,
	}, nil
}

func (api TidalApi) getCurrentUserId() string {
	return "Yeah man"
}

func (api TidalApi) getUserPlaylists(userId string) []Playlist {
	return nil
}

func (api TidalApi) getPlaylistSongs(listId string) string {
	return "Yeah man"
}

func (api TidalApi) searchSong() string {
	return "Yeah man"
}
