package main

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrGuildFull    = errors.New("el gremio está lleno (máximo 15 miembros)")
	ErrPlayerLimit  = errors.New("el jugador ya tiene 3 personajes activos")
	ErrCrossGuild   = errors.New("el jugador ya tiene personajes en otro gremio")
	ErrGuildInvalid = errors.New("gremio no válido")
)

const maxGuildMembers = 15
const maxCharactersPerPlayer = 3

func isActiveStatus(s string) bool {
	return strings.EqualFold(strings.TrimSpace(s), "Activo")
}

// ValidateGuildJoin checks guild capacity and player constraints.
// characterID is 0 when creating a new character; otherwise exclude that character from counts.
// newStatus is the status the character will have after the operation (e.g. form.Status or member.Status).
// Only active characters (status = 'Activo') count towards the 3-per-player limit and cross-guild check.
// Inactive characters (Muerto/Retirado) are exempt and can be guildless or in any guild.
func ValidateGuildJoin(tx *gorm.DB, player string, targetGuildID uint, characterID uint, newStatus string) error {
	if targetGuildID == 0 {
		return nil
	}
	if player == "" {
		return nil
	}

	// Guild capacity: count only active members (Activo + not soft-deleted)
	var guildCount int64
	tx.Raw(`
		SELECT COUNT(*) FROM guild_members gm
		JOIN characters c ON c.id = gm.character_id
		WHERE gm.guild_id = ? AND c.status = 'Activo' AND c.deleted_at IS NULL
	`, targetGuildID).Scan(&guildCount)
	// If character already in this guild, don't count it as new (and only if active)
	if characterID != 0 {
		var already int64
		tx.Raw("SELECT COUNT(*) FROM guild_members WHERE guild_id = ? AND character_id = ?", targetGuildID, characterID).Scan(&already)
		if already > 0 {
			// Already member, capacity not affected
		} else if guildCount >= maxGuildMembers {
			return ErrGuildFull
		}
	} else {
		if guildCount >= maxGuildMembers {
			return ErrGuildFull
		}
	}

	// Only active characters are subject to player limit and cross-guild rules
	if !isActiveStatus(newStatus) {
		return nil
	}

	// Player total active characters (excluding soft-deleted and non-activo)
	var playerTotal int64
	if characterID != 0 {
		tx.Raw("SELECT COUNT(*) FROM characters WHERE player = ? AND deleted_at IS NULL AND status = 'Activo' AND id != ?", player, characterID).Scan(&playerTotal)
		if playerTotal >= maxCharactersPerPlayer {
			return ErrPlayerLimit
		}
	} else {
		tx.Raw("SELECT COUNT(*) FROM characters WHERE player = ? AND deleted_at IS NULL AND status = 'Activo'", player).Scan(&playerTotal)
		if playerTotal >= maxCharactersPerPlayer {
			return ErrPlayerLimit
		}
	}

	// Cross-guild check: player cannot have active characters in more than one guild.
	var otherGuildIDs []uint
	if characterID != 0 {
		tx.Raw(`
			SELECT DISTINCT gm.guild_id FROM guild_members gm
			JOIN characters c ON c.id = gm.character_id
			WHERE c.player = ? AND c.deleted_at IS NULL AND c.status = 'Activo' AND c.id != ?
		`, player, characterID).Scan(&otherGuildIDs)
	} else {
		tx.Raw(`
			SELECT DISTINCT gm.guild_id FROM guild_members gm
			JOIN characters c ON c.id = gm.character_id
			WHERE c.player = ? AND c.deleted_at IS NULL AND c.status = 'Activo'
		`, player).Scan(&otherGuildIDs)
	}
	for _, gid := range otherGuildIDs {
		if gid != targetGuildID {
			return ErrCrossGuild
		}
	}

	return nil
}

// SyncCharacterGuildName updates characters.guild_name cache to reflect guild_members truth.
// If character has no membership, guild_name is cleared. If multiple (should not happen due to validation), first is used.
func SyncCharacterGuildName(tx *gorm.DB, characterID uint) error {
	var guildID uint
	tx.Raw("SELECT guild_id FROM guild_members WHERE character_id = ? LIMIT 1", characterID).Scan(&guildID)
	var guildName string
	if guildID != 0 {
		var g Guild
		if err := tx.Select("name").First(&g, guildID).Error; err == nil {
			guildName = g.Name
		}
	}
	return tx.Model(&Character{}).Where("id = ?", characterID).Update("guild_name", guildName).Error
}

// SyncAllCharacterGuildNames reconciles entire table; used on startup migration.
func SyncAllCharacterGuildNames(tx *gorm.DB) error {
	var characters []Character
	if err := tx.Find(&characters).Error; err != nil {
		return err
	}
	for _, c := range characters {
		if err := SyncCharacterGuildName(tx, c.ID); err != nil {
			return err
		}
	}
	return nil
}
