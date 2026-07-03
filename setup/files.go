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
