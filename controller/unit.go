package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/unitdetect"
	"github.com/gin-gonic/gin"
)

func GetAllUnits(c *gin.Context) {
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

	units, total, err := model.SearchUnits(
		strings.TrimSpace(c.Query("keyword")),
		strings.TrimSpace(c.Query("type")),
		status,
		pageInfo.GetStartIdx(),
		pageInfo.GetPageSize(),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"items":     units,
		"total":     total,
		"page":      pageInfo.GetPage(),
		"page_size": pageInfo.GetPageSize(),
	})
}

func GetUnit(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	unit, err := model.GetUnitById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, unit)
}

func AddUnit(c *gin.Context) {
	var unit model.Unit
	if err := c.ShouldBindJSON(&unit); err != nil {
		common.ApiError(c, err)
		return
	}
	unit.Normalize()
	if unit.Name == "" {
		common.ApiErrorMsg(c, "单位名称不能为空")
		return
	}
	if err := unit.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	if unit.Accounts != nil {
		if err := model.SyncUnitAccounts(unit.Id, unit.Accounts); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	created, err := model.GetUnitById(unit.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, created)
}

func DetectUnitType(c *gin.Context) {
	var req struct {
		SiteURL string `json:"site_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, unitdetect.Detect(c.Request.Context(), req.SiteURL))
}

func UpdateUnit(c *gin.Context) {
	var unit model.Unit
	if err := c.ShouldBindJSON(&unit); err != nil {
		common.ApiError(c, err)
		return
	}
	if unit.Id == 0 {
		common.ApiErrorMsg(c, "缺少单位 ID")
		return
	}
	unit.Normalize()
	if unit.Name == "" {
		common.ApiErrorMsg(c, "单位名称不能为空")
		return
	}
	if err := unit.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	if unit.Accounts != nil {
		if err := model.SyncUnitAccounts(unit.Id, unit.Accounts); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	updated, err := model.GetUnitById(unit.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, updated)
}

func GetUnitAccounts(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	accounts, err := model.GetUnitAccounts(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, accounts)
}

func GetUnitAccountTokens(c *gin.Context) {
	unitID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	accountID, err := strconv.Atoi(c.Param("account_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	unit, err := model.GetUnitById(unitID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	account, err := model.GetUnitAccountById(accountID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if account.UnitID != unitID {
		common.ApiErrorMsg(c, "所属账号不属于该单位")
		return
	}
	tokens, err := service.FetchUnitPlatformTokens(c.Request.Context(), unit, account)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, tokens)
}

func GetUnitAccountTokenOptions(c *gin.Context) {
	unitID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	accountID, err := strconv.Atoi(c.Param("account_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	unit, err := model.GetUnitById(unitID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	account, err := model.GetUnitAccountById(accountID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if account.UnitID != unitID {
		common.ApiErrorMsg(c, "所属账号不属于该单位")
		return
	}
	options, err := service.FetchUnitPlatformTokenOptions(c.Request.Context(), unit, account)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, options)
}

func AddUnitAccountToken(c *gin.Context) {
	unitID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	accountID, err := strconv.Atoi(c.Param("account_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req service.CreateUnitPlatformTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	unit, err := model.GetUnitById(unitID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	account, err := model.GetUnitAccountById(accountID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if account.UnitID != unitID {
		common.ApiErrorMsg(c, "所属账号不属于该单位")
		return
	}
	token, err := service.CreateUnitPlatformToken(c.Request.Context(), unit, account, req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, token)
}

func DeleteUnit(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	unit := model.Unit{Id: id}
	if err := unit.Delete(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, true)
}
