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
	// spotifyApi := NewSpotifyApi()

	// var userId = spotifyApi.getCurrentUserId()
	// var playlists = spotifyApi.getUserPlaylists(userId, 0)

	// fmt.Println("%s", playlists)
	// fmt.Println("Number of playlists: ", len(playlists))

	// firstPlaylist := playlists[0]
	// var tracks = spotifyApi.getPlaylistTracks(firstPlaylist.Id, 0)
	// fmt.Println("%s", tracks)
	// fmt.Println("Number of tracks: ", len(tracks))

	// var firstTrack = tracks[0]
	// resultTrack := spotifyApi.searchTrack(firstTrack.Name, firstTrack.Artist, firstTrack.Album)
	// fmt.Println("Result track: ", resultTrack)

	tidalApi := NewTidalApi()
	// tidalApi.createPlaylist("Test")

	userId := tidalApi.getCurrentUserId()
	fmt.Println("id: ", userId)

	playlists := tidalApi.getUserPlaylists(userId, "")
	fmt.Println("Playlists: %s", playlists)

	firstPlaylist := playlists[0]
	tracks := tidalApi.getPlaylistTracks(firstPlaylist.Id, "")
	fmt.Println("Tracks on first playlist %s", tracks)
	fmt.Println("Number of tracks: ", len(tracks))

	// playlistId := "1147ae9d-1ad2-4759-a2cc-9b9c441ae467"
	// tidalApi.addTrack(playlistId, "179999775")
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

type Playlist struct {
	Id     string
	Name   string
	Length int
}

type Track struct {
	Id     string
	Name   string
	Album  string
	Artist string
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

func doRequestWithRetry(client *http.Client, request *http.Request, printBody bool) (map[string]interface{}, *http.Response) {
	time.Sleep(4000 * time.Millisecond)
	response, err := client.Do(request)
	check(err)
	if response.StatusCode == 429 {
		fmt.Println("Rate limit hit!")
		time.Sleep(5 * time.Second)
		return doRequestWithRetry(client, request, printBody)
	}

	reponseBody := getBody(response)
	var result map[string]interface{}
	err = json.Unmarshal(reponseBody, &result)
	if err != nil {
		fmt.Println("Error: %s, response: ", err, response)
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
