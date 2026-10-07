//go:build !importer

package main

import (
	"testing"
	"time"
)

func TestGetAllCharactersWithStatsSetsLevelFromXP(t *testing.T) {
	initTestDB(t)

	character := Character{
		Number: 1,
		Player: "Tester",
		Name:   "Falric Snow",
		Status: "Activo",
	}
	if err := DB.Create(&character).Error; err != nil {
		t.Fatalf("create character: %s", err)
	}

	registry := CharacterRegistry{
		Date:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CharacterID: character.ID,
		Event:       "Import",
		Experience:  900,
	}
	if err := DB.Create(&registry).Error; err != nil {
		t.Fatalf("create registry: %s", err)
	}

	stats, err := GetAllCharactersWithStats()
	if err != nil {
		t.Fatalf("GetAllCharactersWithStats: %s", err)
	}
	if len(stats) != 1 {
		t.Fatalf("got %d characters, want 1", len(stats))
	}
	if stats[0].XP != 900 {
		t.Fatalf("XP = %v, want 900", stats[0].XP)
	}
	if stats[0].Level != 3 {
		t.Fatalf("Level = %d, want 3", stats[0].Level)
	}
}
