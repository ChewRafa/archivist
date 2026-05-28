package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"codeberg.org/chewrafa/archivist/internal/db"
	"codeberg.org/chewrafa/archivist/internal/handlers"
	"codeberg.org/chewrafa/archivist/internal/models"
	"codeberg.org/chewrafa/archivist/internal/services"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func main() {
	createAdmin := flag.String("create-admin", "", "Create an admin user and exit")
	flag.Parse()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/archivist.db"
	}
	db.Init(dbPath)

	if adminUser := os.Getenv("ADMIN_USERNAME"); adminUser != "" {
		if adminPass := os.Getenv("ADMIN_PASSWORD"); adminPass != "" {
			var count int64
			db.DB.Model(&models.User{}).Count(&count)
			if count == 0 {
				if err := services.CreateUser(adminUser, adminPass); err != nil {
					log.Printf("Failed to create admin user '%s': %s", adminUser, err)
				} else {
					log.Printf("Admin user '%s' created from environment variables", adminUser)
				}
			}
		}
	}

	if *createAdmin != "" {
		fmt.Print("Password: ")
		var password string
		fmt.Scanln(&password)
		if password == "" {
			log.Fatal("Password cannot be empty")
		}
		if err := services.CreateUser(*createAdmin, password); err != nil {
			log.Fatal("Failed to create user: ", err)
		}
		log.Printf("User '%s' created successfully", *createAdmin)
		return
	}

	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.MaxMultipartMemory = 32 << 20

	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
		log.Println("WARNING: SESSION_SECRET not set, using insecure default")
	}
	store := cookie.NewStore([]byte(secret))
	r.Use(sessions.Sessions("archivist_session", store))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	handlers.SetupRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
