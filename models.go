package main

import ()

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
