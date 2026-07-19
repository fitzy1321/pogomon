package store

import (
	"gorm.io/gorm"
)

func GetPokemon(db *gorm.DB) ([]Pokemon, error) {
	var pokedex []Pokemon
	result := db.Find(&pokedex)
	if result.Error != nil {
		return nil, result.Error
	}
	// fmt.Printf("Debugging db stuff %d\n", len(pokedex))
	return pokedex, nil
}
