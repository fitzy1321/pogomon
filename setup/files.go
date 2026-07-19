package setup

import (
	"encoding/gob"
	"os"
)

// func checkForGobExtension(fpath string) error {
// 	gobext := filepath.Ext(fpath)
// 	if gobext != ".gob" {
// 		return fmt.Errorf("ERROR:::File path provided is not a gob file.\nRejeected Filepath: %s", fpath)
// 	}
// 	return nil
// }

func SaveGobFile[T any](data []T, fpath string) error {
	// if err := checkForGobExtension(fpath); err != nil {
	// 	return err
	// }

	file, err := os.Create(fpath)
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

func LoadGobFile[T any](fpath string) ([]T, error) {
	// if err := checkForGobExtension(fpath); err != nil {
	// 	return nil, err
	// }
	var data []T
	file, err := os.Open(fpath)
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
