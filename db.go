package main

import (
	"fmt"
	"log"
	"os"
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
		DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	}
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

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

	log.Println("Database initialized successfully")
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
