//go:build !importer

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	InitLogger()

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
			slog.Info("No .env file found, using environment variables")
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
			slog.Error("Password cannot be empty")
			os.Exit(1)
		}
		if err := CreateUser(*createAdmin, password); err != nil {
			slog.Error("Failed to create user", "error", err)
			os.Exit(1)
		}
		slog.Info("User created successfully", "username", *createAdmin)
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
		slog.Warn("SESSION_SECRET not set, using insecure default")
	}
	store := cookie.NewStore([]byte(secret))
	r.Use(sessions.Sessions("archivist_session", store))

	r.GET("/health", func(c *gin.Context) {
		sqlDB, err := DB.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "db": "unavailable", "version": Version})
			return
		}
		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "db": "unreachable", "version": Version})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "version": Version})
	})

	loadTemplates()
	SetupRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	slog.Info("Server starting", "port", port)
	if err := r.Run(":" + port); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
