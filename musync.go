package main

import (
	"github.com/AndrezHP/musync/cmd"
	"io"
	"log"
	"os"
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

func spotifyPlaylistToTidal(playlistId string, newPlaylistName string) {
	log.Printf("Started migrating playlist: %s to new playlist: %s", playlistId, newPlaylistName)
	spotifyApi := cmd.NewSpotifyApi()
	tidalApi := cmd.NewTidalApi()
	var tracks = spotifyApi.GetPlaylistTracks(playlistId, 0)

	var trackIds []string
	var notFound []cmd.Track
	for i, track := range tracks {
		log.Println("Index:", i, "Lookup for track:", track)
		var id = tidalApi.TrackLookup(track)
		if id == "" {
			notFound = append(notFound, track)
		} else {
			trackIds = append(trackIds, id)
		}
	}

	newPlaylistId := tidalApi.CreatePlaylist(newPlaylistName)
	log.Println("New Playlist ID:", newPlaylistId)

	previousSleep := cmd.Sleep()
	cmd.SetRequestSleep(7000)
	batchSize := 20
	for i := 0; i < len(trackIds); i += batchSize {
		batch := trackIds[i:min(i+batchSize, len(trackIds))]
		log.Println("Adding Tracks:", batchSize)
		tidalApi.AddTracks(newPlaylistId, batch)
	}
	cmd.SetRequestSleep(previousSleep)

	log.Println(len(notFound), "Not found:")
	for _, missing := range notFound {
		log.Println(missing.Name, "-", missing.Artist, "-", missing.Album)
	}
}

func startLogging() {
	file, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	multiWriter := io.MultiWriter(os.Stdout, file)
	log.SetOutput(multiWriter)
}
