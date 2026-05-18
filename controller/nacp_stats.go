package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetNacpStatsOverview(c *gin.Context) {
	rangeHours := parseNacpStatsRangeHours(c)
	overview, err := model.GetNacpStatsOverview(rangeHours)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, overview)
}

func GetNacpStatsModels(c *gin.Context) {
	page, pageSize := parseNacpStatsPage(c)
	result, err := model.GetNacpModelRankStats(parseNacpStatsRangeHours(c), page, pageSize, c.Query("sort"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetNacpStatsUsers(c *gin.Context) {
	page, pageSize := parseNacpStatsPage(c)
	result, err := model.GetNacpUserRankStats(parseNacpStatsRangeHours(c), page, pageSize, c.Query("sort"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetNacpStatsChannels(c *gin.Context) {
	page, pageSize := parseNacpStatsPage(c)
	result, err := model.GetNacpChannelRankStats(parseNacpStatsRangeHours(c), page, pageSize, c.Query("sort"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetNacpStatsDimensions(c *gin.Context) {
	page, pageSize := parseNacpStatsPage(c)
	result, err := model.GetNacpDimensionRankStats(parseNacpStatsRangeHours(c), c.Query("dimension"), page, pageSize, c.Query("sort"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetNacpStatsModelCoverage(c *gin.Context) {
	page, pageSize := parseNacpStatsPage(c)
	result, err := model.GetNacpModelCoverageStats(parseNacpStatsRangeHours(c), page, pageSize, c.Query("sort"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetNacpStatsCosts(c *gin.Context) {
	page, pageSize := parseNacpStatsPage(c)
	result, err := model.GetNacpCostRankStats(c.Query("dimension"), page, pageSize, c.Query("sort"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func parseNacpStatsRangeHours(c *gin.Context) int {
	rangeHours := 24
	if hours := c.Query("hours"); hours != "" {
		parsed, err := strconv.Atoi(hours)
		if err == nil && parsed > 0 {
			rangeHours = parsed
		}
	}
	if rangeHours > 24*30 {
		rangeHours = 24 * 30
	}
	return rangeHours
}

func parseNacpStatsPage(c *gin.Context) (int, int) {
	page := 1
	pageSize := 10
	if raw := c.Query("p"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err == nil && parsed > 0 {
			page = parsed
		}
	}
	if raw := c.Query("page_size"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err == nil && parsed > 0 {
			pageSize = parsed
		}
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
