package main

import (
	"context"
	"golang.org/x/oauth2"
)

type TidalApi struct {
	OAuthHandler OAuthHandler
	Url          string
}

func NewTidalApi() (SpotifyApi, error) {
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
