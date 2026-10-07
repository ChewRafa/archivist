package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

var spanishMonthAbbr = [12]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"}

func formatDate(v any) string {
	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			return "—"
		}
		return fmt.Sprintf("%02d/%s/%04d", t.Day(), spanishMonthAbbr[t.Month()-1], t.Year())
	case *time.Time:
		if t == nil || t.IsZero() {
			return "—"
		}
		return fmt.Sprintf("%02d/%s/%04d", t.Day(), spanishMonthAbbr[t.Month()-1], t.Year())
	default:
		return "—"
	}
}

var templates map[string]*template.Template

func loadTemplates() {
	funcMap := template.FuncMap{
		"mul":        func(a, b int) int { return a * b },
		"add3":       func(a, b, c float64) float64 { return a + b + c },
		"formatDate": formatDate,
		"seq":        func(start, end int) []int { r := make([]int, end-start+1); for i := range r { r[i] = start + i }; return r },
		"add":        func(a, b int) int { return a + b },
		"sub":        func(a, b int) int { return a - b },
	}

	base := template.Must(template.New("base.html").Funcs(funcMap).ParseFiles("resources/base.html"))
	templates = make(map[string]*template.Template)

	pages, err := filepath.Glob("resources/pages/*.html")
	if err != nil {
		slog.Error("Failed to glob page templates", "error", err)
		os.Exit(1)
	}
	for _, page := range pages {
		name := filepath.Base(page)
		tmpl := template.Must(base.Clone())
		tmpl = template.Must(tmpl.ParseFiles(page))
		templates[name] = tmpl
	}
	slog.Info("Loaded page templates", "count", len(templates))
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
		data["CSRFToken"] = token
	}

	if data["Flash"] == nil {
		if flash := getFlash(c); flash != nil {
			data["Flash"] = flash
		}
	}

	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(c.Writer, "base.html", data); err != nil {
		slog.Error("template execute error", "page", page, "error", err)
	}
}
