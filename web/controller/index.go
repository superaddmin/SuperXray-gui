package controller

import (
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/logger"
	"github.com/superaddmin/SuperXray-gui/v2/web/service"
	"github.com/superaddmin/SuperXray-gui/v2/web/session"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// LoginForm represents the login request structure.
type LoginForm struct {
	Username      string `json:"username" form:"username"`
	Password      string `json:"password" form:"password"`
	TwoFactorCode string `json:"twoFactorCode" form:"twoFactorCode"`
}

// Old installations allowed longer names. A signed session cookie already
// limits the complete encoded user record to 4096 bytes.
const maxLegacyLoginUsernameBytes = 4096

func loginDisplayName(username string) string {
	if len(username) > service.MaxUsernameBytes {
		return "[legacy long username]"
	}
	return template.HTMLEscapeString(username)
}

// IndexController handles the main index and login-related routes.
type IndexController struct {
	BaseController

	settingService service.SettingService
	userService    service.UserService
	tgbot          service.Tgbot
	loginAttempts  loginLimiter
}

// NewIndexController creates a new IndexController and initializes its routes.
func NewIndexController(g *gin.RouterGroup) *IndexController {
	a := &IndexController{}
	a.initRouter(g)
	return a
}

// initRouter sets up the routes for index, login, logout, and two-factor authentication.
func (a *IndexController) initRouter(g *gin.RouterGroup) {
	g.GET("/", a.index)
	g.GET("/logout", a.logout)

	g.POST("/login", a.login)
	g.POST("/getTwoFactorEnable", a.getTwoFactorEnable)
}

// index handles the root route, redirecting logged-in users to the panel or showing the login page.
func (a *IndexController) index(c *gin.Context) {
	if session.IsLogin(c) {
		c.Redirect(http.StatusTemporaryRedirect, "panel/")
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, c.GetString("base_path")+"panel/login")
}

// login handles user authentication and session creation.
func (a *IndexController) login(c *gin.Context) {
	var form LoginForm
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
	if err := c.ShouldBind(&form); err != nil {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.invalidFormData"))
		return
	}
	if form.Username == "" {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.emptyUsername"))
		return
	}
	if len(form.Username) > maxLegacyLoginUsernameBytes {
		pureJsonMsg(c, http.StatusBadRequest, false, I18nWeb(c, "pages.login.toasts.invalidFormData"))
		return
	}
	if form.Password == "" {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.emptyPassword"))
		return
	}

	ip := getRemoteIp(c)
	attempt := a.loginAttempts.reserve(ip, form.Username, time.Now())
	if attempt == nil {
		c.Header("Retry-After", "300")
		pureJsonMsg(c, http.StatusTooManyRequests, false, http.StatusText(http.StatusTooManyRequests))
		return
	}
	authenticated := false
	defer func() { a.loginAttempts.finish(ip, attempt, authenticated, time.Now()) }()
	user, checkErr := a.userService.CheckUser(form.Username, form.Password, form.TwoFactorCode)
	authenticated = user != nil
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	safeUser := loginDisplayName(form.Username)

	if user == nil {
		logger.Warningf("failed login username: \"%s\", IP: \"%s\"", safeUser, getRemoteIp(c))
		translated2FAError := a.tgbot.I18nBot("tgbot.messages.2faFailed")
		notifyPass := redactLoginFailurePassword(form.Password, checkErr, translated2FAError)
		a.tgbot.UserLoginNotify(safeUser, notifyPass, getRemoteIp(c), timeStr, 0)
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.wrongUsernameOrPassword"))
		return
	}

	logger.Infof("%s logged in successfully, Ip Address: %s\n", safeUser, getRemoteIp(c))
	a.tgbot.UserLoginNotify(safeUser, ``, getRemoteIp(c), timeStr, 1)

	session.SetLoginUser(c, user)
	session.RegenerateCSRFToken(c)
	if err := sessions.Default(c).Save(); err != nil {
		logger.Warning("Unable to save session: ", err)
		pureJsonMsg(c, http.StatusInternalServerError, false, I18nWeb(c, "somethingWentWrong"))
		return
	}

	logger.Infof("%s logged in successfully", safeUser)
	jsonMsg(c, I18nWeb(c, "pages.login.toasts.successLogin"), nil)
}

// logout handles user logout by clearing the session and redirecting to the login page.
func (a *IndexController) logout(c *gin.Context) {
	user := session.GetLoginUser(c)
	if user != nil {
		logger.Infof("%s logged out successfully", loginDisplayName(user.Username))
	}
	session.ClearSession(c)
	if err := sessions.Default(c).Save(); err != nil {
		logger.Warning("Unable to save session after clearing:", err)
	}
	c.Redirect(http.StatusTemporaryRedirect, c.GetString("base_path"))
}

func redactLoginFailurePassword(password string, err error, twoFactorError string) string {
	redacted := "***"
	if err != nil && err.Error() == "invalid 2fa code" && twoFactorError != "" {
		return fmt.Sprintf("%s (%s)", redacted, twoFactorError)
	}
	return redacted
}

// getTwoFactorEnable retrieves the current status of two-factor authentication.
func (a *IndexController) getTwoFactorEnable(c *gin.Context) {
	status, err := a.settingService.GetTwoFactorEnable()
	if err == nil {
		jsonObj(c, status, nil)
	}
}
