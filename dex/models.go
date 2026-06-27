package dex

// WARN: gorm db structs, do not change without changing sql tables too

type (
	Pokemon struct {
		ID             uint   `gorm:"primaryKey;autoIncrement:false"`
		Name           string `gorm:"uniqueIndex;not null"`
		Type1          string `gorm:"not null"`
		Type2          *string
		HP             int `gorm:"not null"`
		Attack         int `gorm:"not null"`
		Defense        int `gorm:"not null"`
		SpAttack       int `gorm:"not null"`
		SpDefense      int `gorm:"not null"`
		Speed          int `gorm:"not null"`
		BaseExperience *int
		GrowthRate     *string
		FrontSprite    []byte
		BackSprite     []byte

		Moves      []PokemonMove
		Evolutions []Evolution `gorm:"foreignKey:PokemonID"`
	}

	Move struct {
		ID            uint   `gorm:"primaryKey;autoIncrement"`
		Name          string `gorm:"uniqueIndex;not null"`
		Power         *int
		Accuracy      *int
		MaxPP         int `gorm:"not null"`
		Type          *string
		DamageClass   *string
		Ailment       *string
		AilmentChance *int
		Category      *string
		Healing       *int
		Drain         *int
	}

	PokemonMove struct {
		ID           uint `gorm:"primaryKey;autoIncrement"`
		PokemonID    uint `gorm:"uniqueIndex:idx_pokemon_move;not null"`
		MoveID       uint `gorm:"uniqueIndex:idx_pokemon_move;not null"`
		LevelLearned int  `gorm:"not null"`
		LearnMethod  *string

		Pokemon Pokemon `gorm:"foreignKey:PokemonID"`
		Move    Move    `gorm:"foreignKey:MoveID"`
	}

	Evolution struct {
		ID             uint `gorm:"primaryKey;autoIncrement"`
		PokemonID      uint `gorm:"uniqueIndex:idx_evolution;not null"`
		EvolvesIntoID  uint `gorm:"uniqueIndex:idx_evolution;not null"`
		Trigger        *string
		MinLevel       *int
		Item           *string
		IsPlayerChoice bool `gorm:"default:0"`

		Pokemon     Pokemon `gorm:"foreignKey:PokemonID"`
		EvolvesInto Pokemon `gorm:"foreignKey:EvolvesIntoID"`
	}
)
