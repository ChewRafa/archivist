package db

import (
	"log"

	"codeberg.org/chewrafa/archivist/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(dbPath string) {
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	err = DB.AutoMigrate(
		&models.User{},
		&models.Character{},
		&models.DLUsage{},
		&models.Transaction{},
		&models.CostOfLiving{},
		&models.CharacterRegistry{},
		&models.Mission{},
		&models.MissionEntry{},
		&models.Guild{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}

	log.Println("Database initialized successfully")
}
