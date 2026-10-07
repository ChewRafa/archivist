package main

import (
	"errors"
	"log/slog"
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

	var created bool
	err = DB.Transaction(func(tx *gorm.DB) error {
		var existing User
		err := tx.Where("username = ?", username).First(&existing).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			// Not found - insert new
			user := User{Username: username, PasswordHash: hash}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
			created = true
			return nil
		}
		// Found - update password
		existing.PasswordHash = hash
		return tx.Save(&existing).Error
	})
	return created, err
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
		slog.Info("ADMIN_PASSWORD: [set]")
	} else {
		slog.Info("ADMIN_PASSWORD: [empty]")
	}
	if adminUser == "" && adminPass == "" {
		slog.Info("ADMIN_USERNAME/ADMIN_PASSWORD not set, skipping admin seed")
		return
	}
	if adminUser == "" || adminPass == "" {
		slog.Warn("ADMIN_USERNAME and ADMIN_PASSWORD must both be set to seed admin user, skipping")
		return
	}

	if os.Getenv("DATABASE_URL") != "" {
		slog.Info("Seeding admin user against PostgreSQL (DATABASE_URL set)")
	} else {
		slog.Info("Seeding admin user against SQLite", "db_path", os.Getenv("DB_PATH"))
	}

	var count int64
	if err := DB.Model(&User{}).Count(&count).Error; err != nil {
		slog.Error("Failed to count users for admin seed", "error", err)
		return
	}
	created, err := UpsertAdminUser(adminUser, adminPass)
	if err != nil {
		slog.Error("Failed to seed admin user", "username", adminUser, "error", err)
		return
	}
	if created {
		slog.Info("Admin user created from environment variables", "username", adminUser, "pre_existing_users", count)
	} else {
		slog.Info("Admin user password synced from environment variables", "username", adminUser, "total_users", count)
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
