package main

import (
	"errors"

	"gorm.io/gorm"
)

func SyncGuildTreasury(tx *gorm.DB, guildID uint) error {
	var total float64
	if err := tx.Model(&GuildTransaction{}).
		Where("guild_id = ?", guildID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error; err != nil {
		return err
	}
	return tx.Model(&Guild{}).Where("id = ?", guildID).Update("treasury", total).Error
}

func CreateGuildTransaction(tx *gorm.DB, entry GuildTransaction) (bool, error) {
	if entry.Amount == 0 {
		return false, nil
	}

	var existing GuildTransaction
	q := tx.Where("guild_id = ? AND amount = ? AND notes = ?", entry.GuildID, entry.Amount, entry.Notes)
	if entry.Notes != "Registro Inicial" {
		q = q.Where("date = ?", entry.Date)
	}
	err := q.First(&existing).Error
	if err == nil {
		return false, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	if err := tx.Create(&entry).Error; err != nil {
		return false, err
	}
	if err := SyncGuildTreasury(tx, entry.GuildID); err != nil {
		return false, err
	}
	return true, nil
}
