package handlers

import (
	"net/http"

	"codeberg.org/chewrafa/archivist/internal/services"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func LoginPageHandler(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("user_id") != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	c.HTML(http.StatusOK, "login.html", gin.H{
		"Title": "Iniciar Sesión",
		"Error": "",
	})
}

func LoginPostHandler(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"Title": "Iniciar Sesión",
			"Error": "Usuario y contraseña son obligatorios",
		})
		return
	}

	user, err := services.Authenticate(username, password)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"Title": "Iniciar Sesión",
			"Error": "Usuario o contraseña incorrectos",
		})
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Set("csrf_token", generateCSRFToken())
	session.Save()

	c.Redirect(http.StatusFound, "/")
}

func LogoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/login")
}
