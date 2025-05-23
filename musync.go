package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

var sleep float64 = 1500

type Playlist struct {
	Id     string
	Name   string
	Length int
}

type Track struct {
	Id          string
	Name        string
	Artist      string
	Album       string
	AlbumId     string
	TrackNumber int
	DiscNumber  int
}

type ClientParams struct {
	ClientId     string
	ClientSecret string
}

func main() {
	// migrateSinglePlaylist("1X9Egf7h5Spfrabylh6Yqk", "Piano Jazz")
	newTest()
	fmt.Println("Succes")
}

func newTest() {
	tidalApi := NewTidalApi()
	track := Track{
		"",
		"Beethoven: Allegretto in C Minor, Hess 69",
		"\"Allegretto\" - Piano Works - Vol. II",
		"Ludwig Van Beethoven",
	}

	fmt.Println("Searching album for: ", track)
	albumId := tidalApi.searchAlbum(track)

	fmt.Println("Get album: ", albumId)
	tidalApi.getAlbum(albumId, "")
}

func migrateSinglePlaylist(playlistId string, newPlaylistName string) {
	spotifyApi := NewSpotifyApi()
	tidalApi := NewTidalApi()
	var tracks = spotifyApi.getPlaylistTracks(playlistId, 0)

	var trackIdsToAdd []string
	var notFound []Track
	for i, track := range tracks {
		id := tidalApi.searchTrack(track)
		fmt.Println("Index: ", i)
		if id == "" {
			notFound = append(notFound, track)
		} else {
			trackIdsToAdd = append(trackIdsToAdd, id)
		}
	}

	newPlaylistId := tidalApi.createPlaylist(newPlaylistName)
	fmt.Println("New Playlist ID:", newPlaylistId)

	batchSize := 20
	for i := 0; i < len(trackIdsToAdd); i += batchSize {
		end := i + batchSize
		if end > len(trackIdsToAdd) {
			end = len(trackIdsToAdd)
		}
		batch := trackIdsToAdd[i:end]
		tidalApi.addTracks(newPlaylistId, batch)
	}
	fmt.Println("Not found: %s", notFound)
}

func testStuff() {
	// spotifyApi := NewSpotifyApi()
	// // var userId = spotifyApi.getCurrentUserId()
	// // var playlists = spotifyApi.getUserPlaylists(userId, 0)

	// // fmt.Println("%s", playlists)
	// // fmt.Println("Number of playlists: ", len(playlists))

	// // firstPlaylist := playlists[0]
	// var tracks = spotifyApi.getPlaylistTracks("522r13v8YaMF8SCVk7l45i", 0)
	// // fmt.Println("%s", tracks)
	// // fmt.Println("Number of tracks: ", len(tracks))

	// // var firstTrack = tracks[0]
	// // resultTrack := spotifyApi.searchTrack(firstTrack.Name, firstTrack.Artist, firstTrack.Album)
	// // fmt.Println("Result track: ", resultTrack)

	// tidalApi := NewTidalApi()
	// playlistId := tidalApi.createPlaylist("Guessed")

	// // userId := tidalApi.getCurrentUserId()
	// // fmt.Println("id: ", userId)

	// // playlists := tidalApi.getUserPlaylists(userId, "")
	// // fmt.Println("Playlists: %s", playlists)

	// // firstPlaylist := playlists[0]
	// // tracks := tidalApi.getPlaylistTracks(firstPlaylist.Id, "")
	// // fmt.Println("Tracks on first playlist %s", tracks)
	// // fmt.Println("Number of tracks: ", len(tracks))

	// // playlistId := "1147ae9d-1ad2-4759-a2cc-9b9c441ae467"
	// // tidalApi.addTrack(playlistId, "179999775")
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

func doRequestWithRetry(client *http.Client, request *http.Request, printBody bool) (map[string]any, *http.Response) {
	time.Sleep(time.Duration(sleep) * time.Millisecond)
	response, err := client.Do(request)
	check(err)
	if response.StatusCode == 429 {
		sleep = math.Min(sleep+100, 4000)
		fmt.Println("Rate limit hit! Increasing sleep time to: ", sleep)
		time.Sleep(5 * time.Second)
		return doRequestWithRetry(client, request, printBody)
	}

	reponseBody := getBody(response)
	var result map[string]any
	err = json.Unmarshal(reponseBody, &result)
	if err != nil {
		fmt.Println("Error: ", err, "Response", response)
	}

	if printBody {
		printJson(reponseBody)
	}
	return result, response
}

func printJson(body []byte) {
	var prettyJSON bytes.Buffer
	json.Indent(&prettyJSON, body, "", "  ")
	fmt.Println("Json: ", string(prettyJSON.Bytes()))
}
