package controller

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/govnoeby/67-Ui/v3/database/model"
	"github.com/govnoeby/67-Ui/v3/web/service"

	"github.com/gin-gonic/gin"
)

type NodeController struct {
	nodeService service.NodeService
}

func NewNodeController(g *gin.RouterGroup) *NodeController {
	a := &NodeController{}
	a.initRouter(g)
	return a
}

func (a *NodeController) initRouter(g *gin.RouterGroup) {
	g.GET("/list", a.list)
	g.GET("/get/:id", a.get)

	g.POST("/add", a.add)
	g.POST("/update/:id", a.update)
	g.POST("/del/:id", a.del)
	g.POST("/setEnable/:id", a.setEnable)

	g.POST("/test", a.test)
	g.POST("/probe/:id", a.probe)
	g.GET("/history/:id/:metric/:bucket", a.history)
}

// @Summary      List nodes
// @Description  Returns every configured node with connection details, health status, and last heartbeat info.
// @Tags         Nodes
// @Produce      json
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/nodes/list [get]
func (a *NodeController) list(c *gin.Context) {
	nodes, err := a.nodeService.GetAll()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.nodes.toasts.list"), err)
		return
	}
	jsonObj(c, nodes, nil)
}

// @Summary      Get node by ID
// @Description  Fetch a single node configuration by ID.
// @Tags         Nodes
// @Produce      json
// @Param        id path int true "Node ID"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/nodes/get/{id} [get]
func (a *NodeController) get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "get"), err)
		return
	}
	n, err := a.nodeService.GetById(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.nodes.toasts.obtain"), err)
		return
	}
	jsonObj(c, n, nil)
}

// @Summary      Add node
// @Description  Registers a new remote node. Provide URL, API token, and optional label.
// @Tags         Nodes
// @Accept       json
// @Produce      json
// @Param        body body model.Node true "Node configuration"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/nodes/add [post]
func (a *NodeController) add(c *gin.Context) {
	n := &model.Node{}
	if err := c.ShouldBind(n); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.nodes.toasts.add"), err)
		return
	}
	if err := a.nodeService.Create(n); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.nodes.toasts.add"), err)
		return
	}
	jsonMsgObj(c, I18nWeb(c, "pages.nodes.toasts.add"), n, nil)
}

// @Summary      Update node
// @Description  Replaces a node's connection details. Body shape matches /add.
// @Tags         Nodes
// @Accept       json
// @Produce      json
// @Param        id path int true "Node ID"
// @Param        body body model.Node true "Updated node configuration"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/nodes/update/{id} [post]
func (a *NodeController) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "get"), err)
		return
	}
	n := &model.Node{}
	if err := c.ShouldBind(n); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.nodes.toasts.update"), err)
		return
	}
	if err := a.nodeService.Update(id, n); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.nodes.toasts.update"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.nodes.toasts.update"), nil)
}

// @Summary      Delete node
// @Description  Deletes a node. Inbounds bound to it are not auto-migrated.
// @Tags         Nodes
// @Produce      json
// @Param        id path int true "Node ID"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/nodes/del/{id} [post]
func (a *NodeController) del(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "get"), err)
		return
	}
	if err := a.nodeService.Delete(id); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.nodes.toasts.delete"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.nodes.toasts.delete"), nil)
}

// @Summary      Toggle node enable
// @Description  Pauses or resumes traffic sync with a remote node.
// @Tags         Nodes
// @Accept       json
// @Produce      json
// @Param        id path int true "Node ID"
// @Param        body body object{enable=bool} true "Enable flag"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/nodes/setEnable/{id} [post]
func (a *NodeController) setEnable(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "get"), err)
		return
	}
	body := struct {
		Enable bool `json:"enable" form:"enable"`
	}{}
	if err := c.ShouldBind(&body); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.nodes.toasts.update"), err)
		return
	}
	if err := a.nodeService.SetEnable(id, body.Enable); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.nodes.toasts.update"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.nodes.toasts.update"), nil)
}

// @Summary      Test node connection
// @Description  Probes a node without saving it. Returns whether the handshake succeeds.
// @Tags         Nodes
// @Accept       json
// @Produce      json
// @Param        body body model.Node true "Node connection details to test"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/nodes/test [post]
func (a *NodeController) test(c *gin.Context) {
	n := &model.Node{}
	if err := c.ShouldBind(n); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.nodes.toasts.test"), err)
		return
	}
	if n.Scheme == "" {
		n.Scheme = "https"
	}
	if n.BasePath == "" {
		n.BasePath = "/"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 6*time.Second)
	defer cancel()
	patch, err := a.nodeService.Probe(ctx, n)
	jsonObj(c, patch.ToUI(err == nil), nil)
}

// @Summary      Probe existing node
// @Description  Probes an existing node and updates its cached health state and heartbeat.
// @Tags         Nodes
// @Produce      json
// @Param        id path int true "Node ID"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/nodes/probe/{id} [post]
func (a *NodeController) probe(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "get"), err)
		return
	}
	n, err := a.nodeService.GetById(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.nodes.toasts.obtain"), err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 6*time.Second)
	defer cancel()
	patch, probeErr := a.nodeService.Probe(ctx, n)
	if probeErr != nil {
		patch.Status = "offline"
	} else {
		patch.Status = "online"
	}
	_ = a.nodeService.UpdateHeartbeat(id, patch)
	jsonObj(c, patch.ToUI(probeErr == nil), nil)
}

// @Summary      Get node metric history
// @Description  Returns aggregated metric history for a node. Same shape as /server/history, scoped to one node.
// @Tags         Nodes
// @Produce      json
// @Param        id path int true "Node ID"
// @Param        metric path string true "Metric key (cpu, mem, netIn, netOut, etc.)"
// @Param        bucket path int true "Bucket size in seconds"
// @Success      200 {object} entity.Msg
// @Security     BearerAuth
// @Router       /panel/api/nodes/history/{id}/{metric}/{bucket} [get]
func (a *NodeController) history(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "get"), err)
		return
	}
	metric := c.Param("metric")
	if !slices.Contains(service.NodeMetricKeys, metric) {
		jsonMsg(c, "invalid metric", fmt.Errorf("unknown metric"))
		return
	}
	bucket, err := strconv.Atoi(c.Param("bucket"))
	if err != nil || bucket <= 0 || !allowedHistoryBuckets[bucket] {
		jsonMsg(c, "invalid bucket", fmt.Errorf("unsupported bucket"))
		return
	}
	jsonObj(c, a.nodeService.AggregateNodeMetric(id, metric, bucket, 60), nil)
}
