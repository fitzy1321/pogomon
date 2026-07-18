package mvu

import (
	"fmt"
	"time"

	"pogomon/sqlmodels"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/key"
	"gorm.io/gorm"
)

// fields here should be related to view, render, update states
type (
	// * Top Level Bubbletea Model
	AppModel struct {
		width, height int
		viewState     viewState
		internalAppState
	}

	internalAppState struct {
		DB          *gorm.DB
		saveFiles   []saveFileStart
		currentFile *sqlmodels.UserSave
	}

	saveFileStart struct {
		ID   uint
		Name string
	}

	keyMap struct {
		Enter key.Binding
		// Back  key.Binding
		Quit key.Binding
	}

	tickMsg          struct{}
	titleTickDoneMsg struct{}
)

func NewAppModel(db *gorm.DB) (*AppModel, error) {
	var saveFileStarts []saveFileStart
	result := db.Model(&sqlmodels.UserSave{}).Select("id", "name").Scan(&saveFileStarts)
	if result.Error != nil {
		return nil, fmt.Errorf("There was a problem loading save files: %+v\n", result.Error)
	}

	return &AppModel{
		width: 0, height: 0,
		viewState: titleView,
		internalAppState: internalAppState{DB: db,
			saveFiles:   saveFileStarts,
			currentFile: nil,
		},
	}, nil
}

// Gets called once "on start", I think
func (m AppModel) Init() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return titleTickDoneMsg{}
	})
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch t := msg.(type) {
	case titleTickDoneMsg:
		if m.viewState == titleView {
			m.viewState = saveFileOrNewView
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width, m.height = t.Width, t.Height
	case tea.KeyMsg:
		switch {
		// case key.Matches(t, keys.Enter):
		case key.Matches(t, keys.Quit, keys.Enter):
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m AppModel) View() tea.View {
	var view tea.View
	if m.viewState == titleView {
		view = tea.NewView("Welcome to Pokebattle TUI!")
	}
	if m.viewState == saveFileOrNewView {
		view = tea.NewView("I have no idea what I'm doing ...\n  press q or ctrl+c to quit, I guess ...\n")
	}

	return view
}

var keys = keyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	// Back: key.NewBinding(
	// 	key.WithKeys("esc", "backspace"),
	// 	key.WithHelp("esc", "back"),
	// ),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c", "ctrl+d"),
		key.WithHelp("q", "quit"),
	),
}

type viewState int

const (
	titleView viewState = iota
	saveFileOrNewView
)
