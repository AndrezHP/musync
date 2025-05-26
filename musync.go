package main

import (
	"fmt"
	"log"
	"net/http"
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

func main() {
	startLogging()
	start := time.Now()

	migrateSinglePlaylistToTidal("1Nn4jKAU6ipUwrCDdPGmei", "Test")
	// test()

	end := time.Now()
	elapsed := end.Sub(start)
	log.Println("Elapsed: ", time.Duration.Milliseconds(elapsed))
}

func test() {
	// api := NewTidalApi()
	// searchString := "Blanket"
	// var endpoint = "/searchResults/" + searchString + "/relationships/artists"

	// endpoint = "/artists/3947529/relationships/albums?countryCode=DK"
	// var args = []Pair{{"countryCode", "DK"}, {"include", "tracks,albums"}}
	// testEndpoint("https://openapi.tidal.com/"+endpoint, args)

	// searchTrack := Track{
	// 	"",
	// 	"Caribe",
	// 	"Michel Camilo",
	// 	"Michel Camilo",
	// 	"",
	// 	9,
	// 	1,
	// }
}

func artistAlbumLookup(api TidalApi, searchTrack Track) string {
	searchString := cleanSearchString(searchTrack.Artist)
	endpoint := api.Url + "/searchResults/" + searchString + "/relationships/topHits"
	req, err := http.NewRequest("GET", endpoint+"?countryCode=DK", nil)
	check(err)

	var result, _ = doRequestWithRetry(api.Client, req, false)
	data, _ := result["data"].([]any)
	for i := range len(data) {
		item, _ := data[i].(map[string]any)
		if item["type"].(string) == "artists" {
			return findAlbumForArtist(api, searchTrack, item["id"].(string))
		}
	}
	return ""
}

func findAlbumForArtist(api TidalApi, searchTrack Track, artistId string) string {
	endpoint := api.Url + "/artists/" + artistId + "/relationships/albums"
	var req, err = http.NewRequest("GET", endpoint+"?countryCode=DK&include=albums", nil)
	check(err)

	var searchCount = 0
	for searchCount < 3 {
		searchCount++
		result, _ := doRequestWithRetry(api.Client, req, false)
		included, _ := result["included"].([]any)
		for i := range len(included) {
			item, _ := included[i].(map[string]any)
			title, _ := item["attributes"].(map[string]any)["title"].(string)
			if stringMatch(title, searchTrack.Album) {
				log.Println("Album match!")
				return item["id"].(string)
			}
		}

		links, _ := result["links"].(map[string]any)
		newNext, ok := links["next"].(string)
		if ok && searchCount < 3 {
			log.Println("Next albums from: ", newNext)
			req, err = http.NewRequest("GET", api.Url+newNext, nil)
		} else {
			return ""
		}
	}
	return ""
}

func trackLookup(tidalApi TidalApi, searchTrack Track) string {
	var albumId = tidalApi.searchAlbum(searchTrack)
	var trackId string
	if albumId != "" {
		log.Println("Album lookup for: ", searchTrack)
		trackId = checkAlbum(tidalApi, searchTrack, albumId)
	}
	if trackId == "" {
		log.Println("Artist lookup for: ", searchTrack)
		albumId = artistAlbumLookup(tidalApi, searchTrack)
		trackId = checkAlbum(tidalApi, searchTrack, albumId)
	}
	if trackId == "" {
		log.Println("Track lookup for: ", searchTrack)
		trackId = tidalApi.searchTrack(searchTrack)
	}
	return trackId
}

func checkAlbum(tidalApi TidalApi, searchTrack Track, albumId string) string {
	trackIndexMap := make(map[string]DiscIndex)
	var albumTracks []string
	if searchTrack.Album != "" && albumId != "" {
		albumTracks = tidalApi.getAlbum(albumId, searchTrack, "", trackIndexMap)
	}

	tracks := tidalApi.getTracks(albumTracks, trackIndexMap)
	for _, track := range tracks {
		if track.Name == searchTrack.Name ||
			(track.TrackNumber == searchTrack.TrackNumber && track.DiscNumber == searchTrack.DiscNumber) {
			log.Println("Succes! Track: ", track)
			return track.Id
		}
	}
	return ""
}

func migrateSinglePlaylistToTidal(playlistId string, newPlaylistName string) {
	log.Printf("Started migrating playlist: %s to new playlist: %s", playlistId, newPlaylistName)
	spotifyApi := NewSpotifyApi()
	tidalApi := NewTidalApi()
	var tracks = spotifyApi.getPlaylistTracks(playlistId, 0)

	var trackIds []string
	var notFound []Track
	for i, track := range tracks {
		log.Println("Lookup for track: ", track, ", index: ", i)
		var id = trackLookup(tidalApi, track)
		if id == "" {
			notFound = append(notFound, track)
		} else {
			trackIds = append(trackIds, id)
		}
	}

	newPlaylistId := tidalApi.createPlaylist(newPlaylistName)
	log.Println("New Playlist ID:", newPlaylistId)

	sleep = 10000
	batchSize := 20
	for i := 0; i < len(trackIds); i += batchSize {
		batch := trackIds[i:min(i+batchSize, len(trackIds))]
		tidalApi.addTracks(newPlaylistId, batch)
	}
	log.Println("Not found: ", notFound)
}
