package main

import (
	"crypto/rand"
	"encoding/hex"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

var templates map[string]*template.Template

func loadTemplates() {
	funcMap := template.FuncMap{
		"mul":  func(a, b int) int { return a * b },
		"add3": func(a, b, c float64) float64 { return a + b + c },
	}

	base := template.Must(template.New("base.html").Funcs(funcMap).ParseFiles("resources/base.html"))
	templates = make(map[string]*template.Template)

	pages, err := filepath.Glob("resources/pages/*.html")
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

type Flash struct {
	Kind    string
	Message string
}

func setFlash(c *gin.Context, kind, message string) {
	session := sessions.Default(c)
	session.Set("flash", &Flash{Kind: kind, Message: message})
	session.Save()
}

func getFlash(c *gin.Context) *Flash {
	session := sessions.Default(c)
	raw := session.Get("flash")
	if raw == nil {
		return nil
	}
	session.Delete("flash")
	session.Save()
	flash, ok := raw.(*Flash)
	if !ok {
		return nil
	}
	return flash
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
			var user User
			if err := DB.First(&user, userID).Error; err == nil {
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

	if data["Flash"] == nil {
		if flash := getFlash(c); flash != nil {
			data["Flash"] = flash
		}
	}

	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	tmpl.ExecuteTemplate(c.Writer, "base.html", data)
}
