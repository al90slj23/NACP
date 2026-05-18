package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetAllUnitMonitors(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	status := 0
	if statusParam := c.Query("status"); statusParam != "" && statusParam != "all" {
		parsed, err := strconv.Atoi(statusParam)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		status = parsed
	}
	unitID := 0
	if unitIDParam := c.Query("unit_id"); unitIDParam != "" {
		parsed, err := strconv.Atoi(unitIDParam)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		unitID = parsed
	}
	unitAccountID := 0
	if unitAccountIDParam := c.Query("unit_account_id"); unitAccountIDParam != "" {
		parsed, err := strconv.Atoi(unitAccountIDParam)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		unitAccountID = parsed
	}
	monitors, total, err := model.SearchUnitAccountMonitors(
		strings.TrimSpace(c.Query("keyword")),
		status,
		unitID,
		unitAccountID,
		pageInfo.GetStartIdx(),
		pageInfo.GetPageSize(),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"items":     monitors,
		"total":     total,
		"page":      pageInfo.GetPage(),
		"page_size": pageInfo.GetPageSize(),
	})
}

func AddUnitMonitor(c *gin.Context) {
	var req struct {
		UnitID        int    `json:"unit_id"`
		UnitAccountID int    `json:"unit_account_id"`
		Name          string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.UnitID <= 0 || req.UnitAccountID <= 0 {
		common.ApiErrorMsg(c, "请选择单位和账号")
		return
	}
	unit, err := model.GetUnitById(req.UnitID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	account, err := model.GetUnitAccountById(req.UnitAccountID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if account.UnitID != req.UnitID {
		common.ApiErrorMsg(c, "所属账号不属于该单位")
		return
	}
	if existing, err := model.GetUnitAccountMonitorByAccount(req.UnitID, req.UnitAccountID); err == nil && existing.Id > 0 {
		common.ApiSuccess(c, existing)
		return
	} else if err != nil && !model.IsUnitAccountMonitorNotFound(err) {
		common.ApiError(c, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.TrimSpace(unit.Name + " / " + account.Account)
		if name == "" {
			name = unit.Name
		}
	}
	monitor := model.UnitAccountMonitor{
		UnitID:         req.UnitID,
		UnitAccountID:  req.UnitAccountID,
		Name:           name,
		Status:         model.UnitMonitorStatusEnabled,
		BalanceUnit:    "USD",
		PlatformStatus: "pending",
	}
	if err := monitor.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	created, err := model.GetUnitAccountMonitorById(monitor.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, created)
}

func UpdateUnitMonitor(c *gin.Context) {
	var monitor model.UnitAccountMonitor
	if err := c.ShouldBindJSON(&monitor); err != nil {
		common.ApiError(c, err)
		return
	}
	if monitor.Id <= 0 {
		common.ApiErrorMsg(c, "缺少监控 ID")
		return
	}
	if err := monitor.UpdateBasic(); err != nil {
		common.ApiError(c, err)
		return
	}
	updated, err := model.GetUnitAccountMonitorById(monitor.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, updated)
}

func CheckUnitMonitor(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	monitor, err := model.GetUnitAccountMonitorById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	snapshot, err := service.CheckUnitAccountMonitor(c.Request.Context(), monitor)
	if err != nil {
		_ = service.ApplyUnitMonitorError(monitor, err)
		common.ApiError(c, err)
		return
	}
	if err := service.ApplyUnitMonitorSnapshot(monitor, snapshot); err != nil {
		common.ApiError(c, err)
		return
	}
	updated, err := model.GetUnitAccountMonitorById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, updated)
}

func CheckAllUnitMonitors(c *gin.Context) {
	monitors, err := model.ListEnabledUnitAccountMonitors()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	okCount := 0
	failCount := 0
	for _, monitor := range monitors {
		snapshot, err := service.CheckUnitAccountMonitor(c.Request.Context(), monitor)
		if err != nil {
			failCount++
			_ = service.ApplyUnitMonitorError(monitor, err)
			continue
		}
		if err := service.ApplyUnitMonitorSnapshot(monitor, snapshot); err != nil {
			failCount++
			continue
		}
		okCount++
	}
	common.ApiSuccess(c, gin.H{
		"total":  len(monitors),
		"ok":     okCount,
		"failed": failCount,
	})
}

func DeleteUnitMonitor(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	monitor := model.UnitAccountMonitor{Id: id}
	if err := monitor.Delete(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, true)
}
