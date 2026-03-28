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
func UploadPhoto(c *gin.Context) {
	// 讀取請求中的所有二進制數據
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

	// 確保 uploads 目錄存在
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		log.Printf("無法建立上傳資料夾: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "伺服器資料夾建立錯誤",
		})
		return
	}

	// 利用時間戳記建立唯一的檔名
	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("capture_%s.jpg", timestamp)
	filePath := fmt.Sprintf("%s/%s", uploadDir, fileName)

	// 將二進制資料寫入本機檔案
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
		"file_url":  fmt.Sprintf("/uploads/%s", fileName),
	})
}
