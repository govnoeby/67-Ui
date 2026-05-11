package controller

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/mhsanaei/3x-ui/v3/logger"
	"github.com/mhsanaei/3x-ui/v3/util/crypto"
	"github.com/mhsanaei/3x-ui/v3/web/entity"
	"github.com/mhsanaei/3x-ui/v3/web/global"
	"github.com/mhsanaei/3x-ui/v3/web/service"
	"github.com/mhsanaei/3x-ui/v3/web/websocket"

	"github.com/gin-gonic/gin"
)

var filenameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-.]+$`)

// ServerController handles server management and status-related operations.
type ServerController struct {
	BaseController

	serverService  service.ServerService
	settingService service.SettingService
	panelService   service.PanelService
	userService    service.UserService

	lastStatus *service.Status

	lastVersions        []string
	lastGetVersionsTime int64 // unix seconds
}

// NewServerController creates a new ServerController, initializes routes, and starts background tasks.
func NewServerController(g *gin.RouterGroup, userService service.UserService) *ServerController {
	a := &ServerController{
		userService: userService,
	}
	a.initRouter(g)
	a.startTask()
	return a
}

// initRouter sets up the routes for server status, Xray management, and utility endpoints.
func (a *ServerController) initRouter(g *gin.RouterGroup) {

	g.GET("/status", a.status)
	g.GET("/cpuHistory/:bucket", a.getCpuHistoryBucket)
	g.GET("/history/:metric/:bucket", a.getMetricHistoryBucket)
	g.GET("/getXrayVersion", a.getXrayVersion)
	g.GET("/getPanelUpdateInfo", a.getPanelUpdateInfo)
	g.GET("/getConfigJson", a.getConfigJson)
	g.POST("/getDb", a.getDb)
	g.GET("/getNewUUID", a.getNewUUID)
	g.GET("/getNewX25519Cert", a.getNewX25519Cert)
	g.GET("/getNewmldsa65", a.getNewmldsa65)
	g.GET("/getNewmlkem768", a.getNewmlkem768)
	g.GET("/getNewVlessEnc", a.getNewVlessEnc)

	g.POST("/stopXrayService", a.stopXrayService)
	g.POST("/restartXrayService", a.restartXrayService)
	g.POST("/installXray/:version", a.installXray)
	g.POST("/updatePanel", a.updatePanel)
	g.POST("/updateGeofile", a.updateGeofile)
	g.POST("/updateGeofile/:fileName", a.updateGeofile)
	g.POST("/logs/:count", a.getLogs)
	g.POST("/xraylogs/:count", a.getXrayLogs)
	g.POST("/importDB", a.importDB)
	g.POST("/getNewEchCert", a.getNewEchCert)
}

// refreshStatus updates the cached server status and collects time-series
// metrics. CPU/Mem/Net/Online/Load are all written in one call so the
// SystemHistoryModal's tabs share an identical x-axis.
func (a *ServerController) refreshStatus() {
	a.lastStatus = a.serverService.GetStatus(a.lastStatus)
	if a.lastStatus != nil {
		a.serverService.AppendStatusSample(time.Now(), a.lastStatus)
		// Broadcast status update via WebSocket
		websocket.BroadcastStatus(a.lastStatus)
	}
}

// startTask initiates background tasks for continuous status monitoring.
func (a *ServerController) startTask() {
	webServer := global.GetWebServer()
	c := webServer.GetCron()
	c.AddFunc("@every 2s", func() {
		// Always refresh to keep CPU history collected continuously.
		// Sampling is lightweight and capped to ~6 hours in memory.
		a.refreshStatus()
	})
}

// @Summary      Get server status
// @Description  Returns real-time machine snapshot: CPU, memory, swap, disk, network IO, load averages, open connections, Xray state. Cached and refreshed every 2 seconds.
// @Tags         Server
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/status [get]
func (a *ServerController) status(c *gin.Context) { jsonObj(c, a.lastStatus, nil) }

// allowedHistoryBuckets is the bucket-second whitelist shared by both
// /cpuHistory/:bucket and /history/:metric/:bucket. Restricting it
// prevents callers from triggering arbitrary aggregation work and keeps
// the front-end's bucket selector self-documenting.
var allowedHistoryBuckets = map[int]bool{
	2:   true, // Real-time view
	30:  true, // 30s intervals
	60:  true, // 1m intervals
	120: true, // 2m intervals
	180: true, // 3m intervals
	300: true, // 5m intervals
}

// @Summary      Get CPU history (legacy)
// @Description  Legacy: aggregated CPU history. Use /history/cpu/{bucket} instead.
// @Tags         Server
// @Produce      json
// @Param        bucket path int true "Bucket size in seconds (2, 30, 60, 120, 180, 300)"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/cpuHistory/{bucket} [get]
func (a *ServerController) getCpuHistoryBucket(c *gin.Context) {
	bucketStr := c.Param("bucket")
	bucket, err := strconv.Atoi(bucketStr)
	if err != nil || bucket <= 0 {
		jsonMsg(c, "invalid bucket", fmt.Errorf("bad bucket"))
		return
	}
	if !allowedHistoryBuckets[bucket] {
		jsonMsg(c, "invalid bucket", fmt.Errorf("unsupported bucket"))
		return
	}
	points := a.serverService.AggregateCpuHistory(bucket, 60)
	jsonObj(c, points, nil)
}

// @Summary      Get metric history
// @Description  Returns up to 60 buckets of aggregated time-series data for a single metric (cpu, mem, swap, netIn, netOut, tcpCount, udpCount, load1, online).
// @Tags         Server
// @Produce      json
// @Param        metric path string true "Metric name (cpu, mem, swap, netIn, netOut, tcpCount, udpCount, load1, online)"
// @Param        bucket path int true "Bucket size in seconds (2, 30, 60, 120, 180, 300)"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/history/{metric}/{bucket} [get]
func (a *ServerController) getMetricHistoryBucket(c *gin.Context) {
	metric := c.Param("metric")
	if !slices.Contains(service.SystemMetricKeys, metric) {
		jsonMsg(c, "invalid metric", fmt.Errorf("unknown metric"))
		return
	}
	bucket, err := strconv.Atoi(c.Param("bucket"))
	if err != nil || bucket <= 0 || !allowedHistoryBuckets[bucket] {
		jsonMsg(c, "invalid bucket", fmt.Errorf("unsupported bucket"))
		return
	}
	jsonObj(c, a.serverService.AggregateSystemMetric(metric, bucket, 60), nil)
}

// @Summary      Get available Xray versions
// @Description  Lists Xray binary versions available for install on this host. Cached for 15 minutes.
// @Tags         Server
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/getXrayVersion [get]
func (a *ServerController) getXrayVersion(c *gin.Context) {
	const cacheTTLSeconds = 15 * 60

	now := time.Now().Unix()
	if a.lastVersions != nil && now-a.lastGetVersionsTime <= cacheTTLSeconds {
		jsonObj(c, a.lastVersions, nil)
		return
	}

	versions, err := a.serverService.GetXrayVersions()
	if err != nil {
		if a.lastVersions != nil {
			logger.Warning("getXrayVersion failed; serving cached list:", err)
			jsonObj(c, a.lastVersions, nil)
			return
		}
		jsonMsg(c, I18nWeb(c, "getVersion"), err)
		return
	}

	a.lastVersions = versions
	a.lastGetVersionsTime = now

	jsonObj(c, versions, nil)
}

// @Summary      Check panel updates
// @Description  Checks whether a newer 3x-ui release is available on GitHub.
// @Tags         Server
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/getPanelUpdateInfo [get]
func (a *ServerController) getPanelUpdateInfo(c *gin.Context) {
	info, err := a.panelService.GetUpdateInfo()
	if err != nil {
		logger.Debug("panel update check failed:", err)
		c.JSON(http.StatusOK, entity.Msg{Success: false})
		return
	}
	jsonObj(c, info, nil)
}

// @Summary      Install/update Xray
// @Description  Downloads and installs the specified Xray version. Pass "latest" for the newest release.
// @Tags         Server
// @Produce      json
// @Param        version path string true "Xray tag (e.g. v25.10.31) or 'latest'"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/installXray/{version} [post]
func (a *ServerController) installXray(c *gin.Context) {
	version := c.Param("version")
	err := a.serverService.UpdateXray(version)
	jsonMsg(c, I18nWeb(c, "pages.index.xraySwitchVersionPopover"), err)
}

// @Summary      Self-update panel
// @Description  Updates the panel to the latest release. The server restarts on success.
// @Tags         Server
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/updatePanel [post]
func (a *ServerController) updatePanel(c *gin.Context) {
	err := a.panelService.StartUpdate()
	jsonMsg(c, I18nWeb(c, "pages.index.panelUpdateStartedPopover"), err)
}

// @Summary      Update geo file
// @Description  Refreshes a GeoIP/GeoSite data file. Can specify filename or use default.
// @Tags         Server
// @Accept       multipart/form-data
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/updateGeofile [post]
func (a *ServerController) updateGeofile(c *gin.Context) {
	fileName := c.Param("fileName")

	// Validate the filename for security (prevent path traversal attacks)
	if fileName != "" && !a.serverService.IsValidGeofileName(fileName) {
		jsonMsg(c, I18nWeb(c, "pages.index.geofileUpdatePopover"),
			fmt.Errorf("invalid filename: contains unsafe characters or path traversal patterns"))
		return
	}

	err := a.serverService.UpdateGeofile(fileName)
	jsonMsg(c, I18nWeb(c, "pages.index.geofileUpdatePopover"), err)
}

// @Summary      Stop Xray service
// @Description  Stops the Xray binary. All proxies go offline immediately.
// @Tags         Server
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/stopXrayService [post]
func (a *ServerController) stopXrayService(c *gin.Context) {
	err := a.serverService.StopXrayService()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.xray.stopError"), err)
		websocket.BroadcastXrayState("error", err.Error())
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.xray.stopSuccess"), err)
	websocket.BroadcastXrayState("stop", "")
	websocket.BroadcastNotification(
		I18nWeb(c, "pages.xray.stopSuccess"),
		"Xray service has been stopped",
		"warning",
	)
}

// @Summary      Restart Xray service
// @Description  Reloads Xray with the current configuration. Required after structural inbound or routing changes.
// @Tags         Server
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/restartXrayService [post]
func (a *ServerController) restartXrayService(c *gin.Context) {
	err := a.serverService.RestartXrayService()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.xray.restartError"), err)
		websocket.BroadcastXrayState("error", err.Error())
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.xray.restartSuccess"), err)
	websocket.BroadcastXrayState("running", "")
	websocket.BroadcastNotification(
		I18nWeb(c, "pages.xray.restartSuccess"),
		"Xray service has been restarted successfully",
		"success",
	)
}

// @Summary      Get panel logs
// @Description  Returns the last N lines of the panel's log file with optional level and syslog filtering.
// @Tags         Server
// @Accept       multipart/form-data
// @Produce      json
// @Param        count path int true "Number of trailing log lines"
// @Param        level formData string false "Log level filter (info, warning, error, debug)"
// @Param        syslog formData bool false "Include syslog output"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/logs/{count} [post]
func (a *ServerController) getLogs(c *gin.Context) {
	count := c.Param("count")
	level := c.PostForm("level")
	syslog := c.PostForm("syslog")
	logs := a.serverService.GetLogs(count, level, syslog)
	jsonObj(c, logs, nil)
}

// @Summary      Get Xray logs
// @Description  Returns the last N lines of the Xray process log with optional traffic type filtering.
// @Tags         Server
// @Accept       multipart/form-data
// @Produce      json
// @Param        count path int true "Number of trailing log lines"
// @Param        filter formData string false "Log filter string"
// @Param        showDirect formData string false "Show direct traffic logs"
// @Param        showBlocked formData string false "Show blocked traffic logs"
// @Param        showProxy formData string false "Show proxy traffic logs"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/xraylogs/{count} [post]
func (a *ServerController) getXrayLogs(c *gin.Context) {
	count := c.Param("count")
	filter := c.PostForm("filter")
	showDirect := c.PostForm("showDirect")
	showBlocked := c.PostForm("showBlocked")
	showProxy := c.PostForm("showProxy")

	var freedoms []string
	var blackholes []string

	//getting tags for freedom and blackhole outbounds
	config, err := a.settingService.GetDefaultXrayConfig()
	if err == nil && config != nil {
		if cfgMap, ok := config.(map[string]any); ok {
			if outbounds, ok := cfgMap["outbounds"].([]any); ok {
				for _, outbound := range outbounds {
					if obMap, ok := outbound.(map[string]any); ok {
						switch obMap["protocol"] {
						case "freedom":
							if tag, ok := obMap["tag"].(string); ok {
								freedoms = append(freedoms, tag)
							}
						case "blackhole":
							if tag, ok := obMap["tag"].(string); ok {
								blackholes = append(blackholes, tag)
							}
						}
					}
				}
			}
		}
	}

	if len(freedoms) == 0 {
		freedoms = []string{"direct"}
	}
	if len(blackholes) == 0 {
		blackholes = []string{"blocked"}
	}

	logs := a.serverService.GetXrayLogs(count, filter, showDirect, showBlocked, showProxy, freedoms, blackholes)
	jsonObj(c, logs, nil)
}

// @Summary      Get Xray config JSON
// @Description  Returns the assembled Xray configuration that is currently running on this host.
// @Tags         Server
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/getConfigJson [get]
func (a *ServerController) getConfigJson(c *gin.Context) {
	configJson, err := a.serverService.GetConfigJson()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.index.getConfigError"), err)
		return
	}
	jsonObj(c, configJson, nil)
}

// getDbPasswordRequest represents the expected JSON body for database export.
type getDbPasswordRequest struct {
	Password string `json:"password"`
}

// @Summary      Export database
// @Description  Exports the SQLite database file as an attachment. Requires password verification.
// @Tags         Server
// @Accept       json
// @Produce      application/octet-stream
// @Param        body body getDbPasswordRequest true "Password for verification"
// @Success      200 {file} binary "Database file download"
// @Security     BearerAuth
// @Router       /panel/api/server/getDb [post]
func (a *ServerController) getDb(c *gin.Context) {
	var req getDbPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Password == "" {
		pureJsonMsg(c, http.StatusOK, false, "Password is required")
		return
	}

	user, err := a.userService.GetFirstUser()
	if err != nil {
		pureJsonMsg(c, http.StatusOK, false, "An error occurred while retrieving the database.")
		logger.Errorf("getDb: failed to lookup user: %v", err)
		return
	}

	if !crypto.CheckPasswordHash(user.Password, req.Password) {
		logger.Warningf("getDb: wrong password attempt from %s", getRemoteIp(c))
		pureJsonMsg(c, http.StatusOK, false, "Invalid password")
		return
	}

	db, err := a.serverService.GetDb()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.index.getDatabaseError"), err)
		return
	}

	logger.Infof("getDb: database exported by %s from %s", user.Username, getRemoteIp(c))

	filename := "x-ui.db"

	if !isValidFilename(filename) {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("invalid filename"))
		return
	}

	// Set the headers for the response
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename="+filename)

	// Write the file contents to the response
	c.Writer.Write(db)
}

func isValidFilename(filename string) bool {
	// Validate that the filename only contains allowed characters
	return filenameRegex.MatchString(filename)
}

// @Summary      Import database
// @Description  Restores the panel DB from an uploaded SQLite file. Panel restarts after restore. Destructive.
// @Tags         Server
// @Accept       multipart/form-data
// @Produce      json
// @Param        db formData file true "SQLite database file"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/importDB [post]
func (a *ServerController) importDB(c *gin.Context) {
	// Get the file from the request body
	file, _, err := c.Request.FormFile("db")
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.index.readDatabaseError"), err)
		return
	}
	defer file.Close()
	err = a.serverService.ImportDB(file)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.index.importDatabaseError"), err)
		return
	}
	jsonObj(c, I18nWeb(c, "pages.index.importDatabaseSuccess"), nil)
}

// @Summary      Generate X25519 keypair
// @Description  Generates a new X25519 keypair for Reality TLS. Returns public and private keys.
// @Tags         Server
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/getNewX25519Cert [get]
func (a *ServerController) getNewX25519Cert(c *gin.Context) {
	cert, err := a.serverService.GetNewX25519Cert()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.getNewX25519CertError"), err)
		return
	}
	jsonObj(c, cert, nil)
}

// @Summary      Generate ML-DSA-65 keypair
// @Description  Generates a new ML-DSA-65 post-quantum signature keypair.
// @Tags         Server
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/getNewmldsa65 [get]
func (a *ServerController) getNewmldsa65(c *gin.Context) {
	cert, err := a.serverService.GetNewmldsa65()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.getNewmldsa65Error"), err)
		return
	}
	jsonObj(c, cert, nil)
}

// @Summary      Generate ECH keypair
// @Description  Generates a new Encrypted Client Hello (ECH) keypair.
// @Tags         Server
// @Accept       multipart/form-data
// @Produce      json
// @Param        sni formData string false "SNI for ECH certificate"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/getNewEchCert [post]
func (a *ServerController) getNewEchCert(c *gin.Context) {
	sni := c.PostForm("sni")
	cert, err := a.serverService.GetNewEchCert(sni)
	if err != nil {
		jsonMsg(c, "get ech certificate", err)
		return
	}
	jsonObj(c, cert, nil)
}

// @Summary      Generate VLESS encryption key
// @Description  Generates a new VLESS encryption keypair.
// @Tags         Server
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/getNewVlessEnc [get]
func (a *ServerController) getNewVlessEnc(c *gin.Context) {
	out, err := a.serverService.GetNewVlessEnc()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.getNewVlessEncError"), err)
		return
	}
	jsonObj(c, out, nil)
}

// @Summary      Generate UUID
// @Description  Generates a fresh UUID v4. Convenience helper for client IDs.
// @Tags         Server
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/getNewUUID [get]
func (a *ServerController) getNewUUID(c *gin.Context) {
	uuidResp, err := a.serverService.GetNewUUID()
	if err != nil {
		jsonMsg(c, "Failed to generate UUID", err)
		return
	}

	jsonObj(c, uuidResp, nil)
}

// @Summary      Generate ML-KEM-768 keypair
// @Description  Generates a new ML-KEM-768 post-quantum KEM keypair.
// @Tags         Server
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/server/getNewmlkem768 [get]
func (a *ServerController) getNewmlkem768(c *gin.Context) {
	out, err := a.serverService.GetNewmlkem768()
	if err != nil {
		jsonMsg(c, "Failed to generate mlkem768 keys", err)
		return
	}
	jsonObj(c, out, nil)
}
