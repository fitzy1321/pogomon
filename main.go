package main

import (
	"fmt"
	"net/http"
	"os"

	"go-pokebattle/mvu"
	"go-pokebattle/setup"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

func toTitle(str string) string {
	return cases.Title(language.Und, cases.NoLower).String(str)
}

func printErrExit(errs ...error) {
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "Error:: %+v\n", e)
	}
	os.Exit(1)
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

	// * Setup bubbletea inital model ...
	model, err := mvu.NewAppModel(gdb)
	if err != nil {
		printErrExit(err)
	}
	p := tea.NewProgram(*model)

	// * Run Bubbletea app
	if _, err := p.Run(); err != nil {
		printErrExit(err)
	}
}
