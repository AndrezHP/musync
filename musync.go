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
	fmt.Println("Succes")
}

func test2() {
	tidalApi := NewTidalApi()
	track := Track{
		"31EJcKUZGaDCm684ByEx0G",
		"She Looks to Me",
		"Red Hot Chili Pepper",
		"Stadium Arcadium",
		"5lqPdkAz8XUdjYIiTnZTZz",
		5,
		2}
	tidalApi.searchTrack(track)
	tidalApi.searchAlbum(track)
	trackLookup(tidalApi, track)
}

func test() {
	albumId := "68789392"
	endpoint := "/albums/" + albumId
	args := []Pair{{"countryCode", "DK"}, {"include", "artists,items"}}
	testEndpoint("https://openapi.tidal.com/v2"+endpoint, args)
}

func entry() {
	// Tidal:
	// (Check if logged in/token exists and is valid)
	// - "You are not logged into Tidal -> log in"

	// Spotify:
	// (Check if logged in/token exists and is valid)
	// - "You are not logged into Spotify -> log in"
}

func updateFromSpotify() {
	// Get all playlists -> database (if !exist)
	// For all playlists in database:
	// - Get all songs
	// - - Add to track database if !exist
	// - - Add relation
}

func updateFromTidal() {}

func trackLookup(tidalApi TidalApi, searchTrack Track) string {
	albumId := tidalApi.searchAlbum(searchTrack)
	if albumId == "" {
		return ""
	}
	fmt.Println("Get album: ", albumId)
	trackIndexMap := make(map[string]DiscIndex)
	albumTracks := tidalApi.getAlbum(albumId, searchTrack.Album, "", trackIndexMap)
	tracks := tidalApi.getTracks(albumTracks, trackIndexMap)

	for _, track := range tracks {
		if track.Name == searchTrack.Name ||
			(track.TrackNumber == searchTrack.TrackNumber && track.DiscNumber == searchTrack.DiscNumber) {
			fmt.Println("Succes! Track: ", track)
			return track.Id
		}
	}
	return ""
}

func migrateSinglePlaylistToTidal(playlistId string, newPlaylistName string) {
	spotifyApi := NewSpotifyApi()
	tidalApi := NewTidalApi()
	var tracks = spotifyApi.getPlaylistTracks(playlistId, 0)

	var trackIds []string
	var notFound []Track
	for i, track := range tracks {
		var id = trackLookup(tidalApi, track)
		if id == "" {
			id = tidalApi.searchTrack(track)
		}
		fmt.Println("Index: ", i)
		if id == "" {
			notFound = append(notFound, track)
		} else {
			trackIds = append(trackIds, id)
		}
	}

	newPlaylistId := tidalApi.createPlaylist(newPlaylistName)
	fmt.Println("New Playlist ID:", newPlaylistId)

	sleep = 5000
	batchSize := 20
	for i := 0; i < len(trackIds); i += batchSize {
		batch := trackIds[i:min(i+batchSize, len(trackIds))]
		tidalApi.addTracks(newPlaylistId, batch)
	}
	fmt.Println("Not found: ", notFound)
}

type Pair struct {
	Key   string
	Value string
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
