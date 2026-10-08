//go:build !importer

package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestImportJSONRestoresExportRoundTrip(t *testing.T) {
	initTestDB(t)

	character := Character{
		Number:    7,
		Player:    "Tester",
		Name:      "Falric Snow",
		Status:    "Activo",
		GuildName: "Dagorath",
	}
	if err := DB.Create(&character).Error; err != nil {
		t.Fatalf("create character: %s", err)
	}

	registry := CharacterRegistry{
		Date:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CharacterID: character.ID,
		Event:       "Import",
		Experience:  900,
		Gold:        50,
		Renown:      5,
	}
	if err := DB.Create(&registry).Error; err != nil {
		t.Fatalf("create registry: %s", err)
	}

	mission := Mission{
		Date: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		DM:   "DM Pedro",
		Name: "Las Minas de Moria",
	}
	if err := DB.Create(&mission).Error; err != nil {
		t.Fatalf("create mission: %s", err)
	}
	if err := DB.Delete(&mission).Error; err != nil {
		t.Fatalf("soft-delete mission: %s", err)
	}

	guild := Guild{
		Name:     "Dagorath",
		HallType: "Sede Central",
		Treasury: 120,
	}
	if err := DB.Create(&guild).Error; err != nil {
		t.Fatalf("create guild: %s", err)
	}
	if err := DB.Exec("INSERT INTO guild_members (guild_id, character_id) VALUES (?, ?)", guild.ID, character.ID).Error; err != nil {
		t.Fatalf("insert guild member: %s", err)
	}
	guildTx := GuildTransaction{
		Date:    time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		GuildID: guild.ID,
		Amount:  120,
		Notes:   "donación inicial",
	}
	if err := DB.Create(&guildTx).Error; err != nil {
		t.Fatalf("create guild transaction: %s", err)
	}

	export, err := ExportAll(DB)
	if err != nil {
		t.Fatalf("ExportAll: %s", err)
	}

	Init(filepath.Join(t.TempDir(), "restore.db"))

	result := ImportJSON(export)
	if len(result.Errors) > 0 {
		t.Fatalf("ImportJSON errors: %v", result.Errors)
	}
	if result.Characters != 1 {
		t.Errorf("Characters = %d, want 1", result.Characters)
	}
	if result.Guilds != 1 {
		t.Errorf("Guilds = %d, want 1", result.Guilds)
	}
	if result.GuildMembers != 1 {
		t.Errorf("GuildMembers = %d, want 1", result.GuildMembers)
	}
	if result.Missions != 1 {
		t.Errorf("Missions = %d, want 1", result.Missions)
	}

	var got Character
	if err := DB.First(&got, character.ID).Error; err != nil {
		t.Fatalf("restored character: %s", err)
	}
	if got.Name != character.Name || got.Number != character.Number || got.GuildName != character.GuildName {
		t.Errorf("character = %+v, want name=%q number=%d guild=%q", got, character.Name, character.Number, character.GuildName)
	}

	var gotMission Mission
	if err := DB.Unscoped().Where("name = ?", mission.Name).First(&gotMission).Error; err != nil {
		t.Fatalf("restored mission: %s", err)
	}
	if !gotMission.DeletedAt.Valid {
		t.Error("mission should be restored as soft-deleted")
	}

	var memberCount int64
	if err := DB.Table("guild_members").Count(&memberCount).Error; err != nil {
		t.Fatalf("count guild_members: %s", err)
	}
	if memberCount != 1 {
		t.Errorf("guild_members = %d, want 1", memberCount)
	}

	var gotGuild Guild
	if err := DB.First(&gotGuild, guild.ID).Error; err != nil {
		t.Fatalf("restored guild: %s", err)
	}
	if gotGuild.Treasury != 120 {
		t.Errorf("treasury = %v, want 120", gotGuild.Treasury)
	}

	stats, err := GetAllCharactersWithStats()
	if err != nil {
		t.Fatalf("GetAllCharactersWithStats: %s", err)
	}
	if len(stats) != 1 {
		t.Fatalf("stats rows = %d, want 1", len(stats))
	}
	if stats[0].XP != 900 {
		t.Errorf("XP = %v, want 900", stats[0].XP)
	}
	if stats[0].Level != 3 {
		t.Errorf("Level = %d, want 3", stats[0].Level)
	}
}

func TestImportJSONWipesExistingDataButKeepsUsers(t *testing.T) {
	initTestDB(t)

	if err := DB.Create(&User{Username: "keeper", PasswordHash: "hash"}).Error; err != nil {
		t.Fatalf("create user: %s", err)
	}
	if err := DB.Create(&Character{Number: 99, Player: "Old", Name: "Old Char"}).Error; err != nil {
		t.Fatalf("create old character: %s", err)
	}

	export := &ExportData{
		ExportedAt: time.Now(),
		Version:    "test",
		Characters: []Character{{ID: 1, Number: 1, Player: "New", Name: "New Hero"}},
	}
	result := ImportJSON(export)
	if len(result.Errors) > 0 {
		t.Fatalf("ImportJSON errors: %v", result.Errors)
	}

	var err error
	if err = DB.Where("name = ?", "Old Char").First(&Character{}).Error; err == nil {
		t.Error("old character should have been wiped")
	}

	var got Character
	if err := DB.First(&got, 1).Error; err != nil {
		t.Fatalf("restored character: %s", err)
	}
	if got.Name != "New Hero" {
		t.Errorf("name = %q, want %q", got.Name, "New Hero")
	}

	if err = DB.Where("username = ?", "keeper").First(&User{}).Error; err != nil {
		t.Errorf("user should survive a restore: %s", err)
	}
}

func TestImportJSONKeepsIDSequence(t *testing.T) {
	initTestDB(t)

	export := &ExportData{
		ExportedAt: time.Now(),
		Version:    "test",
		Characters: []Character{
			{ID: 5, Number: 1, Player: "P", Name: "Alpha"},
			{ID: 9, Number: 2, Player: "P", Name: "Beta"},
		},
	}
	result := ImportJSON(export)
	if len(result.Errors) > 0 {
		t.Fatalf("ImportJSON errors: %v", result.Errors)
	}

	fresh := Character{Number: 3, Player: "P", Name: "Gamma"}
	if err := DB.Create(&fresh).Error; err != nil {
		t.Fatalf("create character after restore: %s", err)
	}
	if fresh.ID <= 9 {
		t.Errorf("new character ID = %d, want > 9", fresh.ID)
	}
}

func TestImportJSONNilDataIsRejected(t *testing.T) {
	initTestDB(t)

	result := ImportJSON(nil)
	if len(result.Errors) != 1 {
		t.Fatalf("errors = %v, want 1 error", result.Errors)
	}
}

func TestValidateExportJSON(t *testing.T) {
	valid, err := json.Marshal(&ExportData{
		ExportedAt: time.Now(),
		Version:    "test",
		Characters: []Character{{Number: 1, Name: "Hero"}},
	})
	if err != nil {
		t.Fatalf("marshal: %s", err)
	}
	if _, err := ValidateExportJSON(valid); err != nil {
		t.Errorf("valid export rejected: %s", err)
	}

	bom := append([]byte{0xEF, 0xBB, 0xBF}, valid...)
	if _, err := ValidateExportJSON(bom); err != nil {
		t.Errorf("BOM-prefixed export rejected: %s", err)
	}

	for _, tc := range []struct {
		name string
		body string
	}{
		{"not json", "hello world"},
		{"empty object", "{}"},
		{"unrelated object", `{"foo": 1}`},
		{"array", `[]`},
	} {
		if _, err := ValidateExportJSON([]byte(tc.body)); err == nil {
			t.Errorf("%s: expected error", tc.name)
		}
	}
}

func TestDetectImportFormat(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		head     []byte
		want     string
	}{
		{"json body", "data.bin", []byte(`{"exported_at": "2026-01-01T00:00:00Z"}`), formatJSON},
		{"json array body", "data.bin", []byte("[1,2]"), formatJSON},
		{"bom json", "data.bin", []byte("\xEF\xBB\xBF{"), formatJSON},
		{"whitespace json", "data.bin", []byte("\n  {"), formatJSON},
		{"xlsx magic", "book.bin", []byte{0x50, 0x4B, 0x03, 0x04, 0x14, 0x00}, formatXLSX},
		{"json extension", "export.json", []byte("nonsense"), formatJSON},
		{"xlsx extension", "book.xlsx", []byte("nonsense"), formatXLSX},
		{"unknown", "file.txt", []byte("nonsense"), ""},
		{"empty", "file.txt", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectImportFormat(tc.filename, tc.head); got != tc.want {
				t.Errorf("detectImportFormat(%q, %q) = %q, want %q", tc.filename, tc.head, got, tc.want)
			}
		})
	}
}
