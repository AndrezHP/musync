package main

import (
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func main() {
	startLogging()
	start := time.Now()

	migrateSinglePlaylistToTidal("", "")
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

	api := NewTidalApi()
	searchString := cleanSearchString("Unplugged (Deluxe Edition) Eric Clapton")
	log.Println(searchString)
	endpoint := "/searchResults/" + searchString + "/relationships/albums"
	testEndpoint(api.Url+endpoint, []Pair{{"countryCode", "DK"}, {"include", "albums"}})
}

var sleep float64 = 1500

type Playlist struct {
	Id     string
	Name   string
	Length int
}

type Track struct {
	Id          string
	Name        string
	Version     string
	Artist      string
	Album       string
	AlbumType   string
	AlbumId     string
	TrackNumber int
	DiscNumber  int
}

func artistAlbumLookup(api TidalApi, searchTrack Track) string {
	searchString := cleanSearchString(searchTrack.Artist)
	endpoint := api.Url + "/searchResults/" + searchString + "/relationships/topHits"
	req, err := http.NewRequest("GET", endpoint+"?countryCode=DK", nil)
	check(err)

	var result, _ = doRequestWithRetry(api.Client, req, false)
	data := JsonWrapper{result}.getSlice("data")
	for i := range len(data) {
		item := makeJson(data[i])
		if item.getString("type") == "artists" {
			return findAlbumForArtist(api, searchTrack, item.getString("id"))
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
		resultJson := JsonWrapper{result}
		included := resultJson.getSlice("included")
		for i := range len(included) {
			item := makeJson(included[i])
			title := item.get("attributes").getString("title")
			if stringMatch(title, searchTrack.Album) {
				log.Println("Album match!")
				return item.getString("id")
			}
		}

		links := resultJson.get("links")
		newNext, ok := links.content["next"].(string)
		if ok {
			log.Println("Next albums from:", newNext, ", search count:", searchCount)
			req, err = http.NewRequest("GET", api.Url+newNext, nil)
		} else {
			return ""
		}
	}
	return ""
}

func trackLookup(tidalApi TidalApi, track Track) string {
	log.Println("Album lookup for:", track)
	var albumId = tidalApi.searchAlbum(track, track.Album+" "+track.Artist)
	var trackId = checkAlbum(tidalApi, track, albumId)
	if trackId != "" {
		log.Println("Succes:1")
	}

	if trackId == "" {
		log.Println("Artist lookup for:", track)
		albumId := artistAlbumLookup(tidalApi, track)
		trackId = checkAlbum(tidalApi, track, albumId)
		if trackId != "" {
			log.Println("Succes:2")
		}
	}

	if trackId == "" {
		log.Println("Track lookup for:", track.Name, track.Artist)
		trackId = tidalApi.searchTrack(track, track.Name+" "+track.Artist)
		if trackId != "" {
			log.Println("Succes:3")
		}
	}

	if trackId == "" {
		log.Println("Track lookup for:", track.Name)
		regex := regexp.MustCompile(`(?i)(the\ )`)
		split := strings.Split(regex.ReplaceAllString(track.Artist, " "), " ")
		partial := split[:(min(2, len(split)))]
		trackId = tidalApi.searchTrack(track, strings.Join(partial, " "))
		if trackId != "" {
			log.Println("Succes:4")
		}
	}
	return trackId
}

func checkAlbum(tidalApi TidalApi, searchTrack Track, albumId string) string {
	trackIndexMap := make(map[string]DiscIndex)
	var albumTracks []string
	log.Println("Get album for:", searchTrack, ", with album id:", albumId)
	if searchTrack.Album != "" && albumId != "" {
		albumTracks = tidalApi.getAlbum(albumId, searchTrack, "", trackIndexMap)
	}

	tracks := tidalApi.getTracks(albumTracks, trackIndexMap)

	var bestSimilarity = 0.0
	var bestId string
	for _, track := range tracks {
		trackName := track.Name + " " + track.Version
		var similarity = similarity(trackName, searchTrack.Name)
		log.Println("Similarity for:", trackName, "and", searchTrack.Name, "=", similarity)
		// if track.TrackNumber == searchTrack.TrackNumber || track.DiscNumber == searchTrack.DiscNumber {
		//   // Is there any case where this might be useful?
		// }
		if similarity > bestSimilarity {
			bestSimilarity = similarity
			bestId = track.Id
		}
	}

	if bestSimilarity < 0.8 {
		return ""
	} else {
		return bestId
	}
}

func migrateSinglePlaylistToTidal(playlistId string, newPlaylistName string) {
	log.Printf("Started migrating playlist: %s to new playlist: %s", playlistId, newPlaylistName)
	spotifyApi := NewSpotifyApi()
	tidalApi := NewTidalApi()
	var tracks = spotifyApi.getPlaylistTracks(playlistId, 0)

	var trackIds []string
	var notFound []Track
	for i, track := range tracks {
		log.Println("Index:", i, "Lookup for track:", track)
		var id = trackLookup(tidalApi, track)
		if id == "" {
			notFound = append(notFound, track)
		} else {
			trackIds = append(trackIds, id)
		}
	}

	newPlaylistId := tidalApi.createPlaylist(newPlaylistName)
	log.Println("New Playlist ID:", newPlaylistId)

	sleep = 7000
	batchSize := 20
	for i := 0; i < len(trackIds); i += batchSize {
		batch := trackIds[i:min(i+batchSize, len(trackIds))]
		log.Println("Adding Tracks:", batchSize)
		tidalApi.addTracks(newPlaylistId, batch)
	}

	log.Println(len(notFound), "Not found:")
	for _, missing := range notFound {
		printTrack(missing)
	}
}
