// Package session provides session management utilities for the SuperXray web panel.
// It handles user authentication state, login sessions, and session storage using Gin sessions.
package session

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/gob"
	"net/http"
	"strings"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	loginUserKey = "LOGIN_USER"
	csrfTokenKey = "CSRF_TOKEN"
)

func init() {
	gob.Register(model.User{})
}

// SetLoginUser stores the authenticated user in the session.
// The user object is serialized and stored for subsequent requests.
func SetLoginUser(c *gin.Context, user *model.User) {
	if user == nil {
		return
	}
	s := sessions.Default(c)
	s.Set(loginUserKey, *user)
}

// RegenerateCSRFToken creates a fresh per-session token after login.
func RegenerateCSRFToken(c *gin.Context) string {
	token, err := newCSRFToken()
	if err != nil {
		return ""
	}
	s := sessions.Default(c)
	s.Set(csrfTokenKey, token)
	return token
}

// EnsureCSRFToken returns the current session token, creating one for older sessions if needed.
func EnsureCSRFToken(c *gin.Context) string {
	s := sessions.Default(c)
	if token, ok := s.Get(csrfTokenKey).(string); ok && token != "" {
		return token
	}
	token := RegenerateCSRFToken(c)
	if token != "" {
		_ = s.Save()
	}
	return token
}

// VerifyCSRFToken compares the submitted token with the value bound to this session.
func VerifyCSRFToken(c *gin.Context, token string) bool {
	expected, ok := sessions.Default(c).Get(csrfTokenKey).(string)
	if !ok || expected == "" || token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(token)) == 1
}

// GetLoginUser retrieves the authenticated user from the session.
// Returns nil if no user is logged in or if the session data is invalid.
func GetLoginUser(c *gin.Context) *model.User {
	s := sessions.Default(c)
	obj := s.Get(loginUserKey)
	if obj == nil {
		return nil
	}
	user, ok := obj.(model.User)
	if !ok {

		s.Delete(loginUserKey)
		return nil
	}
	return &user
}

// IsLogin checks if a user is currently authenticated in the session.
// Returns true if a valid user session exists, false otherwise.
func IsLogin(c *gin.Context) bool {
	return GetLoginUser(c) != nil
}

// ClearSession removes all session data and invalidates the session.
// This effectively logs out the user and clears any stored session information.
func ClearSession(c *gin.Context) {
	s := sessions.Default(c)
	s.Clear()
	cookiePath := c.GetString("base_path")
	if cookiePath == "" {
		cookiePath = "/"
	}
	secureCookie := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	s.Options(sessions.Options{
		Path:     cookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func newCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
