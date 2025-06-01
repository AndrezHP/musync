package main

import (
	"log"
	"time"
)

func main() {
	startLogging()
	start := time.Now()

	// migrateSinglePlaylistToTidal("", "")
	spotifyPlaylistToTidal("", "Test")
	// test()

	end := time.Now()
	elapsed := end.Sub(start)
	log.Println("Elapsed:", time.Duration.Milliseconds(elapsed))
}

func test() {
	// api := NewTidalApi()
	// searchString := cleanSearchString("Toto TOTO")
	// log.Println(searchString)
	// endpoint := "/searchResults/" + searchString + "/relationships/albums"
	// testEndpoint(api.Url+endpoint, []Pair{{"countryCode", "DK"}, {"include", "albums"}})
}

var sleep float64 = 1500

func spotifyPlaylistToTidal(playlistId string, newPlaylistName string) {
	log.Printf("Started migrating playlist: %s to new playlist: %s", playlistId, newPlaylistName)
	spotifyApi := NewSpotifyApi()
	tidalApi := NewTidalApi()
	var tracks = spotifyApi.getPlaylistTracks(playlistId, 0)

	var trackIds []string
	var notFound []Track
	for i, track := range tracks {
		log.Println("Index:", i, "Lookup for track:", track)
		var id = tidalApi.trackLookup(track)
		if id == "" {
			notFound = append(notFound, track)
		} else {
			trackIds = append(trackIds, id)
		}
	}

	newPlaylistId := tidalApi.createPlaylist(newPlaylistName)
	log.Println("New Playlist ID:", newPlaylistId)

	previousSleep := sleep
	sleep = 7000
	batchSize := 20
	for i := 0; i < len(trackIds); i += batchSize {
		batch := trackIds[i:min(i+batchSize, len(trackIds))]
		log.Println("Adding Tracks:", batchSize)
		tidalApi.addTracks(newPlaylistId, batch)
	}
	sleep = previousSleep

	log.Println(len(notFound), "Not found:")
	for _, missing := range notFound {
		printTrack(missing)
	}
}
