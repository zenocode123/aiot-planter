package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// UploadPhoto 負責接收由 ESP32 ESP-EYE 相機 HTTP POST 原生傳來的 JPEG 影像位元組
// 透過 query param device_id 區分裝置，照片存至 uploads/{device_id}/
func UploadPhoto(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "必須指定 device_id",
		})
		return
	}

	// 限制上傳大小為 10 MB，防止 OOM
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)

	imgData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "無法讀取圖片資料",
		})
		return
	}

	if len(imgData) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "請求為空，沒有圖片資料",
		})
		return
	}

	// 確保 uploads/{device_id}/ 目錄存在
	uploadDir := fmt.Sprintf("./uploads/%s", deviceID)
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		log.Printf("無法建立上傳資料夾: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "伺服器資料夾建立錯誤",
		})
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("capture_%s.jpg", timestamp)
	filePath := fmt.Sprintf("%s/%s", uploadDir, fileName)

	if err := os.WriteFile(filePath, imgData, 0644); err != nil {
		log.Printf("存檔失敗: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "伺服器檔案儲存錯誤",
		})
		return
	}

	log.Printf("📸 成功接收並儲存植物照片: %s (大小: %d bytes)\n", filePath, len(imgData))

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "上傳成功",
		"file_name": fileName,
		"file_url":  fmt.Sprintf("/uploads/%s/%s", deviceID, fileName),
	})
}
