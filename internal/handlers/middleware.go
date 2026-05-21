package handlers

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

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
