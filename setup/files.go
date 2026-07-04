package setup

import (
	"encoding/gob"
	"os"
)

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

func SaveGobFile(data []PokeApiData, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := gob.NewEncoder(file)
	if err := enc.Encode(&data); err != nil {
		return err
	}
	return nil
}

func LoadGobFile(filePath string) ([]PokeApiData, error) {
	var data []PokeApiData
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	dec := gob.NewDecoder(file)
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

// TODO: Keep or delete this func?
// func osLevelStuff() error {
// 	home_path, ok := os.LookupEnv("HOME")
// 	if !ok {
// 		return fmt.Errorf("No Home ENV, something is wrong ...\n")

// 	}
// 	fmt.Println("Home path:", home_path)

// 	xdg_data := os.Getenv("XDG_DATA_HOME")
// 	fmt.Println("idk if this is real? :", xdg_data)

// 	xdg_config := os.Getenv("XDG_CONFIG_HOME")
// 	fmt.Println("XDG_CONFIG_HOME:", xdg_config)

// 	osname := runtime.GOOS
// 	switch osname {
// 	case "windows":
// 		fmt.Println("Windows specific stuff")
// 	case "darwin":
// 		fmt.Println("MacOS stuff")
// 	case "linux":
// 		fmt.Println("linux stuff")
// 	default:
// 		fmt.Println("I have no idea what you're on ...")
// 	}

// 	return nil
// }
