package models

import (
	"time"

	"gorm.io/gorm"
)

type Character struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Number    int       `json:"number"`
	Player    string    `gorm:"size:255" json:"player"`
	Name      string    `gorm:"size:255;uniqueIndex" json:"name"`
	Status    string    `gorm:"size:50;default:'Activo'" json:"status"`
	Species   string    `gorm:"size:100" json:"species"`
	Class     string    `gorm:"size:100" json:"class"`
	GuildName string    `gorm:"size:255" json:"guild_name"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DLUsage struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Date        time.Time `gorm:"index" json:"date"`
	CharacterID uint      `gorm:"index" json:"character_id"`
	Character   Character `gorm:"foreignKey:CharacterID" json:"character"`
	DLUsed      int       `json:"dl_used"`
	GoldChange  float64   `json:"gold_change"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Transaction struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Date        time.Time `gorm:"index" json:"date"`
	CharacterID uint      `gorm:"index" json:"character_id"`
	Character   Character `gorm:"foreignKey:CharacterID" json:"character"`
	Amount      float64   `json:"amount"`
	Notes       string    `gorm:"type:text" json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
}

type CostOfLiving struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Date      time.Time `gorm:"index" json:"date"`
	Inn       float64   `json:"inn"`
	Guardiana float64   `json:"guardiana"`
	PlumaNegra float64  `json:"pluma_negra"`
	HijosAlba float64   `json:"hijos_alba"`
	CreatedAt time.Time `json:"created_at"`
}

type CharacterRegistry struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Date        time.Time `gorm:"index" json:"date"`
	CharacterID uint      `gorm:"index" json:"character_id"`
	Character   Character `gorm:"foreignKey:CharacterID" json:"character"`
	Event       string    `gorm:"size:255" json:"event"`
	Experience  float64   `json:"experience"`
	Gold        float64   `json:"gold"`
	Renown      float64   `json:"renown"`
	Notes       string    `gorm:"type:text" json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
}

type Mission struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Date       time.Time      `gorm:"index" json:"date"`
	DM         string         `gorm:"size:255" json:"dm"`
	Name       string         `gorm:"size:255" json:"name"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	CreatedAt  time.Time      `json:"created_at"`
	Entries    []MissionEntry `gorm:"foreignKey:MissionID" json:"entries"`
}

type MissionEntry struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	MissionID     uint           `gorm:"index" json:"mission_id"`
	Mission       Mission        `gorm:"foreignKey:MissionID" json:"mission"`
	CharacterID   uint           `gorm:"index" json:"character_id"`
	Character     Character      `gorm:"foreignKey:CharacterID" json:"character"`
	XPMission     float64        `json:"xp_mission"`
	XPReport      float64        `json:"xp_report"`
	XPGuild       float64        `json:"xp_guild"`
	Gold          float64        `json:"gold"`
	Renown        float64        `json:"renown"`
	Notes         string         `gorm:"type:text" json:"notes"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	CreatedAt     time.Time      `json:"created_at"`
}

type Guild struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"size:255;uniqueIndex" json:"name"`
	LeaderID        *uint     `json:"leader_id"`
	Leader          *Character `gorm:"foreignKey:LeaderID" json:"leader"`
	MemberIDs       string    `gorm:"type:text" json:"-"`
	HallType        string    `gorm:"size:100" json:"hall_type"`
	Notes           string    `gorm:"type:text" json:"notes"`
	CostOfLiving    float64   `json:"cost_of_living"`
	Treasury        float64   `json:"treasury"`
	RegisteredAt    *time.Time `json:"registered_at"`
	ApprovedAt      *time.Time `json:"approved_at"`
	Members         []Character `gorm:"many2many:guild_members;" json:"members"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
