package cmd

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func Run() {
	spotify := NewSpotifyApi()
	userId := spotify.GetCurrentUserId()
	playlists := spotify.GetUserPlaylists(userId, 0)

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
	return s + "\nPress q to quit.\n"
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
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected[m.cursor] = struct{}{}
			return m, tea.Quit
		}
	}

	return m, nil
}
