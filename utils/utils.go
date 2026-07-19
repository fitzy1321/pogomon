package utils

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var GoModAppName string

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		GoModAppName = "pogomon"
	}
	GoModAppName = path.Base(info.Main.Path)
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}

	if os.IsNotExist(err) {
		return false
	}

	return false
}

func ToTitle(str string) string {
	return cases.Title(language.Und, cases.NoLower).String(str)
}

func GetDataDirPath() (string, error) {
	// 1. Check XDG_DATA_HOME first
	dataHome, ok := os.LookupEnv("XDG_DATA_HOME")

	// 2. Fall back to OS-specific defaults
	if !ok || dataHome == "" {
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
	appDir := filepath.Join(dataHome, GoModAppName)
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		return "", fmt.Errorf("creating app data dir: %w", err)
	}
	return appDir, nil
}
