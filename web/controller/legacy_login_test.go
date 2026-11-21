package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/util/crypto"
	"github.com/superaddmin/SuperXray-gui/v2/web/session"
)

func TestLegacyLongUsernameLoginAndSession(t *testing.T) {
	if err := database.InitDB(filepath.Join(t.TempDir(), "auth.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.CloseDB() })
	hash, err := crypto.HashPasswordAsBcrypt("fixture")
	if err != nil {
		t.Fatal(err)
	}
	name := strings.Repeat("a", 1500)
	if err := database.GetDB().Create(&model.User{Username: name, Password: hash}).Error; err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(sessions.Sessions("panel", cookie.NewStore([]byte("test-only-cookie-signing-key"))))
	NewIndexController(router.Group("/"))
	router.GET("/check", func(c *gin.Context) {
		user := session.GetLoginUser(c)
		if user == nil || user.Username != name {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Status(http.StatusNoContent)
	})
	login := func(username string) *httptest.ResponseRecorder {
		form := url.Values{"username": {username}, "password": {"fixture"}}
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}
	response := login(name)
	if response.Code != http.StatusOK || len(response.Result().Cookies()) == 0 || loginDisplayName(name) == name {
		t.Fatal("legacy username could not establish a session with a bounded display name")
	}
	request := httptest.NewRequest(http.MethodGet, "/check", nil)
	for _, stored := range response.Result().Cookies() {
		request.AddCookie(stored)
	}
	checked := httptest.NewRecorder()
	router.ServeHTTP(checked, request)
	if checked.Code != http.StatusNoContent {
		t.Fatal("legacy session did not preserve the account")
	}
	if login(strings.Repeat("x", maxLegacyLoginUsernameBytes+1)).Code != http.StatusBadRequest {
		t.Fatal("unbounded username passed the login form")
	}
}
