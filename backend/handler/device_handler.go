package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
)

// CapturePhoto 發送拍照指令 {"cmd":"capture"} 的 JSON 到指定的 MQTT Topic (使用閉包注入 Dependency)
func CapturePhoto(mqttClient mqtt.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 取得網址路徑上的裝置 ID，例如 api/devices/esp32-s3-01/capture
		deviceID := c.Param("id")

		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "未提供裝置 ID",
			})
			return
		}

		// 建構要發送的主題與訊息本體
		// 參考 ESP32 內的設定： PROJECT_PREFIX DEVICE_ID "/cmd"
		publishTopic := fmt.Sprintf("aiot-planter/v1/devices/%s/cmd", deviceID)
		
		payload := map[string]string{
			"cmd": "capture",
		}

		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "JSON 編碼失敗",
			})
			return
		}

		// 執行推播 (QoS 0，不保留 Retain)，最多等待 5 秒
		token := mqttClient.Publish(publishTopic, 0, false, jsonBytes)
		if !token.WaitTimeout(5 * time.Second) {
			log.Printf("⚠️ MQTT 推播逾時: %s\n", publishTopic)
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"success": false,
				"error":   "MQTT 推送逾時",
			})
			return
		}

		if token.Error() != nil {
			log.Printf("❌ 廣播拍照指令失敗: %v\n", token.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "MQTT 推送失敗",
			})
			return
		}

		log.Printf("📷 已成功向 %s 送出拍照指令！\n", publishTopic)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "已成功送出拍照指令，請稍候硬體執行上傳。",
		})
	}
}
