package main

import (
	"fmt"
	"os"

	"pogomon/consts"
	"pogomon/mvu"
	"pogomon/setup"
	"pogomon/utils"

	tea "charm.land/bubbletea/v2"
	"gorm.io/gorm"
)

func printErrExit(errs ...error) {
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "Error:: %+v\n", e)
	}
	os.Exit(1)
}

func main() {
	// TODO * fix dbFilePath for XDG and OS specific locations later
	_, err := utils.GetDataDirPath()
	if err != nil {
		printErrExit(err)
	}
	dbFilePath := consts.DBFILEPATH
	var gdb *gorm.DB = nil

	if !utils.FileExists(dbFilePath) {
		var errs []error
		// * Fetch Data From PokeAPI, Create SQLite DB, seeded with API Data
		gdb, errs = setup.FetchDataAndCreateDB(dbFilePath)
		if errs != nil || len(errs) > 0 {
			printErrExit(errs...)
		}
		// * Wait for terminal input
		fmt.Print("> ")
		fmt.Scanln()
	} else {
		var err error = nil
		gdb, err = setup.GetGormSqliteDB(dbFilePath)
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
