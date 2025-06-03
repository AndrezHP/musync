package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/oauth2"
)

type TidalApi struct {
	Client *http.Client
	Url    string
}

func NewTidalApi() TidalApi {
	clientParams := readClientParams(".tidalParams.json")
	scopes := []string{"user.read", "collection.read", "collection.write", "playlists.read", "playlists.write"}
	apiUrl := "https://openapi.tidal.com/v2"
	config := &oauth2.Config{
		ClientID:     clientParams.ClientId,
		ClientSecret: clientParams.ClientSecret,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.tidal.com/authorize",
			TokenURL: "https://auth.tidal.com/v1/oauth2/token",
		},
		RedirectURL: "http://localhost:8081/callback",
	}

	token := getToken(config, context.Background(), apiUrl, ".tidalToken.json", "8081", true)
	client := config.Client(context.Background(), token)
	return TidalApi{
		client,
		apiUrl,
	}
}

func (api TidalApi) GetCurrentUserId() string {
	req, err := http.NewRequest("GET", api.Url+"/users/me", nil)
	Check(err)
	result, _ := DoRequest(api.Client, req, false)
	return JsonWrapper{result}.get("data").getString("id")
}

func (api TidalApi) GetUserPlaylists(userId string, next string) []Playlist {
	var request *http.Request
	if next == "" {
		endpoint := api.Url + fmt.Sprintf("/playlists?filter[r.owners.id]=%s", userId)
		req, err := http.NewRequest("GET", endpoint+"&countryCode=DK", nil)
		Check(err)
		request = req
	} else {
		req, err := http.NewRequest("GET", api.Url+next, nil)
		Check(err)
		request = req
	}

	result, _ := DoRequest(api.Client, request, false)
	resultJson := JsonWrapper{result}
	var playlists []Playlist

	data := resultJson.getSlice("data")
	for i := range len(data) {
		item := makeJson(data[i])
		id := item.getString("id")
		name := item.get("attributes").getString("name")
		length := item.get("attributes").getInt("numberOfItems")
		playlists = append(playlists, Playlist{id, name, int(length)})
	}

	links := resultJson.get("links")
	newNext, ok := links.content["next"].(string)
	if ok {
		log.Println("Fetching next playlists from:", newNext)
		return append(playlists, api.GetUserPlaylists(userId, newNext)...)
	} else {
		return playlists
	}
}

func (api TidalApi) GetPlaylistTracks(playlistId string, next string) []Track {
	var request *http.Request
	if next == "" {
		endpoint := api.Url + fmt.Sprintf("/playlists/%s/relationships/items", playlistId)
		req, err := http.NewRequest("GET", endpoint+"countryCode=DK", nil)
		Check(err)
		request = req
	} else {
		req, err := http.NewRequest("GET", api.Url+next, nil)
		Check(err)
		request = req
	}

	result, _ := DoRequest(api.Client, request, false)
	resultJson := JsonWrapper{result}

	data := resultJson.getSlice("data")
	var trackIds []string
	for i := range len(data) {
		id := makeJson(data[i]).getString("id")
		trackIds = append(trackIds, id)
	}

	tracks := api.GetTracks(trackIds, nil)
	links := resultJson.get("links")
	newNext, ok := links.content["next"].(string)
	if ok {
		log.Println("Fetching next tracks from:", newNext)
		return append(tracks, api.GetPlaylistTracks(playlistId, newNext)...)
	} else {
		return tracks
	}
}

func (api TidalApi) GetTracks(trackIds []string, trackIndexMap map[string]DiscIndex) []Track {
	var tracks []Track
	batchSize := 20
	for i := 0; i < len(trackIds); i += batchSize {
		batch := trackIds[i:min(i+batchSize, len(trackIds))]
		tracks = append(tracks, api.GetTracksBatch(batch, trackIndexMap)...)
	}
	return tracks
}

func (api TidalApi) GetTracksBatch(trackIds []string, trackIndexMap map[string]DiscIndex) []Track {
	req, err := http.NewRequest("GET", api.Url+"/tracks", nil)
	Check(err)
	params := req.URL.Query()
	params.Set("countryCode", "DK")
	params.Set("include", "albums,artists,tracks")
	var combinedIds string
	for i, id := range trackIds {
		if i == 0 {
			combinedIds = combinedIds + id
		} else {
			combinedIds = combinedIds + "," + id
		}
	}
	params.Set("filter[id]", combinedIds)
	req.URL.RawQuery = params.Encode()

	result, _ := DoRequest(api.Client, req, false)
	resultJson := JsonWrapper{result}

	albumMap := make(map[string]string)
	artistMap := make(map[string]string)

	included := resultJson.getSlice("included")
	for i := range len(included) {
		item := makeJson(included[i])
		itemId := item.getString("id")
		attributes := item.get("attributes")

		itemType := item.getString("type")
		if itemType == "albums" {
			albumMap[itemId] = attributes.getString("title")
		} else if itemType == "artists" {
			artistMap[itemId] = attributes.getString("name")
		} else {
			panic("Unknown item type: " + itemType)
		}
	}

	var tracks []Track
	data := resultJson.getSlice("data")
	for i := range len(data) {
		item := makeJson(data[i])
		trackId := item.getString("id")
		trackTitle := item.get("attributes").getString("title")
		version := item.get("attributes").getString("version")

		relationships := item.get("relationships")
		albums := relationships.get("albums")
		albumId := albums.getAt("data", 0).getString("id")

		artists := relationships.get("artists")
		artistId := artists.getAt("data", 0).getString("id")

		var trackNumber = 0
		var discNumber = 0
		discIndex, ok := trackIndexMap[trackId]
		if ok {
			trackNumber = discIndex.TrackNumber
			discNumber = discIndex.DiscNumber
		}

		track := Track{
			trackId,
			cleanTitle(trackTitle),
			version,
			artistMap[artistId],
			"",
			albumMap[albumId],
			albumId,
			trackNumber,
			discNumber,
		}
		tracks = append(tracks, track)
	}

	return tracks
}

func (api TidalApi) TrackLookup(track Track) string {
	log.Println("Album lookup for:", track)
	var albumId = api.SearchAlbum(track, track.Album+" "+cleanArtistName(track.Artist))
	var trackId = api.findTrackOnAlbum(track, albumId)
	if trackId != "" {
		log.Println("Succes:1")
	}

	if trackId == "" {
		log.Println("Track lookup for:", track.Name, cleanArtistName(track.Artist))
		trackId = api.SearchTrack(track, track.Name+" "+cleanArtistName(track.Artist))
		if trackId != "" {
			log.Println("Succes:2")
		}
	}

	if trackId == "" {
		partialArtistName := getPartialName(cleanArtistName(track.Artist))
		log.Println("Track lookup for:", track.Name, partialArtistName)
		trackId = api.SearchTrack(track, track.Name+" "+partialArtistName)
		if trackId != "" {
			log.Println("Succes:3")
		}
	}
	return trackId
}

func (api TidalApi) AddTracks(playlistId string, trackIds []string) {
	endpoint := api.Url + fmt.Sprintf("/playlists/%s/relationships/items", playlistId)
	var tracksToAdd []any
	for _, id := range trackIds {
		trackToAdd := map[string]any{
			"id":   id,
			"type": "tracks",
		}
		tracksToAdd = append(tracksToAdd, trackToAdd)
	}
	payload := map[string]any{
		"data": tracksToAdd,
	}
	jsonData, err := json.Marshal(payload)
	Check(err)

	req, err := http.NewRequest("POST", endpoint+"?countryCode=DK", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	Check(err)
	log.Println("Adding tracks with request:", req)

	_, response := DoRequest(api.Client, req, false)
	if response.StatusCode != 201 {
		log.Println("Could not add tracks:", response)
	}
}

func (api TidalApi) CreatePlaylist(name string) string {
	endpoint := api.Url + "/playlists"
	payload := map[string]any{
		"data": map[string]any{
			"type": "playlists",
			"attributes": map[string]any{
				"name":        name,
				"description": "",
				"privacy":     "PRIVATE",
			},
		},
	}
	jsonData, err := json.Marshal(payload)
	Check(err)

	req, err := http.NewRequest("POST", endpoint+"?countryCode=DK", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	Check(err)

	result, _ := DoRequest(api.Client, req, false)
	data := JsonWrapper{result}.get("data")
	if id, ok := data.content["id"].(string); ok {
		return id
	} else {
		panic("Id of new playlist not found")
	}
}

func (api TidalApi) SearchTrack(track Track, searchString string) string {
	clean := cleanString(searchString)
	endpoint := api.Url + fmt.Sprintf("/searchResults/%s/relationships/tracks", clean)
	req, err := http.NewRequest("GET", endpoint+"?countryCode=DK&include=tracks", nil)
	Check(err)
	result, _ := DoRequest(api.Client, req, false)
	resultJson := JsonWrapper{result}

	titleMap := make(map[string]string)
	versionMap := make(map[string]string)
	included := resultJson.getSlice("included")
	for i := range len(included) {
		item := makeJson(included[i])
		title := cleanTitle(item.get("attributes").getString("title"))
		version := item.get("attributes").getString("version")

		remix := regexp.MustCompile(`(?i)\sremix\s`)
		if remix.MatchString(title+" "+version) && !remix.MatchString(track.Name) {
			titleMap[item.getString("id")] = ""
			versionMap[item.getString("id")] = ""
		} else {
			titleMap[item.getString("id")] = cleanTitle(title)
			versionMap[item.getString("id")] = cleanTitle(version)
		}
	}

	// Makes sure to look at the included tracks in the order it was given
	data := resultJson.getSlice("data")
	for i := range len(data) {
		id := makeJson(data[i]).getString("id")
		title := titleMap[id]
		titleWithVersion := title + " " + versionMap[id]
		if title != "" && approximateMatch(title, track.Name, 0.2) {
			return id
		} else if approximateMatch(titleWithVersion, track.Name, 0.2) {
			return id
		}
	}
	return ""
}

func (api TidalApi) SearchAlbum(track Track, searchString string) string {
	clean := cleanString(searchString)
	endpoint := api.Url + fmt.Sprintf("/searchResults/%s/relationships/albums", clean)
	req, err := http.NewRequest("GET", endpoint+"?countryCode=DK&include=albums", nil)
	Check(err)

	var result, _ = DoRequest(api.Client, req, false)
	included := JsonWrapper{result}.getSlice("included")
	for i := range len(included) {
		item := makeJson(included[i])
		title := item.get("attributes").getString("title")
		if stringMatch(title, track.Album) {
			return item.getString("id")
		}
	}
	return ""
}

type DiscIndex struct {
	TrackNumber int
	DiscNumber  int
}

func (api TidalApi) GetAlbum(albumId string, searchTrack Track, next string, trackIndexMap map[string]DiscIndex) []string {
	var req *http.Request
	var err error
	if next == "" {
		req, err = http.NewRequest("GET", api.Url+"/albums/"+albumId, nil)
		params := req.URL.Query()
		params.Set("countryCode", "DK")
		params.Set("include", "items,artists")
		req.URL.RawQuery = params.Encode()
	} else {
		req, err = http.NewRequest("GET", api.Url+next+"&include=albums", nil)
	}

	Check(err)
	result, _ := DoRequest(api.Client, req, false)
	resultJson := JsonWrapper{result}

	if next == "" {
		included := resultJson.getSlice("included")
		var artistMatchFound = false
		var artists []string
		for i := range len(included) {
			item := makeJson(included[i])
			attributes := item.get("attributes")
			if item.getString("type") == "artists" {
				artist := attributes.getString("name")
				artists = append(artists, artist)
				artistMatchFound = stringMatch(artist, searchTrack.Artist)
			}
			if artistMatchFound {
				break
			}
		}

		if !artistMatchFound {
			log.Println("Album artist match not found!", searchTrack.Artist, artists)
			return nil
		}
	}

	var links map[string]any
	var data []any
	if next == "" {
		relationships := resultJson.get("data").get("relationships")
		data = relationships.get("items").getSlice("data")
		links = relationships.get("items").get("links").content
	} else {
		data = resultJson.getSlice("data")
		links = resultJson.get("links").content
	}

	var trackIds []string
	for i := range len(data) {
		item := makeJson(data[i])
		id := item.getString("id")
		trackNumber := item.get("meta").getInt("trackNumber")
		discNumber := item.get("meta").getInt("volumeNumber")

		trackIndexMap[id] = DiscIndex{int(trackNumber), int(discNumber)}
		trackIds = append(trackIds, id)
	}

	newNext, ok := links["next"].(string)
	if ok {
		fmt.Println("Fetching next tracks on album")
		nextTrackIds := api.GetAlbum(albumId, searchTrack, newNext, trackIndexMap)
		return append(trackIds, nextTrackIds...)
	} else {
		return trackIds
	}
}

func (api TidalApi) findTrackOnAlbum(searchTrack Track, albumId string) string {
	trackIndexMap := make(map[string]DiscIndex)
	var albumTracks []string
	log.Println("Get album for:", searchTrack, ", with album id:", albumId)
	if searchTrack.Album != "" && albumId != "" {
		albumTracks = api.GetAlbum(albumId, searchTrack, "", trackIndexMap)
	}

	tracks := api.GetTracks(albumTracks, trackIndexMap)

	var bestSimilarity = 0.0
	var bestId string
	for _, track := range tracks {
		trackName := track.Name + " " + track.Version
		var similarity = similarity(trackName, searchTrack.Name)
		log.Println("Similarity for:", trackName, "and", searchTrack.Name, "=", similarity)
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

func getPartialName(name string) string {
	filtered := regexp.MustCompile(`(?i)(the\s)`).ReplaceAllString(name, " ")
	splitName := strings.Split(strings.TrimSpace(filtered), " ")
	partialSplit := splitName[:(min(2, len(splitName)))]
	return strings.Join(partialSplit, " ")
}
