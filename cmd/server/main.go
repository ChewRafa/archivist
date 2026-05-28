package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"codeberg.org/chewrafa/archivist/internal/db"
	"codeberg.org/chewrafa/archivist/internal/handlers"
	"codeberg.org/chewrafa/archivist/internal/services"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func main() {
	createAdmin := flag.String("create-admin", "", "Create an admin user and exit")
	flag.Parse()

	db.Init("data/archivist.db")

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

	handlers.SetupRoutes(r)

	log.Println("Server starting on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
