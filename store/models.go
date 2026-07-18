package store

// NOTE: thoughts about integer typing below
// NOTE: IDs = uints, because we'll never have a negative id value.
// NOTE: Any other integer = ints, various moves and pokemon information could have negative values.
// NOTE: A pointer means that field/column is nullable
// WARN: gorm db structs, do not change without running migrations
type (
	UserSave struct {
		ID   uint   `gorm:"primarykey"`
		Name string `gorm:"not null"`

		PartyPokemon []PartyPokemon `gorm:"foreignKey:UserSaveID"`
	}

	PartyPokemon struct {
		ID           uint `gorm:"primaryKey"`
		UserSaveID   uint `gorm:"not null"`
		PokemonID    uint `gorm:"not null"`
		Nickname     *string
		Level        uint
		Experience   uint
		CurrentHP    uint
		StatusEffect *string
		PartySlot    uint // 1 - 4

		Pokemon Pokemon            `gorm:"foreignKey:PokemonID"`
		Moves   []PartyPokemonMove `gorm:"foreignKey:PartyPokemonID"`
	}

	PartyPokemonMove struct {
		ID             uint `gorm:"primaryKey"`
		PartyPokemonID uint
		MoveID         uint
		MoveSlot       uint // 1-4
		CurrentPP      uint

		Move Move `gorm:"foreignKey:MoveID"`
	}

	// * Static Models
	Pokemon struct {
		ID             uint   `gorm:"primaryKey;autoIncrement:false;not null"` // comes from api
		Name           string `gorm:"uniqueIndex:idx_pokemon_name;not null"`
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
		ID            uint   `gorm:"primaryKey;autoIncrement:false;not null"` // comes from api
		Name          string `gorm:"uniqueIndex:idx_move_name;not null"`
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
		ID           uint `gorm:"primaryKey;autoIncrement;not null"`
		PokemonID    uint `gorm:"uniqueIndex:idx_pokemon_move;not null"`
		MoveID       uint `gorm:"uniqueIndex:idx_pokemon_move;not null"`
		LevelLearned int  `gorm:"not null;default:0"`
		LearnMethod  *string

		Pokemon Pokemon `gorm:"foreignKey:PokemonID"`
		Move    Move    `gorm:"foreignKey:MoveID"`
	}

	Evolution struct {
		ID              uint `gorm:"primaryKey;autoIncrement;not null"`
		PokemonID       uint `gorm:"uniqueIndex:idx_evolution;not null"`
		EvolvesIntoID   uint `gorm:"uniqueIndex:idx_evolution;not null"`
		EvolvesIntoName *string
		Trigger         *string
		MinLevel        *int
		Item            *string
		IsPlayerChoice  bool `gorm:"default:0"`

		Pokemon     Pokemon `gorm:"foreignKey:PokemonID"`
		EvolvesInto Pokemon `gorm:"foreignKey:EvolvesIntoID"`
	}
)
