package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"aiot-planter/database"
	"aiot-planter/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetLatestSensor 取得最新的感測器資料
func GetLatestSensor(c *gin.Context) {
	deviceID := c.Query("device_id")

	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "必須指定 device_id",
		})
		return
	}

	var sensor model.SensorRecord
	result := database.DB.Where("device_id = ?", deviceID).Order("created_at desc").First(&sensor)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "目前資料庫尚未有該裝置的資料",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "資料庫查詢失敗",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    sensor,
	})
}

// GetSensorHistory 取得感測器歷史資料（供 Dashboard 圖表使用）
// GET /api/sensors/history?device_id=X&limit=50&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z
func GetSensorHistory(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "必須指定 device_id",
		})
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	// 解析 start_time 和 end_time（RFC3339 格式）
	// 若兩者皆未提供，預設回傳最近 24 小時的資料
	var startTime, endTime time.Time
	startStr := c.Query("start_time")
	endStr := c.Query("end_time")

	if startStr == "" && endStr == "" {
		endTime = time.Now()
		startTime = endTime.Add(-24 * time.Hour)
	} else {
		if startStr != "" {
			parsed, err := time.Parse(time.RFC3339, startStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "start_time 格式錯誤，請使用 RFC3339 格式（例如 2024-01-01T00:00:00Z）",
				})
				return
			}
			startTime = parsed
		}
		if endStr != "" {
			parsed, err := time.Parse(time.RFC3339, endStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "end_time 格式錯誤，請使用 RFC3339 格式（例如 2024-01-02T00:00:00Z）",
				})
				return
			}
			endTime = parsed
		}
		if !startTime.IsZero() && !endTime.IsZero() && endTime.Before(startTime) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "end_time 不能早於 start_time",
			})
			return
		}
	}

	query := database.DB.Where("device_id = ?", deviceID)
	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}

	var sensors []model.SensorRecord
	result := query.
		Order("created_at desc").
		Limit(limit).
		Find(&sensors)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "資料庫查詢失敗",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    sensors,
	})
}
