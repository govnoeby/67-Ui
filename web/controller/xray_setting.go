package controller

import (
	"encoding/json"

	"github.com/govnoeby/67-Ui/v3/util/common"
	"github.com/govnoeby/67-Ui/v3/web/service"

	"github.com/gin-gonic/gin"
)

// XraySettingController handles Xray configuration and settings operations.
type XraySettingController struct {
	XraySettingService service.XraySettingService
	SettingService     service.SettingService
	InboundService     service.InboundService
	OutboundService    service.OutboundService
	XrayService        service.XrayService
	WarpService        service.WarpService
	NordService        service.NordService
}

// NewXraySettingController creates a new XraySettingController and initializes its routes.
func NewXraySettingController(g *gin.RouterGroup) *XraySettingController {
	a := &XraySettingController{}
	a.initRouter(g)
	return a
}

// initRouter sets up the routes for Xray settings management.
func (a *XraySettingController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/xray")
	g.GET("/getDefaultJsonConfig", a.getDefaultXrayConfig)
	g.GET("/getOutboundsTraffic", a.getOutboundsTraffic)
	g.GET("/getXrayResult", a.getXrayResult)

	g.POST("/", a.getXraySetting)
	g.POST("/warp/:action", a.warp)
	g.POST("/nord/:action", a.nord)
	g.POST("/update", a.updateSetting)
	g.POST("/resetOutboundsTraffic", a.resetOutboundsTraffic)
	g.POST("/testOutbound", a.testOutbound)
}

// @Summary      Get Xray settings
// @Description  Returns the Xray configuration template, inbound tags, and outbound test URL.
// @Tags         Xray Settings
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/xray/ [post]
func (a *XraySettingController) getXraySetting(c *gin.Context) {
	xraySetting, err := a.SettingService.GetXrayConfigTemplate()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.getSettings"), err)
		return
	}
	// Older versions of this handler embedded the raw DB value as
	// `xraySetting` in the response without checking if the value
	// already had that wrapper shape. When the frontend saved it
	// back through the textarea verbatim, the wrapper got persisted
	// and every subsequent save nested another layer, which is what
	// eventually produced the blank Xray Settings page in #4059.
	// Strip any such wrapper here, and heal the DB if we found one so
	// the next read is O(1) instead of climbing the same pile again.
	if unwrapped := service.UnwrapXrayTemplateConfig(xraySetting); unwrapped != xraySetting {
		if saveErr := a.XraySettingService.SaveXraySetting(unwrapped); saveErr == nil {
			xraySetting = unwrapped
		} else {
			// Don't fail the read — just serve the unwrapped value
			// and leave the DB healing for a later save.
			xraySetting = unwrapped
		}
	}
	inboundTags, err := a.InboundService.GetInboundTags()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.getSettings"), err)
		return
	}
	clientReverseTags, err := a.InboundService.GetClientReverseTags()
	if err != nil {
		clientReverseTags = "[]"
	}
	outboundTestUrl, _ := a.SettingService.GetXrayOutboundTestUrl()
	if outboundTestUrl == "" {
		outboundTestUrl = "https://www.google.com/generate_204"
	}
	xrayResponse := map[string]any{
		"xraySetting":       json.RawMessage(xraySetting),
		"inboundTags":       json.RawMessage(inboundTags),
		"clientReverseTags": json.RawMessage(clientReverseTags),
		"outboundTestUrl":   outboundTestUrl,
	}
	result, err := json.Marshal(xrayResponse)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.getSettings"), err)
		return
	}
	jsonObj(c, string(result), nil)
}

// @Summary      Update Xray settings
// @Description  Updates the Xray configuration and outbound test URL.
// @Tags         Xray Settings
// @Accept       multipart/form-data
// @Produce      json
// @Param        xraySetting formData string true "Xray configuration JSON"
// @Param        outboundTestUrl formData string false "Outbound test URL"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/xray/update [post]
func (a *XraySettingController) updateSetting(c *gin.Context) {
	xraySetting := c.PostForm("xraySetting")
	if err := a.XraySettingService.SaveXraySetting(xraySetting); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.modifySettings"), err)
		return
	}
	outboundTestUrl := c.PostForm("outboundTestUrl")
	if outboundTestUrl == "" {
		outboundTestUrl = "https://www.google.com/generate_204"
	}
	if err := a.SettingService.SetXrayOutboundTestUrl(outboundTestUrl); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.modifySettings"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.settings.toasts.modifySettings"), nil)
}

// @Summary      Get default Xray config
// @Description  Returns the default Xray configuration template.
// @Tags         Xray Settings
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/xray/getDefaultJsonConfig [get]
func (a *XraySettingController) getDefaultXrayConfig(c *gin.Context) {
	defaultJsonConfig, err := a.SettingService.GetDefaultXrayConfig()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.getSettings"), err)
		return
	}
	jsonObj(c, defaultJsonConfig, nil)
}

// @Summary      Get Xray result
// @Description  Returns the current Xray process output/status.
// @Tags         Xray Settings
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/xray/getXrayResult [get]
func (a *XraySettingController) getXrayResult(c *gin.Context) {
	jsonObj(c, a.XrayService.GetXrayResult(), nil)
}

// @Summary      Warp operations
// @Description  Manages Warp integration: get data, delete, get config, register, set license.
// @Tags         Xray Settings
// @Accept       multipart/form-data
// @Produce      json
// @Param        action path string true "Action (data, del, config, reg, license)"
// @Param        privateKey formData string false "Private key for registration"
// @Param        publicKey formData string false "Public key for registration"
// @Param        license formData string false "License key"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/xray/warp/{action} [post]
func (a *XraySettingController) warp(c *gin.Context) {
	action := c.Param("action")
	var resp string
	var err error
	switch action {
	case "data":
		resp, err = a.WarpService.GetWarpData()
	case "del":
		err = a.WarpService.DelWarpData()
	case "config":
		resp, err = a.WarpService.GetWarpConfig()
	case "reg":
		skey := c.PostForm("privateKey")
		pkey := c.PostForm("publicKey")
		resp, err = a.WarpService.RegWarp(skey, pkey)
	case "license":
		license := c.PostForm("license")
		resp, err = a.WarpService.SetWarpLicense(license)
	}

	jsonObj(c, resp, err)
}

// @Summary      NordVPN operations
// @Description  Manages NordVPN integration: list countries, servers, register, set key, get data, delete.
// @Tags         Xray Settings
// @Accept       multipart/form-data
// @Produce      json
// @Param        action path string true "Action (countries, servers, reg, setKey, data, del)"
// @Param        countryId formData string false "Country ID for server list"
// @Param        token formData string false "NordVPN API token for registration"
// @Param        key formData string false "Access key"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/xray/nord/{action} [post]
func (a *XraySettingController) nord(c *gin.Context) {
	action := c.Param("action")
	var resp string
	var err error
	switch action {
	case "countries":
		resp, err = a.NordService.GetCountries()
	case "servers":
		countryId := c.PostForm("countryId")
		resp, err = a.NordService.GetServers(countryId)
	case "reg":
		token := c.PostForm("token")
		resp, err = a.NordService.GetCredentials(token)
	case "setKey":
		key := c.PostForm("key")
		resp, err = a.NordService.SetKey(key)
	case "data":
		resp, err = a.NordService.GetNordData()
	case "del":
		err = a.NordService.DelNordData()
	}

	jsonObj(c, resp, err)
}

// @Summary      Get outbounds traffic
// @Description  Returns traffic statistics for all Xray outbounds.
// @Tags         Xray Settings
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/xray/getOutboundsTraffic [get]
func (a *XraySettingController) getOutboundsTraffic(c *gin.Context) {
	outboundsTraffic, err := a.OutboundService.GetOutboundsTraffic()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.getOutboundTrafficError"), err)
		return
	}
	jsonObj(c, outboundsTraffic, nil)
}

// @Summary      Reset outbound traffic
// @Description  Resets traffic statistics for a specified outbound by tag.
// @Tags         Xray Settings
// @Accept       multipart/form-data
// @Produce      json
// @Param        tag formData string true "Outbound tag to reset"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/xray/resetOutboundsTraffic [post]
func (a *XraySettingController) resetOutboundsTraffic(c *gin.Context) {
	tag := c.PostForm("tag")
	err := a.OutboundService.ResetOutboundTraffic(tag)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.settings.toasts.resetOutboundTrafficError"), err)
		return
	}
	jsonObj(c, "", nil)
}

// @Summary      Test outbound
// @Description  Tests an outbound configuration and returns the delay/response time. Supports TCP probe mode for parallel-safe testing.
// @Tags         Xray Settings
// @Accept       multipart/form-data
// @Produce      json
// @Param        outbound formData string true "Outbound configuration JSON"
// @Param        allOutbounds formData string false "JSON array of all outbounds (for dependency resolution)"
// @Param        mode formData string false "Test mode: 'tcp' for fast dial-only probe"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/xray/testOutbound [post]
func (a *XraySettingController) testOutbound(c *gin.Context) {
	outboundJSON := c.PostForm("outbound")
	allOutboundsJSON := c.PostForm("allOutbounds")
	mode := c.PostForm("mode")

	if outboundJSON == "" {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), common.NewError("outbound parameter is required"))
		return
	}

	// Load the test URL from server settings to prevent SSRF via user-controlled URLs
	testURL, _ := a.SettingService.GetXrayOutboundTestUrl()

	result, err := a.OutboundService.TestOutbound(outboundJSON, testURL, allOutboundsJSON, mode)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}

	jsonObj(c, result, nil)
}
