package router

import (
	"aiot-planter/handler"
	"aiot-planter/middleware"
	"net/http"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
)

// corsMiddleware 允許前端跨來源請求
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Setup 負責綁定應用程式的所有路由規則
func Setup(r *gin.Engine, mqttClient mqtt.Client) {
	r.Use(corsMiddleware())

	// 開放上傳目錄靜態讀取存取權限，例如 GET /uploads/latest.jpg
	r.Static("/uploads", "./uploads")

	// 所有 API 的根群組
	api := r.Group("/api")
	{
		// 公開路由（不需要 JWT）
		api.POST("/login", handler.LoginUser)
		api.POST("/user", handler.CreateUser)

		// 5. IoT 照片上傳端點 (供 ESP32 POST raw binary，不需要 JWT)
		api.POST("/upload", handler.UploadPhoto)

		// 6. Email 通知端點（不需要 JWT，由呼叫方帶入 Gmail 憑證）
		api.POST("/notify/send", handler.SendNotification)

		// 7. ML 心情寫入端點（供內部 Python 服務呼叫，不需要 JWT）
		api.POST("/mood", handler.PostMood(mqttClient))

		// 8. 感測器讀取（裝置資料，非使用者私密資料，內部服務與前端皆可存取）
		sensors := api.Group("/sensors")
		{
			sensors.GET("/latest", handler.GetLatestSensor)
			sensors.GET("/history", handler.GetSensorHistory)
		}

		// 需要 JWT 驗證的路由（使用者身分相關）
		auth := api.Group("")
		auth.Use(middleware.AuthRequired())
		{
			// 1. 心情查詢端點（前端讀取用，需要 JWT）
			moods := auth.Group("/mood")
			{
				moods.GET("/latest", handler.GetLatestMood)
				moods.GET("/history", handler.GetMoodHistory)
			}

			// 3. 裝置控制相關端點
			devices := auth.Group("/devices")
			{
				// /api/devices/esp32-s3-01/capture
				devices.POST("/:id/capture", handler.CapturePhoto(mqttClient))
			}

			// 4. 使用者相關端點
			users := auth.Group("/user")
			{
				users.GET("", handler.FindAllUsers)
			}
		}
	}
}
