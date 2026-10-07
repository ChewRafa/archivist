package main

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

var (
	loginAttempts = make(map[string][]time.Time)
	loginMutex    sync.Mutex
)

func getLoginRateLimitConfig() (maxAttempts int, window time.Duration) {
	maxAttempts = 5
	if v := os.Getenv("LOGIN_MAX_ATTEMPTS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			maxAttempts = parsed
		}
	}
	windowMinutes := 15
	if v := os.Getenv("LOGIN_WINDOW_MINUTES"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			windowMinutes = parsed
		}
	}
	window = time.Duration(windowMinutes) * time.Minute
	return
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("user_id")
		if userID == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Set("user_id", userID)
		c.Next()
	}
}

func CSRFRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodDelete {
			session := sessions.Default(c)
			token := session.Get("csrf_token")
			formToken := c.PostForm("csrf_token")
			if token == nil || formToken == "" || token != formToken {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
		}
		c.Next()
	}
}

func RateLimitLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		maxAttempts, window := getLoginRateLimitConfig()
		ip := c.ClientIP()

		loginMutex.Lock()
		now := time.Now()
		attempts := loginAttempts[ip]

		// Clean old attempts outside the window
		var recent []time.Time
		for _, t := range attempts {
			if now.Sub(t) < window {
				recent = append(recent, t)
			}
		}

		if len(recent) >= maxAttempts {
			loginAttempts[ip] = recent
			loginMutex.Unlock()
			c.HTML(http.StatusTooManyRequests, "login.html", gin.H{
				"Title": "Iniciar Sesión",
				"Error": "Demasiados intentos fallidos. Intente de nuevo en " + window.String() + ".",
			})
			c.Abort()
			return
		}

		loginAttempts[ip] = recent
		loginMutex.Unlock()
		c.Next()
	}
}

func recordFailedLoginAttempt(ip string) {
	_, window := getLoginRateLimitConfig()

	loginMutex.Lock()
	now := time.Now()
	attempts := loginAttempts[ip]

	var recent []time.Time
	for _, t := range attempts {
		if now.Sub(t) < window {
			recent = append(recent, t)
		}
	}
	recent = append(recent, now)
	loginAttempts[ip] = recent
	loginMutex.Unlock()
}
