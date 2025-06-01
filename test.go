package main

import (
	"github.com/AndrezHP/musync/cmd"
	"log"
	"net/http"
)

type Pair struct {
	Key   string
	Value string
}

func testEndpoint(endpoint string, args []Pair) {
	tidalApi := cmd.NewTidalApi()
	req, err := http.NewRequest("GET", endpoint, nil)
	cmd.Check(err)

	params := req.URL.Query()
	for _, arg := range args {
		params.Set(arg.Key, arg.Value)
	}
	req.URL.RawQuery = params.Encode()
	cmd.DoRequest(tidalApi.Client, req, true)
}

// func testStuff() {
// 	tidalApi := NewTidalApi()

// 	tidalUserId := tidalApi.getCurrentUserId()
// 	fmt.Println("id: ", tidalUserId)

// 	playlists := tidalApi.getUserPlaylists(tidalUserId, "")
// 	fmt.Println("Playlists: ", playlists)

// 	firstPlaylist := playlists[0]
// 	tracks := tidalApi.getPlaylistTracks(firstPlaylist.Id, "")
// 	fmt.Println("Tracks on first playlist: ", tracks)
// 	fmt.Println("Number of tracks: ", len(tracks))
// }

// func testSpotifyCalls() {
// 	spotifyApi := NewSpotifyApi()

// 	var userId = spotifyApi.getCurrentUserId()
// 	fmt.Println("UserId:", userId)

// 	var playlists = spotifyApi.getUserPlaylists(userId, 0)
// 	fmt.Println("Playlists: ", playlists)
// 	fmt.Println("Number of playlists: ", len(playlists))

// 	firstPlaylist := playlists[0]
// 	var tracks = spotifyApi.getPlaylistTracks(firstPlaylist.Id, 0)
// 	fmt.Println("Tracks: ", tracks)
// 	fmt.Println("Number of tracks: ", len(tracks))

// 	var firstTrack = tracks[0]
// 	resultTrack := spotifyApi.searchTrack(firstTrack.Name, firstTrack.Artist, firstTrack.Album)
// 	fmt.Println("Result track: ", resultTrack)
// }

// Does not work...
func deleteAllTestPlaylists() {
	api := cmd.NewTidalApi()
	userId := api.GetCurrentUserId()
	playlists := api.GetUserPlaylists(userId, "")
	for _, playlist := range playlists {
		if playlist.Name == "Test" {
			deletePlaylist(api, playlist.Id)
		}
	}
}

func deletePlaylist(api cmd.TidalApi, playlistId string) {
	req, err := http.NewRequest("DELETE", api.Url+"/playlists/"+playlistId, nil)
	cmd.Check(err)
	_, response := cmd.DoRequest(api.Client, req, false)
	if response.StatusCode == 204 {
		log.Println("Playlist deleted: ", playlistId)
	}
}
