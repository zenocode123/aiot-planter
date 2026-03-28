package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"aiot-planter/database"
	"aiot-planter/model"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func PostMood(mqttClient mqtt.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.MoodRecord
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "JSON 格式錯誤",
			})
			return
		}

		if err := database.DB.Create(&req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "儲存心情失敗",
			})
			return
		}

		log.Printf("🤖 機器學習回傳了剛算好的心情 ➡️ %s\n", req.Mood)

		// 透過注入的 mqttClient 將心情推送回給對應設備
		publishTopic := "aiot-planter/v1/devices/" + req.DeviceId + "/mood"
		token := mqttClient.Publish(publishTopic, 0, false, req.Mood)
		if !token.WaitTimeout(5 * time.Second) {
			log.Printf("⚠️ MQTT 推播逾時: %s\n", publishTopic)
		} else if token.Error() != nil {
			log.Printf("❌ MQTT 推播失敗: %v\n", token.Error())
		} else {
			log.Printf("🚀 已將心情 (%s) 推播至 MQTT: %s\n", req.Mood, publishTopic)
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"message": "心情已成功記錄並推播至盆栽！",
				"mood":    req.Mood,
			},
		})
	}
}

// GetLatestMood 取得裝置最新心情
// GET /api/mood/latest?device_id=X
func GetLatestMood(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "必須指定 device_id",
		})
		return
	}

	var mood model.MoodRecord
	result := database.DB.Where("device_id = ?", deviceID).Order("created_at desc").First(&mood)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "尚無心情紀錄",
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
		"data":    mood,
	})
}

// GetMoodHistory 取得心情歷史紀錄（供 Dashboard 圖表使用）
// GET /api/mood/history?device_id=X&limit=50
func GetMoodHistory(c *gin.Context) {
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

	var moods []model.MoodRecord
	result := database.DB.
		Where("device_id = ?", deviceID).
		Order("created_at desc").
		Limit(limit).
		Find(&moods)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "資料庫查詢失敗",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    moods,
	})
}
