package setup

type (
	PokeApiData struct {
		Id             uint
		Name           string
		Type1          string
		Type2          *string // nullable
		BaseExperience *int    // nullable
		Moves          []MoveData
		NextEvolutions []nextEvoData
		GrowthRate     *string // nullable
		Sprites        sprites
		stats
	}

	MoveData struct {
		Name          string
		LevelLearned  uint
		LearnMethod   string
		MaxPp         int
		Power         *int         // nullable
		Accuracy      *int         // nullable
		Type          *string      // TODO: should this be nullable?
		DamageClass   *string      // nullable
		Ailment       *string      // nullable
		AilmentChance *int         // nullable
		MoveCategory  *string      // nullable
		Healing       *int         // nullable
		Drain         *int         // nullable
		StatChanges   []statChange // TODO: maybe nullable?

	}

	stats struct {
		Attack         int
		Defense        int
		Hp             int
		SpecialAttack  int
		SpecialDefense int
		Speed          int
	}

	statChange struct {
		Stat   string
		Change any // TODO: check type
	}

	nextEvoData struct {
		EvolvesIntoId uint
		Trigger       string
		MinLevel      uint
		Item          *string // nullable
	}

	sprites struct {
		front, back []byte
	}

	_mvIR struct {
		name   string
		level  int
		url    string
		method string
	}
)
