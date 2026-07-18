package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"pogomon/mvu"
	"pogomon/setup"

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

func appName() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "go-pokebattle"
	}
	return path.Base(info.Main.Path)
}

func getDataDir() (string, error) {
	// 1. Check XDG_DATA_HOME first
	dataHome := os.Getenv("XDG_DATA_HOME")

	// 2. Fall back to OS-specific defaults
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home dir: %w", err)
		}
		switch runtime.GOOS {
		case "darwin":
			// MacOS data location: "~/Library/Application Support"
			dataHome = filepath.Join(home, "Library", "Application Support")
		// // not supporting windows right now, maybe later
		// case "windows":
		// 	dataHome = os.Getenv("APPDATA")
		// 	if dataHome == "" {
		// 		dataHome = filepath.Join(home, "AppData", "Roaming")
		// 	}
		default: // linux, bsd, etc.
			dataHome = filepath.Join(home, ".local", "share")
		}
	}

	// 3. Create the app folder
	appDir := filepath.Join(dataHome, appName())
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		return "", fmt.Errorf("creating app data dir: %w", err)
	}
	return appDir, nil
}

func main() {
	dbPath := "pokedata.db"
	var gdb *gorm.DB = nil

	if !setup.FileExists(dbPath) {
		var errs []error
		// * Fetch Data From PokeAPI, Create SQLite DB, seeded with API Data
		gdb, errs = setup.FetchDataAndCreateDB(dbPath, nil)
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
