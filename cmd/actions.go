package cmd

import (
	"log"
)

func SpotifyPlaylistToTidal(playlistId string, newPlaylistName string) {
	log.Printf("Started migrating playlist: %s to new playlist: %s", playlistId, newPlaylistName)
	spotifyApi := NewSpotifyApi()
	tidalApi := NewTidalApi()
	var tracks = spotifyApi.GetPlaylistTracks(playlistId, 0)

	var trackIds []string
	var notFound []Track
	for i, track := range tracks {
		log.Println("Index:", i, "Lookup for track:", track)
		var id = tidalApi.TrackLookup(track)
		if id != "" && len(id) < 15 {
			trackIds = append(trackIds, id)
		} else {
			notFound = append(notFound, track)
		}
	}

	newPlaylistId := tidalApi.CreatePlaylist(newPlaylistName)
	log.Println("New Playlist ID:", newPlaylistId)

	previousSleep := Sleep()
	SetRequestSleep(7000)
	batchSize := 20
	for i := 0; i < len(trackIds); i += batchSize {
		batch := trackIds[i:min(i+batchSize, len(trackIds))]
		log.Println("Adding Tracks:", batchSize)
		tidalApi.AddTracks(newPlaylistId, batch)
	}
	SetRequestSleep(previousSleep)

	log.Println(len(notFound), "Not found:")
	for _, missing := range notFound {
		log.Println(missing.Name, "-", missing.Artist, "-", missing.Album)
	}
}
