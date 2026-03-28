package handler

import (
	"errors"
	"net/http"
	"strconv"

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
// GET /api/sensors/history?device_id=X&limit=50
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

	var sensors []model.SensorRecord
	result := database.DB.
		Where("device_id = ?", deviceID).
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
