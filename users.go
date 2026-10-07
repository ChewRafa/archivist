package main

import (
	"errors"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func CreateUser(username, password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	user := User{
		Username:     username,
		PasswordHash: hash,
	}
	return DB.Create(&user).Error
}

// UpsertAdminUser creates the user if missing, otherwise resets its password.
// Returns created=true when a new row was inserted.
func UpsertAdminUser(username, password string) (bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return false, errors.New("username cannot be empty")
	}
	if password == "" {
		return false, errors.New("password cannot be empty")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return false, err
	}
	var user User
	err = DB.Where("username = ?", username).First(&user).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, err
		}
		if err := DB.Create(&User{Username: username, PasswordHash: hash}).Error; err != nil {
			// Race: another instance created it concurrently — fall back to update.
			if findErr := DB.Where("username = ?", username).First(&user).Error; findErr != nil {
				return false, err
			}
			user.PasswordHash = hash
			return false, DB.Save(&user).Error
		}
		return true, nil
	}
	user.PasswordHash = hash
	return false, DB.Save(&user).Error
}

// EnsureAdminFromEnv seeds/resets the admin user from ADMIN_USERNAME and
// ADMIN_PASSWORD. Intended for first deploy (e.g. Render blueprint): if the
// user is missing it is created; if it already exists its password is reset
// so a lost password stays recoverable while the env vars remain set.
// Unset the env vars after the first successful login.
func EnsureAdminFromEnv() {
	adminUser := strings.TrimSpace(os.Getenv("ADMIN_USERNAME"))
	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass != "" {
		log.Println("ADMIN_PASSWORD: [set]")
	} else {
		log.Println("ADMIN_PASSWORD: [empty]")
	}
	if adminUser == "" && adminPass == "" {
		log.Println("ADMIN_USERNAME/ADMIN_PASSWORD not set, skipping admin seed")
		return
	}
	if adminUser == "" || adminPass == "" {
		log.Println("WARNING: ADMIN_USERNAME and ADMIN_PASSWORD must both be set to seed admin user, skipping")
		return
	}

	if os.Getenv("DATABASE_URL") != "" {
		log.Println("Seeding admin user against PostgreSQL (DATABASE_URL set)")
	} else {
		log.Printf("Seeding admin user against SQLite (DATABASE_URL empty, DB_PATH=%s)", os.Getenv("DB_PATH"))
	}

	var count int64
	if err := DB.Model(&User{}).Count(&count).Error; err != nil {
		log.Printf("Failed to count users for admin seed: %s", err)
		return
	}
	created, err := UpsertAdminUser(adminUser, adminPass)
	if err != nil {
		log.Printf("Failed to seed admin user '%s': %s", adminUser, err)
		return
	}
	if created {
		log.Printf("Admin user '%s' created from environment variables (%d pre-existing user(s))", adminUser, count)
	} else {
		log.Printf("Admin user '%s' password synced from environment variables (%d user(s) total)", adminUser, count)
	}
}

func Authenticate(username, password string) (*User, error) {
	var user User
	if err := DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, errors.New("usuario o contraseña incorrectos")
	}
	if err := CheckPassword(user.PasswordHash, password); err != nil {
		return nil, errors.New("usuario o contraseña incorrectos")
	}
	return &user, nil
}
