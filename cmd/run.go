package cmd

import (
	"fmt"
	"log"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func Run() {
	spotify := NewSpotifyApi()
	log.Println("Finding Spotify Playlists...")
	spotifyUser := spotify.GetCurrentUserId()
	spotifyPlaylists := spotify.GetUserPlaylists(spotifyUser, 0)

	tidal := NewTidalApi()
	log.Println("Finding Tidal playlists...")
	tidalUser := tidal.GetCurrentUserId()
	tidalPlaylists := tidal.GetUserPlaylists(tidalUser, "")

	var playlists []Playlist
	for _, playlist := range spotifyPlaylists {
		var existsAlready = false
		for _, cmp := range tidalPlaylists {
			if playlist.Name == cmp.Name {
				existsAlready = true
				break
			}
		}
		if !existsAlready {
			playlists = append(playlists, playlist)
		}
	}

	model := initialModel(playlists)
	_, err := tea.NewProgram(model).Run()
	Check(err)

	for i := range model.selected {
		playlist := model.choices[i]
		SpotifyPlaylistToTidal(playlist.Id, playlist.Name)
	}
}

func (m model) View() string {
	s := "Pick a playlist to migrate to tidal:\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}

		s += fmt.Sprintf("%s [%s] (%s) %s\n", cursor, checked, strconv.Itoa(choice.Length), choice.Name)
	}
	return s + "\n| q - quit | C-a - select all | SPC - select | ENTER - start |\n"
}

func initialModel(playlists []Playlist) model {
	return model{
		choices:  playlists,
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

type model struct {
	choices  []Playlist
	cursor   int
	selected map[int]struct{}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.selected = nil
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		case "ctrl+a":
			if len(m.selected) == len(m.choices) {
				m.selected = make(map[int]struct{})
			} else {
				for i := range len(m.choices) {
					m.selected[i] = struct{}{}
				}
			}
		case "enter":
			return m, tea.Quit
		}
	}

	return m, nil
}
