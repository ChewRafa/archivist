package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	sqlite "github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(dbPath string) {
	var err error

	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	} else {
		if dbPath != ":memory:" && !strings.HasPrefix(dbPath, "file:") {
			dir := filepath.Dir(dbPath)
			if dir != "." && dir != "" {
				if err := os.MkdirAll(dir, 0755); err != nil {
					log.Fatal("Failed to create database directory: ", err)
				}
			}
		}
		DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	}
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	// Clean guild_members join table before AutoMigrate (source of truth fix)
	deduplicateGuildMembers()
	cleanupGuildMembersOrphans()

	// Deduplicate tables before AutoMigrate adds unique indexes
	deduplicateTable("dl_usages", []string{"date", "character_id", "dl_used", "description"})
	deduplicateTable("transactions", []string{"date", "character_id", "amount", "notes"})
	deduplicateTable("cost_of_livings", []string{"date", "character_id", "amount"})
	deduplicateTable("character_registries", []string{"date", "character_id", "event"})
	deduplicateTable("guild_transactions", []string{"date", "guild_id", "amount", "notes"})

	err = DB.AutoMigrate(
		&User{},
		&Character{},
		&DLUsage{},
		&Transaction{},
		&CostOfLiving{},
		&CharacterRegistry{},
		&Mission{},
		&MissionEntry{},
		&Guild{},
		&GuildTransaction{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}

	// Drop orphan columns from old CostOfLiving model
	migrator := DB.Migrator()
	for _, col := range []string{"inn", "guardiana", "pluma_negra", "hijos_alba"} {
		if migrator.HasColumn(&CostOfLiving{}, col) {
			migrator.DropColumn(&CostOfLiving{}, col)
		}
	}

	// Ensure guild_members has UNIQUE(guild_id, character_id) to prevent duplicates
	if DB.Migrator().HasTable("guild_members") {
		// Create unique index if not exists (SQLite/Postgres compatible via GORM)
		if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_guild_members_unique ON guild_members(guild_id, character_id)").Error; err != nil {
			log.Printf("Warning: failed to create guild_members unique index: %v", err)
		}
		// Post-migrate cleanup again in case concurrent inserts created dupes
		deduplicateGuildMembers()
		cleanupGuildMembersOrphans()
		// Reconcile characters.guild_name cache to join table truth
		if err := SyncAllCharacterGuildNames(DB); err != nil {
			log.Printf("Warning: failed to sync guild_name cache: %v", err)
		}
	}

	log.Println("Database initialized successfully")
}

func deduplicateGuildMembers() {
	if !DB.Migrator().HasTable("guild_members") {
		return
	}
	// Remove duplicate (guild_id, character_id) keeping earliest row
	res := DB.Exec("DELETE FROM guild_members WHERE rowid NOT IN (SELECT MIN(rowid) FROM guild_members GROUP BY guild_id, character_id)")
	// Fallback for Postgres which has no rowid: use id
	if res.Error != nil {
		res = DB.Exec("DELETE FROM guild_members WHERE id NOT IN (SELECT MIN(id) FROM guild_members GROUP BY guild_id, character_id)")
	}
	if res.Error != nil {
		log.Printf("Warning: failed to deduplicate guild_members: %v", res.Error)
		return
	}
	if res.RowsAffected > 0 {
		log.Printf("Removed %d duplicate row(s) from guild_members", res.RowsAffected)
	}
}

func cleanupGuildMembersOrphans() {
	if !DB.Migrator().HasTable("guild_members") {
		return
	}
	res := DB.Exec("DELETE FROM guild_members WHERE guild_id NOT IN (SELECT id FROM guilds) OR character_id NOT IN (SELECT id FROM characters WHERE deleted_at IS NULL)")
	if res.Error != nil {
		// Table guilds/characters may not exist yet on first init
		return
	}
	if res.RowsAffected > 0 {
		log.Printf("Removed %d orphan row(s) from guild_members", res.RowsAffected)
	}
}

func deduplicateTable(table string, columns []string) {
	if !DB.Migrator().HasTable(table) {
		return
	}
	cols := strings.Join(columns, ", ")
	sql := fmt.Sprintf(
		"DELETE FROM %s WHERE id NOT IN (SELECT id FROM (SELECT MIN(id) AS id FROM %s GROUP BY %s))",
		table, table, cols,
	)
	res := DB.Exec(sql)
	if res.Error != nil {
		log.Printf("Warning: failed to deduplicate %s: %v", table, res.Error)
		return
	}
	if res.RowsAffected > 0 {
		log.Printf("Removed %d duplicate row(s) from %s", res.RowsAffected, table)
	}
}
