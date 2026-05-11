package controller

import (
	"errors"
	"time"

	"github.com/govnoeby/3x-ui/v3/util/crypto"
	"github.com/govnoeby/3x-ui/v3/web/entity"
	"github.com/govnoeby/3x-ui/v3/web/service"
	"github.com/govnoeby/3x-ui/v3/web/session"

	"github.com/gin-gonic/gin"
)

// updateUserForm represents the form for updating user credentials.
type updateUserForm struct {
	OldUsername string `json:"oldUsername" form:"oldUsername"`
	OldPassword string `json:"oldPassword" form:"oldPassword"`
	NewUsername string `json:"newUsername" form:"newUsername"`
	NewPassword string `json:"newPassword" form:"newPassword"`
}

// SettingController handles settings and user management operations.
type SettingController struct {
	settingService service.SettingService
	userService    service.UserService
	panelService   service.PanelService
}

// NewSettingController creates a new SettingController and initializes its routes.
func NewSettingController(g *gin.RouterGroup) *SettingController {
	a := &SettingController{}
	a.initRouter(g)
	return a
}

// initRouter sets up the routes for settings management.
func (a *SettingController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/setting")

	g.POST("/all", a.getAllSetting)
	g.POST("/defaultSettings", a.getDefaultSettings)
	g.POST("/update", a.updateSetting)
	g.POST("/updateUser", a.updateUser)
	g.POST("/restartPanel", a.restartPanel)
	g.GET("/getDefaultJsonConfig", a.getDefaultXrayConfig)
	g.GET("/getApiToken", a.getApiToken)
	g.POST("/regenerateApiToken", a.regenerateApiToken)
}

// @Summary      Get all settings
// @Description  Returns all panel configuration settings including web server, Telegram bot, subscription, security, and LDAP settings.
// @Tags         Settings
// @Produce      json
// @Success      200 {object} entity.Msg{obj=entity.AllSetting}
// @Security     BearerAuth
// @Router       /panel/setting/all [post]
func (a *SettingController) getAllSetting(c *gin.Context) {
	allSetting, err := a.settingService.GetAllSetting()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.getSettings"), err)
		return
	}
	jsonObj(c, allSetting, nil)
}

// @Summary      Get default settings
// @Description  Returns the default panel settings based on the request host.
// @Tags         Settings
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/setting/defaultSettings [post]
func (a *SettingController) getDefaultSettings(c *gin.Context) {
	result, err := a.settingService.GetDefaultSettings(c.Request.Host)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.getSettings"), err)
		return
	}
	jsonObj(c, result, nil)
}

// @Summary      Update settings
// @Description  Updates all panel settings with the provided configuration.
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Param        body body entity.AllSetting true "All panel settings"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/setting/update [post]
func (a *SettingController) updateSetting(c *gin.Context) {
	allSetting := &entity.AllSetting{}
	err := c.ShouldBind(allSetting)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.modifySettings"), err)
		return
	}
	err = a.settingService.UpdateAllSetting(allSetting)
	jsonMsg(c, I18nWeb(c, "pages.settings.toasts.modifySettings"), err)
}

// @Summary      Update user credentials
// @Description  Changes the panel admin username and password. Requires old credentials for verification.
// @Tags         Settings
// @Accept       json
// @Produce      json
// @Param        body body updateUserForm true "Old and new credentials"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/setting/updateUser [post]
func (a *SettingController) updateUser(c *gin.Context) {
	form := &updateUserForm{}
	err := c.ShouldBind(form)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.modifySettings"), err)
		return
	}
	user := session.GetLoginUser(c)
	if user.Username != form.OldUsername || !crypto.CheckPasswordHash(user.Password, form.OldPassword) {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.modifyUserError"), errors.New(I18nWeb(c, "pages.settings.toasts.originalUserPassIncorrect")))
		return
	}
	if form.NewUsername == "" || form.NewPassword == "" {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.modifyUserError"), errors.New(I18nWeb(c, "pages.settings.toasts.userPassMustBeNotEmpty")))
		return
	}
	err = a.userService.UpdateUser(user.Id, form.NewUsername, form.NewPassword)
	if err == nil {
		user.Username = form.NewUsername
		user.Password, _ = crypto.HashPasswordAsBcrypt(form.NewPassword)
		if saveErr := session.SetLoginUser(c, user); saveErr != nil {
			err = saveErr
		}
	}
	jsonMsg(c, I18nWeb(c, "pages.settings.toasts.modifyUser"), err)
}

// @Summary      Restart panel
// @Description  Restarts the panel service after a 3-second delay.
// @Tags         Settings
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/setting/restartPanel [post]
func (a *SettingController) restartPanel(c *gin.Context) {
	err := a.panelService.RestartPanel(time.Second * 3)
	jsonMsg(c, I18nWeb(c, "pages.settings.restartPanelSuccess"), err)
}

// @Summary      Get default Xray config
// @Description  Returns the default Xray configuration template.
// @Tags         Settings
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/setting/getDefaultJsonConfig [get]
func (a *SettingController) getDefaultXrayConfig(c *gin.Context) {
	defaultJsonConfig, err := a.settingService.GetDefaultXrayConfig()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.getSettings"), err)
		return
	}
	jsonObj(c, defaultJsonConfig, nil)
}

// @Summary      Get API token
// @Description  Returns the panel's API token used for Bearer authentication. Auto-generated on first read for seamless upgrades.
// @Tags         Settings
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/setting/getApiToken [get]
func (a *SettingController) getApiToken(c *gin.Context) {
	tok, err := a.settingService.GetApiToken()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.getSettings"), err)
		return
	}
	jsonObj(c, tok, nil)
}

// @Summary      Regenerate API token
// @Description  Rotates the API token. Existing integrations using the old token will immediately fail until updated.
// @Tags         Settings
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/setting/regenerateApiToken [post]
func (a *SettingController) regenerateApiToken(c *gin.Context) {
	tok, err := a.settingService.RegenerateApiToken()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.modifySettings"), err)
		return
	}
	jsonObj(c, tok, nil)
}
