package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	spotifyApi := NewSpotifyApi()

	var userId = spotifyApi.getCurrentUserId()
	var playlists = spotifyApi.getUserPlaylists(userId, 0)

	fmt.Println("%s", playlists)
	fmt.Println("Number of playlists: ", len(playlists))

	firstPlaylist := playlists[0]
	var tracks = spotifyApi.getPlaylistTracks(firstPlaylist.Id, 0)
	fmt.Println("%s", tracks)
	fmt.Println("Number of tracks: ", len(tracks))

	var firstTrack = tracks[0]
	resultTrack := spotifyApi.searchTrack(firstTrack.Name, firstTrack.Artist, firstTrack.Album)
	fmt.Println("Result track: ", resultTrack)
}

func getBody(response *http.Response) []byte {
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode > 299 {
		log.Fatalf("Response failed with status code: %d and\nbody: %s\n", response.StatusCode, body)
	}
	check(err)
	return body
}

type ClientParams struct {
	ClientId     string
	ClientSecret string
}

func readClientParams(filePath string) ClientParams {
	var clientParams ClientParams
	file, err := os.Open(filePath)
	defer file.Close()
	check(err)

	json.NewDecoder(file).Decode(&clientParams)
	return clientParams
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
