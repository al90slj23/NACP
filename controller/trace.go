package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// GetTraceList 处理 GET /api/log/traces
// 解析分页和筛选参数，调用 service 层获取链路摘要列表
func GetTraceList(c *gin.Context) {
	// Parse page parameter, default 1
	page, _ := strconv.Atoi(c.Query("p"))
	if page < 1 {
		page = 1
	}

	// Parse page_size parameter, default 20, range 1-100
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Parse optional filter parameters
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	modelName := c.Query("model_name")
	username := c.Query("username")
	tokenName := c.Query("token_name")
	status := c.Query("status")

	// Validate status parameter if provided
	if status != "" && status != "success" && status != "failed" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "status 参数无效，仅支持 success 或 failed",
		})
		return
	}

	params := service.TraceListParams{
		Page:           page,
		PageSize:       pageSize,
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		ModelName:      modelName,
		Username:       username,
		TokenName:      tokenName,
		Status:         status,
	}

	items, total, err := service.GetTraceList(params)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "查询失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"page":      page,
			"page_size": pageSize,
			"total":     total,
			"items":     items,
		},
	})
}

// GetTraceDetail 处理 GET /api/log/trace
// 解析 request_id 参数，调用 service 层获取链路详情
func GetTraceDetail(c *gin.Context) {
	requestId := c.Query("request_id")

	// Validate request_id: must be non-empty and max 64 characters
	if requestId == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "request_id 参数不能为空",
		})
		return
	}
	if len(requestId) > 64 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "request_id 参数长度不能超过 64 字符",
		})
		return
	}

	detail, err := service.GetTraceDetail(requestId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "查询失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    detail,
	})
}
