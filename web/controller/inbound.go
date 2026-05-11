package controller

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/database/model"
	"github.com/mhsanaei/3x-ui/v3/web/service"
	"github.com/mhsanaei/3x-ui/v3/web/session"
	"github.com/mhsanaei/3x-ui/v3/web/websocket"

	"github.com/gin-gonic/gin"
)

// InboundController handles HTTP requests related to Xray inbounds management.
type InboundController struct {
	inboundService service.InboundService
	xrayService    service.XrayService
}

// NewInboundController creates a new InboundController and sets up its routes.
func NewInboundController(g *gin.RouterGroup) *InboundController {
	a := &InboundController{}
	a.initRouter(g)
	return a
}

// broadcastInboundsUpdateClientLimit is the threshold past which we skip the
// full-list push over WebSocket and signal the frontend to re-fetch via REST.
// Mirrors the same heuristic used by the periodic traffic job.
const broadcastInboundsUpdateClientLimit = 5000

// broadcastInboundsUpdate fetches and broadcasts the inbound list for userId.
// At scale (10k+ clients) the marshaled JSON exceeds the WS payload ceiling,
// so we send an invalidate signal instead — frontend re-fetches via REST.
// Skipped entirely when no WebSocket clients are connected.
func (a *InboundController) broadcastInboundsUpdate(userId int) {
	if !websocket.HasClients() {
		return
	}
	inbounds, err := a.inboundService.GetInbounds(userId)
	if err != nil {
		return
	}
	totalClients := 0
	for _, ib := range inbounds {
		totalClients += len(ib.ClientStats)
	}
	if totalClients > broadcastInboundsUpdateClientLimit {
		websocket.BroadcastInvalidate(websocket.MessageTypeInbounds)
		return
	}
	websocket.BroadcastInbounds(inbounds)
}

// initRouter initializes the routes for inbound-related operations.
func (a *InboundController) initRouter(g *gin.RouterGroup) {

	g.GET("/list", a.getInbounds)
	g.GET("/get/:id", a.getInbound)
	g.GET("/getClientTraffics/:email", a.getClientTraffics)
	g.GET("/getClientTrafficsById/:id", a.getClientTrafficsById)
	g.GET("/getSubLinks/:subId", a.getSubLinks)
	g.GET("/getClientLinks/:id/:email", a.getClientLinks)

	g.POST("/add", a.addInbound)
	g.POST("/del/:id", a.delInbound)
	g.POST("/update/:id", a.updateInbound)
	g.POST("/setEnable/:id", a.setInboundEnable)
	g.POST("/clientIps/:email", a.getClientIps)
	g.POST("/clearClientIps/:email", a.clearClientIps)
	g.POST("/addClient", a.addInboundClient)
	g.POST("/:id/copyClients", a.copyInboundClients)
	g.POST("/:id/delClient/:clientId", a.delInboundClient)
	g.POST("/updateClient/:clientId", a.updateInboundClient)
	g.POST("/:id/resetClientTraffic/:email", a.resetClientTraffic)
	g.POST("/resetAllTraffics", a.resetAllTraffics)
	g.POST("/resetAllClientTraffics/:id", a.resetAllClientTraffics)
	g.POST("/delDepletedClients/:id", a.delDepletedClients)
	g.POST("/import", a.importInbound)
	g.POST("/onlines", a.onlines)
	g.POST("/lastOnline", a.lastOnline)
	g.POST("/updateClientTraffic/:email", a.updateClientTraffic)
	g.POST("/:id/delClientByEmail/:email", a.delInboundClientByEmail)
}

type CopyInboundClientsRequest struct {
	SourceInboundID int      `form:"sourceInboundId" json:"sourceInboundId"`
	ClientEmails    []string `form:"clientEmails" json:"clientEmails"`
	Flow            string   `form:"flow" json:"flow"`
}

// @Summary      List inbounds
// @Description  Returns all inbounds owned by the authenticated user, including client traffic stats.
// @Tags         Inbounds
// @Produce      json
// @Success      200 {object} entity.Msg{obj=[]model.Inbound}
// @Security     BearerAuth
// @Router       /panel/api/inbounds/list [get]
func (a *InboundController) getInbounds(c *gin.Context) {
	user := session.GetLoginUser(c)
	inbounds, err := a.inboundService.GetInbounds(user.Id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.obtain"), err)
		return
	}
	jsonObj(c, inbounds, nil)
}

// @Summary      Get inbound by ID
// @Description  Fetch a single inbound configuration with traffic stats.
// @Tags         Inbounds
// @Produce      json
// @Param        id path int true "Inbound ID"
// @Success      200 {object} entity.Msg{obj=model.Inbound}
// @Security     BearerAuth
// @Router       /panel/api/inbounds/get/{id} [get]
func (a *InboundController) getInbound(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "get"), err)
		return
	}
	inbound, err := a.inboundService.GetInbound(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.obtain"), err)
		return
	}
	jsonObj(c, inbound, nil)
}

// @Summary      Get client traffic by email
// @Description  Returns traffic counters for a client identified by email.
// @Tags         Inbounds
// @Produce      json
// @Param        email path string true "Client email"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/getClientTraffics/{email} [get]
func (a *InboundController) getClientTraffics(c *gin.Context) {
	email := c.Param("email")
	clientTraffics, err := a.inboundService.GetClientTrafficByEmail(email)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.trafficGetError"), err)
		return
	}
	jsonObj(c, clientTraffics, nil)
}

// @Summary      Get client traffic by ID
// @Description  Returns traffic counters for a client identified by its UUID/password.
// @Tags         Inbounds
// @Produce      json
// @Param        id path string true "Client UUID or password"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/getClientTrafficsById/{id} [get]
func (a *InboundController) getClientTrafficsById(c *gin.Context) {
	id := c.Param("id")
	clientTraffics, err := a.inboundService.GetClientTrafficByID(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.trafficGetError"), err)
		return
	}
	jsonObj(c, clientTraffics, nil)
}

// @Summary      Create inbound
// @Description  Creates a new inbound configuration with protocol, port, stream settings, and clients.
// @Tags         Inbounds
// @Accept       json
// @Produce      json
// @Param        body body model.Inbound true "Inbound configuration"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/add [post]
func (a *InboundController) addInbound(c *gin.Context) {
	inbound := &model.Inbound{}
	err := c.ShouldBind(inbound)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundCreateSuccess"), err)
		return
	}
	user := session.GetLoginUser(c)
	inbound.UserId = user.Id
	// Treat NodeID=0 as "no node" — gin's *int form binding can land on
	// 0 when the field is absent or empty, and 0 is never a valid Node
	// row id. Without this normalization the runtime layer would try to
	// load Node id=0 and surface "record not found".
	if inbound.NodeID != nil && *inbound.NodeID == 0 {
		inbound.NodeID = nil
	}
	// When the central panel deploys an inbound to a remote node, it sends
	// the Tag pre-computed (so both DBs agree on the identifier). Local
	// UI submits don't include a Tag — we compute one from listen+port
	// using the original collision-avoiding scheme.
	if inbound.Tag == "" {
		if inbound.Listen == "" || inbound.Listen == "0.0.0.0" || inbound.Listen == "::" || inbound.Listen == "::0" {
			inbound.Tag = fmt.Sprintf("inbound-%v", inbound.Port)
		} else {
			inbound.Tag = fmt.Sprintf("inbound-%v:%v", inbound.Listen, inbound.Port)
		}
	}

	inbound, needRestart, err := a.inboundService.AddInbound(inbound)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsgObj(c, I18nWeb(c, "pages.inbounds.toasts.inboundCreateSuccess"), inbound, nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
	a.broadcastInboundsUpdate(user.Id)
}

// @Summary      Delete inbound
// @Description  Deletes an inbound and its associated client traffic stats by ID.
// @Tags         Inbounds
// @Produce      json
// @Param        id path int true "Inbound ID"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/del/{id} [post]
func (a *InboundController) delInbound(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundDeleteSuccess"), err)
		return
	}
	needRestart, err := a.inboundService.DelInbound(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsgObj(c, I18nWeb(c, "pages.inbounds.toasts.inboundDeleteSuccess"), id, nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
	user := session.GetLoginUser(c)
	a.broadcastInboundsUpdate(user.Id)
}

// @Summary      Update inbound
// @Description  Replaces an inbound configuration. Body shape matches /add.
// @Tags         Inbounds
// @Accept       json
// @Produce      json
// @Param        id path int true "Inbound ID"
// @Param        body body model.Inbound true "Updated inbound configuration"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/update/{id} [post]
func (a *InboundController) updateInbound(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}
	inbound := &model.Inbound{
		Id: id,
	}
	err = c.ShouldBind(inbound)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}
	// Same NodeID=0 → nil normalisation as addInbound. UpdateInbound
	// loads the existing row's NodeID from DB anyway (Phase 1 doesn't
	// support migrating an inbound between nodes), but normalising here
	// keeps the wire shape consistent.
	if inbound.NodeID != nil && *inbound.NodeID == 0 {
		inbound.NodeID = nil
	}
	inbound, needRestart, err := a.inboundService.UpdateInbound(inbound)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsgObj(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), inbound, nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
	user := session.GetLoginUser(c)
	a.broadcastInboundsUpdate(user.Id)
}

// @Summary      Toggle inbound enable
// @Description  Flips the enable flag without serialising the full settings JSON. Lightweight alternative to /update for UI switches.
// @Tags         Inbounds
// @Produce      json
// @Param        id path int true "Inbound ID"
// @Param        body body object{enable=bool} true "Enable flag"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/setEnable/{id} [post]
func (a *InboundController) setInboundEnable(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}
	type form struct {
		Enable bool `json:"enable" form:"enable"`
	}
	var f form
	if err := c.ShouldBind(&f); err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	needRestart, err := a.inboundService.SetInboundEnable(id, f.Enable)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
	// Cross-admin sync: lightweight invalidate signal (a few hundred bytes)
	// instead of fetching + serialising the whole inbound list. Other open
	// sessions re-fetch via REST. The toggling admin's own UI already
	// updated optimistically.
	websocket.BroadcastInvalidate(websocket.MessageTypeInbounds)
}

// @Summary      Get client IPs
// @Description  Returns source IPs that have connected with the given client's credentials. Includes timestamps.
// @Tags         Inbounds
// @Produce      json
// @Param        email path string true "Client email"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/clientIps/{email} [post]
func (a *InboundController) getClientIps(c *gin.Context) {
	email := c.Param("email")

	ips, err := a.inboundService.GetInboundClientIps(email)
	if err != nil || ips == "" {
		jsonObj(c, "No IP Record", nil)
		return
	}

	// Prefer returning a normalized string list for consistent UI rendering
	type ipWithTimestamp struct {
		IP        string `json:"ip"`
		Timestamp int64  `json:"timestamp"`
	}

	var ipsWithTime []ipWithTimestamp
	if err := json.Unmarshal([]byte(ips), &ipsWithTime); err == nil && len(ipsWithTime) > 0 {
		formatted := make([]string, 0, len(ipsWithTime))
		for _, item := range ipsWithTime {
			if item.IP == "" {
				continue
			}
			if item.Timestamp > 0 {
				ts := time.Unix(item.Timestamp, 0).Local().Format("2006-01-02 15:04:05")
				formatted = append(formatted, fmt.Sprintf("%s (%s)", item.IP, ts))
				continue
			}
			formatted = append(formatted, item.IP)
		}
		jsonObj(c, formatted, nil)
		return
	}

	var oldIps []string
	if err := json.Unmarshal([]byte(ips), &oldIps); err == nil && len(oldIps) > 0 {
		jsonObj(c, oldIps, nil)
		return
	}

	// If parsing fails, return as string
	jsonObj(c, ips, nil)
}

// @Summary      Clear client IPs
// @Description  Resets the recorded IP address list for a client.
// @Tags         Inbounds
// @Produce      json
// @Param        email path string true "Client email"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/clearClientIps/{email} [post]
func (a *InboundController) clearClientIps(c *gin.Context) {
	email := c.Param("email")

	err := a.inboundService.ClearClientIps(email)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.updateSuccess"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.logCleanSuccess"), nil)
}

// @Summary      Add client to inbound
// @Description  Adds one or more clients to an existing inbound. The settings field carries the JSON-encoded clients array.
// @Tags         Inbounds
// @Accept       json
// @Produce      json
// @Param        body body model.Inbound true "Inbound with new client settings"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/addClient [post]
func (a *InboundController) addInboundClient(c *gin.Context) {
	data := &model.Inbound{}
	err := c.ShouldBind(data)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}

	needRestart, err := a.inboundService.AddInboundClient(data)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundClientAddSuccess"), nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
}

// @Summary      Copy clients between inbounds
// @Description  Copies selected clients from one inbound to another. Useful for duplicating user lists across protocols.
// @Tags         Inbounds
// @Accept       json
// @Produce      json
// @Param        id path int true "Target inbound ID"
// @Param        body body CopyInboundClientsRequest true "Copy request"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/{id}/copyClients [post]
func (a *InboundController) copyInboundClients(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}

	req := &CopyInboundClientsRequest{}
	err = c.ShouldBind(req)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	if req.SourceInboundID <= 0 {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), fmt.Errorf("invalid source inbound id"))
		return
	}

	result, needRestart, err := a.inboundService.CopyInboundClients(targetID, req.SourceInboundID, req.ClientEmails, req.Flow)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonObj(c, result, nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
}

// @Summary      Delete client from inbound
// @Description  Removes a client from an inbound by its UUID/password.
// @Tags         Inbounds
// @Produce      json
// @Param        id path int true "Inbound ID"
// @Param        clientId path string true "Client UUID or password"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/{id}/delClient/{clientId} [post]
func (a *InboundController) delInboundClient(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}
	clientId := c.Param("clientId")

	needRestart, err := a.inboundService.DelInboundClient(id, clientId)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundClientDeleteSuccess"), nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
}

// @Summary      Update client in inbound
// @Description  Updates a single client's configuration without rewriting the whole settings JSON.
// @Tags         Inbounds
// @Accept       json
// @Produce      json
// @Param        clientId path string true "Client UUID or password"
// @Param        body body model.Inbound true "Inbound payload with updated client settings"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/updateClient/{clientId} [post]
func (a *InboundController) updateInboundClient(c *gin.Context) {
	clientId := c.Param("clientId")

	inbound := &model.Inbound{}
	err := c.ShouldBind(inbound)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}

	needRestart, err := a.inboundService.UpdateInboundClient(inbound, clientId)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundClientUpdateSuccess"), nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
}

// @Summary      Reset client traffic
// @Description  Zeros out upload and download counters for a single client.
// @Tags         Inbounds
// @Produce      json
// @Param        id path int true "Inbound ID"
// @Param        email path string true "Client email"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/{id}/resetClientTraffic/{email} [post]
func (a *InboundController) resetClientTraffic(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}
	email := c.Param("email")

	needRestart, err := a.inboundService.ResetClientTraffic(id, email)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.resetInboundClientTrafficSuccess"), nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
}

// @Summary      Reset all traffic
// @Description  Resets upload and download counters on every inbound. Destructive — accounting history is lost.
// @Tags         Inbounds
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/resetAllTraffics [post]
func (a *InboundController) resetAllTraffics(c *gin.Context) {
	err := a.inboundService.ResetAllTraffics()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	} else {
		a.xrayService.SetToNeedRestart()
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.resetAllTrafficSuccess"), nil)
}

// @Summary      Reset all client traffic in inbound
// @Description  Resets traffic for every client in a specific inbound.
// @Tags         Inbounds
// @Produce      json
// @Param        id path int true "Inbound ID"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/resetAllClientTraffics/{id} [post]
func (a *InboundController) resetAllClientTraffics(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}

	err = a.inboundService.ResetAllClientTraffics(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	} else {
		a.xrayService.SetToNeedRestart()
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.resetAllClientTrafficSuccess"), nil)
}

// @Summary      Import inbound
// @Description  Bulk-imports an inbound from a JSON blob (e.g., exported from another panel). Uses form encoding with a single "data" field.
// @Tags         Inbounds
// @Accept       multipart/form-data
// @Produce      json
// @Param        data formData string true "JSON-encoded inbound configuration"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/import [post]
func (a *InboundController) importInbound(c *gin.Context) {
	inbound := &model.Inbound{}
	err := json.Unmarshal([]byte(c.PostForm("data")), inbound)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	user := session.GetLoginUser(c)
	inbound.Id = 0
	inbound.UserId = user.Id
	if inbound.Tag == "" {
		if inbound.Listen == "" || inbound.Listen == "0.0.0.0" || inbound.Listen == "::" || inbound.Listen == "::0" {
			inbound.Tag = fmt.Sprintf("inbound-%v", inbound.Port)
		} else {
			inbound.Tag = fmt.Sprintf("inbound-%v:%v", inbound.Listen, inbound.Port)
		}
	}

	for index := range inbound.ClientStats {
		inbound.ClientStats[index].Id = 0
		inbound.ClientStats[index].Enable = true
	}

	needRestart := false
	inbound, needRestart, err = a.inboundService.AddInbound(inbound)
	jsonMsgObj(c, I18nWeb(c, "pages.inbounds.toasts.inboundCreateSuccess"), inbound, err)
	if err == nil && needRestart {
		a.xrayService.SetToNeedRestart()
	}
}

// @Summary      Delete depleted clients
// @Description  Removes clients whose traffic cap or expiry has elapsed. Pass id=-1 to sweep every inbound.
// @Tags         Inbounds
// @Produce      json
// @Param        id path int true "Inbound ID, or -1 for all"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/delDepletedClients/{id} [post]
func (a *InboundController) delDepletedClients(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}
	err = a.inboundService.DelDepletedClients(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.delDepletedClientsSuccess"), nil)
}

// @Summary      Get online clients
// @Description  Returns emails of currently connected clients (last seen within heartbeat window).
// @Tags         Inbounds
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/onlines [post]
func (a *InboundController) onlines(c *gin.Context) {
	jsonObj(c, a.inboundService.GetOnlineClients(), nil)
}

// @Summary      Get last online timestamps
// @Description  Returns a map of client email to last-seen unix timestamp.
// @Tags         Inbounds
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/lastOnline [post]
func (a *InboundController) lastOnline(c *gin.Context) {
	data, err := a.inboundService.GetClientsLastOnline()
	jsonObj(c, data, err)
}

// @Summary      Update client traffic manually
// @Description  Manually adjusts a client's upload and download counters. Useful for migrations from external accounting systems.
// @Tags         Inbounds
// @Accept       json
// @Produce      json
// @Param        email path string true "Client email"
// @Param        body body object{upload=int64,download=int64} true "Traffic values in bytes"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/updateClientTraffic/{email} [post]
func (a *InboundController) updateClientTraffic(c *gin.Context) {
	email := c.Param("email")

	// Define the request structure for traffic update
	type TrafficUpdateRequest struct {
		Upload   int64 `json:"upload"`
		Download int64 `json:"download"`
	}

	var request TrafficUpdateRequest
	err := c.ShouldBindJSON(&request)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundUpdateSuccess"), err)
		return
	}

	err = a.inboundService.UpdateClientTrafficByEmail(email, request.Upload, request.Download)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}

	jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.inboundClientUpdateSuccess"), nil)
}

// @Summary      Delete client by email
// @Description  Removes a client from an inbound using email instead of UUID.
// @Tags         Inbounds
// @Produce      json
// @Param        id path int true "Inbound ID"
// @Param        email path string true "Client email"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/inbounds/{id}/delClientByEmail/{email} [post]
func (a *InboundController) delInboundClientByEmail(c *gin.Context) {
	inboundId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "Invalid inbound ID", err)
		return
	}

	email := c.Param("email")
	needRestart, err := a.inboundService.DelInboundClientByEmail(inboundId, email)
	if err != nil {
		jsonMsg(c, "Failed to delete client by email", err)
		return
	}

	jsonMsg(c, "Client deleted successfully", nil)
	if needRestart {
		a.xrayService.SetToNeedRestart()
	}
}

// resolveHost mirrors what sub.SubService.ResolveRequest does for the host
// field: prefers X-Forwarded-Host (first entry of any list, port stripped),
// then X-Real-IP, then the host portion of c.Request.Host. Keeping it in the
// controller layer means the service interface stays HTTP-agnostic — service
// methods receive a plain host string instead of a *gin.Context.
func resolveHost(c *gin.Context) string {
	if h := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); h != "" {
		if i := strings.Index(h, ","); i >= 0 {
			h = strings.TrimSpace(h[:i])
		}
		if hp, _, err := net.SplitHostPort(h); err == nil {
			return hp
		}
		return h
	}
	if h := c.GetHeader("X-Real-IP"); h != "" {
		return h
	}
	if h, _, err := net.SplitHostPort(c.Request.Host); err == nil {
		return h
	}
	return c.Request.Host
}

// @Summary      Get subscription links
// @Description  Returns every protocol URL (vless://, vmess://, etc.) for clients matching the subscription ID. Same data as /sub/{subId} but as JSON array.
// @Tags         Inbounds
// @Produce      json
// @Param        subId path string true "Subscription ID from client's subId field"
// @Success      200 {object} entity.Msg{obj=[]string}
// @Security     BearerAuth
// @Router       /panel/api/inbounds/getSubLinks/{subId} [get]
func (a *InboundController) getSubLinks(c *gin.Context) {
	links, err := a.inboundService.GetSubLinks(resolveHost(c), c.Param("subId"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.obtain"), err)
		return
	}
	jsonObj(c, links, nil)
}

// @Summary      Get client links
// @Description  Returns the URL(s) for one client on one inbound (same as the Copy URL button in the UI). Supports vmess, vless, trojan, shadowsocks, hysteria, hysteria2.
// @Tags         Inbounds
// @Produce      json
// @Param        id path int true "Inbound ID"
// @Param        email path string true "Client email"
// @Success      200 {object} entity.Msg{obj=[]string}
// @Security     BearerAuth
// @Router       /panel/api/inbounds/getClientLinks/{id}/{email} [get]
func (a *InboundController) getClientLinks(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "get"), err)
		return
	}
	links, err := a.inboundService.GetClientLinks(resolveHost(c), id, c.Param("email"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.inbounds.toasts.obtain"), err)
		return
	}
	jsonObj(c, links, nil)
}
