package main

import (
	"fmt"
	"net/http"
	"os"

	"go-pokebattle/setup"
	"go-pokebattle/sqlmodels"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/key"
	"gorm.io/gorm"
)

type keyMap struct {
	Enter key.Binding
	Back  key.Binding
	Quit  key.Binding
}

var keys = keyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"), // both trigger the same action
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c", "ctrl+d"),
		key.WithHelp("q", "quit"),
	),
}

type appState int

const (
	titleScreen appState = iota
	saveOrCreateFile
)

type AppModel struct {
	DB                  *gorm.DB
	saveFiles           []saveFileStart
	currentFile         *sqlmodels.SaveFile
	state               appState
	tuiWidth, tuiHeight uint
}

func (m AppModel) Init() tea.Cmd {
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch t := msg.(type) {
	case tea.WindowSizeMsg:
		m.tuiWidth, m.tuiHeight = uint(t.Width), uint(t.Height)
	case tea.KeyMsg:
		switch {
		case key.Matches(t, keys.Enter):
		case key.Matches(t, keys.Quit):
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m AppModel) View() tea.View {
	var view tea.View
	if m.state == titleScreen {
		view = tea.NewView("Welcome to Pokebattle TUI!")
	}
	return view
}

func printErrExit(errs ...error) {
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "Error:: %+v\n", e)
	}
	os.Exit(1)

}

type saveFileStart struct {
	ID   uint
	Name string
}

func main() {
	dbPath := "pokedata.db"
	var gdb *gorm.DB = nil

	if !setup.FileExists(dbPath) {
		var errs []error
		// * Fetch Data From PokeAPI, Create SQLite DB, seeded with API Data
		gdb, errs = setup.FetchDataAndCreateDB(dbPath, http.DefaultClient)
		if errs != nil || len(errs) > 0 {
			printErrExit(errs...)
		}
		// * Wait for terminal input
		fmt.Print("> ")
		fmt.Scanln()
	} else {
		var err error = nil
		gdb, err = setup.GetGormSqliteDB(dbPath)
		if err != nil {
			printErrExit(fmt.Errorf("Something failed connecting to pokemon db: %v\n", err))
		}
	}

	var saveFileStarts []saveFileStart
	result := gdb.Model(&sqlmodels.SaveFile{}).Select("id", "name").Scan(&saveFileStarts)
	if result.Error != nil {
		printErrExit(fmt.Errorf("There was a problem loading save files: %+v\n", result.Error))
	}

	// * Setup bubbletea inital model ...
	p := tea.NewProgram(AppModel{
		DB:          gdb,
		saveFiles:   saveFileStarts,
		currentFile: nil,
		state:       titleScreen,
		tuiWidth:    0,
		tuiHeight:   0,
	})

	// * Run Bubbletea app
	if _, err := p.Run(); err != nil {
		printErrExit(err)
	}
}
