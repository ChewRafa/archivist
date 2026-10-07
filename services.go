package main

import (
	"gorm.io/gorm"
)

var XPThresholds = []float64{
	0, 300, 900, 2700, 6500, 14000, 23000, 34000, 48000, 64000,
	85000, 100000, 120000, 140000, 165000, 195000, 225000, 265000, 305000, 355000,
}

func CalculateLevel(xp float64) int {
	level := 1
	for i := len(XPThresholds) - 1; i >= 0; i-- {
		if xp >= XPThresholds[i] {
			level = i + 1
			break
		}
	}
	if level > 20 {
		level = 20
	}
	return level
}

func GetGoldBalance(characterID uint) float64 {
	var total float64

	DB.Model(&CharacterRegistry{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(gold), 0)").
		Scan(&total)

	var missionGold float64
	DB.Model(&MissionEntry{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(gold), 0)").
		Scan(&missionGold)
	total += missionGold

	var usageGold float64
	DB.Model(&DLUsage{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(gold_change), 0)").
		Scan(&usageGold)
	total += usageGold

	var txGold float64
	DB.Model(&Transaction{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&txGold)
	total += txGold

	var colGold float64
	DB.Model(&CostOfLiving{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&colGold)
	total += colGold

	return total
}

func GetRenownTotal(characterID uint) float64 {
	var total float64

	DB.Model(&CharacterRegistry{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(renown), 0)").
		Scan(&total)

	var missionRenown float64
	DB.Model(&MissionEntry{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(renown), 0)").
		Scan(&missionRenown)
	total += missionRenown

	return total
}

func GetXPTotal(characterID uint) float64 {
	var total float64

	DB.Model(&CharacterRegistry{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(experience), 0)").
		Scan(&total)

	var missionXP float64
	DB.Model(&MissionEntry{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(xp_mission + xp_report + xp_guild), 0)").
		Scan(&missionXP)
	total += missionXP

	return total
}

type CharacterStats struct {
	Character
	XP          float64 `json:"xp"`
	Level       int     `json:"level"`
	GoldBalance float64 `json:"gold_balance"`
	Renown      float64 `json:"renown"`
}

func GetCharacterWithStats(id uint) (*CharacterStats, error) {
	var character Character
	if err := DB.First(&character, id).Error; err != nil {
		return nil, err
	}

	xp := GetXPTotal(id)

	return &CharacterStats{
		Character:   character,
		XP:          xp,
		Level:       CalculateLevel(xp),
		GoldBalance: GetGoldBalance(id),
		Renown:      GetRenownTotal(id),
	}, nil
}

// RefreshCharacterStatsView drops and recreates the character_stats view.
// Call this after any write operation that affects character stats.
func RefreshCharacterStatsView(tx *gorm.DB) error {
	if err := tx.Exec("DROP VIEW IF EXISTS character_stats").Error; err != nil {
		return err
	}
	viewSQL := `
		CREATE VIEW character_stats AS
		SELECT
			c.id,
			c.number,
			c.player,
			c.name,
			c.status,
			c.species,
			c.class,
			c.guild_name,
			c.guild_role,
			c.mount,
			c.created_at,
			c.updated_at,
			c.deleted_at,
			COALESCE((SELECT SUM(experience) FROM character_registries WHERE character_id = c.id), 0) +
			COALESCE((SELECT SUM(xp_mission + xp_report + xp_guild) FROM mission_entries WHERE character_id = c.id), 0) AS xp,
			COALESCE((SELECT SUM(gold) FROM character_registries WHERE character_id = c.id), 0) +
			COALESCE((SELECT SUM(gold) FROM mission_entries WHERE character_id = c.id), 0) +
			COALESCE((SELECT SUM(gold_change) FROM dl_usages WHERE character_id = c.id), 0) +
			COALESCE((SELECT SUM(amount) FROM transactions WHERE character_id = c.id), 0) +
			COALESCE((SELECT SUM(amount) FROM cost_of_livings WHERE character_id = c.id), 0) AS gold_balance,
			COALESCE((SELECT SUM(renown) FROM character_registries WHERE character_id = c.id), 0) +
			COALESCE((SELECT SUM(renown) FROM mission_entries WHERE character_id = c.id), 0) AS renown
		FROM characters c
		WHERE c.deleted_at IS NULL
	`
	return tx.Exec(viewSQL).Error
}

func GetAllCharactersWithStats() ([]CharacterStats, error) {
	var stats []CharacterStats
	if err := DB.Table("character_stats").Scan(&stats).Error; err != nil {
		return nil, err
	}
	for i := range stats {
		stats[i].Level = CalculateLevel(stats[i].XP)
	}
	return stats, nil
}
