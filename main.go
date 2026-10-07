//go:build !importer

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	createAdmin := flag.String("create-admin", "", "Create an admin user and exit")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(Version)
		return
	}

	// Load .env for local dev. In production (e.g. Render) env vars are
	// injected directly, so a missing .env is not an error.
	// Real environment variables always take precedence over .env values.
	if os.Getenv("GIN_MODE") != "release" {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found, using environment variables")
		}
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/archivist.db"
	}
	Init(dbPath)

	EnsureAdminFromEnv()

	if *createAdmin != "" {
		fmt.Print("Password: ")
		var password string
		fmt.Scanln(&password)
		if password == "" {
			log.Fatal("Password cannot be empty")
		}
		if err := CreateUser(*createAdmin, password); err != nil {
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
		c.JSON(http.StatusOK, gin.H{"status": "ok", "version": Version})
	})

	loadTemplates()
	SetupRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
