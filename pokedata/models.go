package pokedata

// WARN: gorm struct, do not change the member names
type Pokemon struct {
	Id              uint `gorm:"primaryKey"`
	Name            string
	Type_1          string
	Type_2          *string
	Base_hp         uint
	Base_attack     uint
	Base_defense    uint
	Base_sp_attack  uint
	Base_sp_defense uint
	Base_speed      uint
	Base_experience *uint
	Growth_rate     *string
	Front_sprite    []byte
	Back_sprite     []byte
}

func (Pokemon) TableName() string {
	return "dex_pokemon"
}

// WARN: gorm struct, do not change the member names
type Move struct {
	Id             uint `gorm:"primaryKey"`
	Name           string
	Power          *uint
	Accuracy       *uint
	Max_pp         uint
	Type           *string
	Damage_class   *string
	Ailment        *string
	Ailment_chance *uint
	Move_category  *string
	Healing        *uint
	Drain          *int
}

func (Move) TableName() string {
	return "dex_move"
}
