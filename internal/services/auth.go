package services

import (
	"errors"

	"codeberg.org/chewrafa/archivist/internal/db"
	"codeberg.org/chewrafa/archivist/internal/models"
	"golang.org/x/crypto/bcrypt"
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
	user := models.User{
		Username:     username,
		PasswordHash: hash,
	}
	return db.DB.Create(&user).Error
}

func Authenticate(username, password string) (*models.User, error) {
	var user models.User
	if err := db.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, errors.New("usuario o contraseña incorrectos")
	}
	if err := CheckPassword(user.PasswordHash, password); err != nil {
		return nil, errors.New("usuario o contraseña incorrectos")
	}
	return &user, nil
}
