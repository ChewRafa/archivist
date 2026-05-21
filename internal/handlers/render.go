package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"codeberg.org/chewrafa/archivist/internal/db"
	"codeberg.org/chewrafa/archivist/internal/models"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

var templates map[string]*template.Template

func init() {
	funcMap := template.FuncMap{
		"mul":  func(a, b int) int { return a * b },
		"add3": func(a, b, c float64) float64 { return a + b + c },
	}

	base := template.Must(template.New("base.html").Funcs(funcMap).ParseFiles("templates/base.html"))
	templates = make(map[string]*template.Template)

	pages, err := filepath.Glob("templates/pages/*.html")
	if err != nil {
		log.Fatal("Failed to glob page templates: ", err)
	}
	for _, page := range pages {
		name := filepath.Base(page)
		tmpl := template.Must(base.Clone())
		tmpl = template.Must(tmpl.ParseFiles(page))
		templates[name] = tmpl
	}
	log.Println("Loaded", len(templates), "page templates")
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func render(c *gin.Context, status int, page string, data gin.H) {
	tmpl, ok := templates[page]
	if !ok {
		c.String(http.StatusInternalServerError, "template not found: "+page)
		return
	}

	session := sessions.Default(c)
	if data["User"] == nil {
		userID := session.Get("user_id")
		if userID != nil {
			var user models.User
			if err := db.DB.First(&user, userID).Error; err == nil {
				data["User"] = &user
			}
		}
	}
	if data["User"] == nil {
		data["User"] = nil
	}

	if data["CSRFToken"] == nil {
		token := session.Get("csrf_token")
		if token == nil {
			token = generateCSRFToken()
			session.Set("csrf_token", token)
			session.Save()
		}
		data["CSRFToken"] = token
	}

	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	tmpl.ExecuteTemplate(c.Writer, "base.html", data)
}
