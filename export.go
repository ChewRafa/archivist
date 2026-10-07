package main

import (
	"time"

	"gorm.io/gorm"
)

type GuildMember struct {
	GuildID     uint `json:"guild_id"`
	CharacterID uint `json:"character_id"`
}

type ExportDLUsage struct {
	ID          uint      `json:"id"`
	Date        time.Time `json:"date"`
	CharacterID uint      `json:"character_id"`
	DLUsed      int       `json:"dl_used"`
	GoldChange  float64   `json:"gold_change"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type ExportTransaction struct {
	ID          uint      `json:"id"`
	Date        time.Time `json:"date"`
	CharacterID uint      `json:"character_id"`
	Amount      float64   `json:"amount"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
}

type ExportCostOfLiving struct {
	ID          uint      `json:"id"`
	Date        time.Time `json:"date"`
	CharacterID uint      `json:"character_id"`
	Amount      float64   `json:"amount"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
}

type ExportCharacterRegistry struct {
	ID          uint      `json:"id"`
	Date        time.Time `json:"date"`
	CharacterID uint      `json:"character_id"`
	Event       string    `json:"event"`
	Experience  float64   `json:"experience"`
	Gold        float64   `json:"gold"`
	Renown      float64   `json:"renown"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
}

type ExportMission struct {
	ID        uint           `json:"id"`
	Date      time.Time      `json:"date"`
	DM        string         `json:"dm"`
	Name      string         `json:"name"`
	DeletedAt gorm.DeletedAt `json:"deleted_at"`
	CreatedAt time.Time      `json:"created_at"`
}

type ExportMissionEntry struct {
	ID          uint           `json:"id"`
	MissionID   uint           `json:"mission_id"`
	CharacterID uint           `json:"character_id"`
	XPMission   float64        `json:"xp_mission"`
	XPReport    float64        `json:"xp_report"`
	XPGuild     float64        `json:"xp_guild"`
	Gold        float64        `json:"gold"`
	Renown      float64        `json:"renown"`
	Notes       string         `json:"notes"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at"`
	CreatedAt   time.Time      `json:"created_at"`
}

type ExportGuild struct {
	ID           uint       `json:"id"`
	Name         string     `json:"name"`
	LeaderID     *uint      `json:"leader_id"`
	HallType     string     `json:"hall_type"`
	Notes        string     `json:"notes"`
	CostOfLiving float64    `json:"cost_of_living"`
	Treasury     float64    `json:"treasury"`
	RegisteredAt *time.Time `json:"registered_at"`
	ApprovedAt   *time.Time `json:"approved_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type ExportGuildTransaction struct {
	ID        uint      `json:"id"`
	Date      time.Time `json:"date"`
	GuildID   uint      `json:"guild_id"`
	Amount    float64   `json:"amount"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

type ExportData struct {
	ExportedAt        time.Time                  `json:"exported_at"`
	Version           string                     `json:"version"`
	Characters        []Character                `json:"characters"`
	DLUsages          []ExportDLUsage            `json:"dl_usages"`
	Transactions      []ExportTransaction        `json:"transactions"`
	CostOfLivings     []ExportCostOfLiving       `json:"cost_of_livings"`
	Registries        []ExportCharacterRegistry  `json:"character_registries"`
	Missions          []ExportMission            `json:"missions"`
	MissionEntries    []ExportMissionEntry       `json:"mission_entries"`
	Guilds            []ExportGuild              `json:"guilds"`
	GuildTransactions []ExportGuildTransaction   `json:"guild_transactions"`
	GuildMembers      []GuildMember              `json:"guild_members"`
	CharacterStats    []CharacterStats           `json:"character_stats"`
}

func ExportAll(db *gorm.DB) (*ExportData, error) {
	data := &ExportData{
		ExportedAt: time.Now(),
		Version:    Version,
	}

	if err := db.Unscoped().Order("id ASC").Find(&data.Characters).Error; err != nil {
		return nil, err
	}
	if err := db.Table("dl_usages").Order("id ASC").Scan(&data.DLUsages).Error; err != nil {
		return nil, err
	}
	if err := db.Table("transactions").Order("id ASC").Scan(&data.Transactions).Error; err != nil {
		return nil, err
	}
	if err := db.Table("cost_of_livings").Order("id ASC").Scan(&data.CostOfLivings).Error; err != nil {
		return nil, err
	}
	if err := db.Table("character_registries").Order("id ASC").Scan(&data.Registries).Error; err != nil {
		return nil, err
	}
	if err := db.Table("missions").Order("id ASC").Scan(&data.Missions).Error; err != nil {
		return nil, err
	}
	if err := db.Table("mission_entries").Order("id ASC").Scan(&data.MissionEntries).Error; err != nil {
		return nil, err
	}
	if err := db.Table("guilds").Order("id ASC").Scan(&data.Guilds).Error; err != nil {
		return nil, err
	}
	if err := db.Table("guild_transactions").Order("id ASC").Scan(&data.GuildTransactions).Error; err != nil {
		return nil, err
	}
	if err := db.Table("guild_members").Order("guild_id ASC, character_id ASC").Scan(&data.GuildMembers).Error; err != nil {
		return nil, err
	}
	if err := db.Table("character_stats").Find(&data.CharacterStats).Error; err != nil {
		return nil, err
	}

	return data, nil
}
