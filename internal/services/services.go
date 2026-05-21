package services

import (
	"codeberg.org/chewrafa/archivist/internal/db"
	"codeberg.org/chewrafa/archivist/internal/models"
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

	db.DB.Model(&models.CharacterRegistry{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(gold), 0)").
		Scan(&total)

	var missionGold float64
	db.DB.Model(&models.MissionEntry{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(gold), 0)").
		Scan(&missionGold)
	total += missionGold

	var usageGold float64
	db.DB.Model(&models.DLUsage{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(gold_change), 0)").
		Scan(&usageGold)
	total += usageGold

	var txGold float64
	db.DB.Model(&models.Transaction{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&txGold)
	total += txGold

	return total
}

func GetRenownTotal(characterID uint) float64 {
	var total float64

	db.DB.Model(&models.CharacterRegistry{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(renown), 0)").
		Scan(&total)

	var missionRenown float64
	db.DB.Model(&models.MissionEntry{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(renown), 0)").
		Scan(&missionRenown)
	total += missionRenown

	return total
}

func GetXPTotal(characterID uint) float64 {
	var total float64

	db.DB.Model(&models.CharacterRegistry{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(experience), 0)").
		Scan(&total)

	var missionXP float64
	db.DB.Model(&models.MissionEntry{}).
		Where("character_id = ?", characterID).
		Select("COALESCE(SUM(xp_mission + xp_report + xp_guild), 0)").
		Scan(&missionXP)
	total += missionXP

	return total
}

type CharacterStats struct {
	models.Character
	XP          float64 `json:"xp"`
	Level       int     `json:"level"`
	GoldBalance float64 `json:"gold_balance"`
	Renown      float64 `json:"renown"`
}

func GetCharacterWithStats(id uint) (*CharacterStats, error) {
	var character models.Character
	if err := db.DB.First(&character, id).Error; err != nil {
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

func GetAllCharactersWithStats() ([]CharacterStats, error) {
	var characters []models.Character
	db.DB.Find(&characters)

	var stats []CharacterStats
	for _, c := range characters {
		xp := GetXPTotal(c.ID)

		stats = append(stats, CharacterStats{
			Character:   c,
			XP:          xp,
			Level:       CalculateLevel(xp),
			GoldBalance: GetGoldBalance(c.ID),
			Renown:      GetRenownTotal(c.ID),
		})
	}

	return stats, nil
}
