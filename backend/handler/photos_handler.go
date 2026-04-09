package handler

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type PhotoItem struct {
	FileName string    `json:"file_name"`
	URL      string    `json:"url"`
	TakenAt  time.Time `json:"taken_at"`
}

// GetPhotos 列出指定裝置的照片清單（從檔案系統讀取，不需要 DB）
// GET /api/photos?device_id=X&limit=30
func GetPhotos(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "必須指定 device_id",
		})
		return
	}

	limit := 30
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	dir := fmt.Sprintf("./uploads/%s", deviceID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, gin.H{"success": true, "data": []PhotoItem{}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "無法讀取照片目錄",
		})
		return
	}

	var photos []PhotoItem
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".jpg") {
			continue
		}
		takenAt := parsePhotoTime(e.Name())
		photos = append(photos, PhotoItem{
			FileName: e.Name(),
			URL:      fmt.Sprintf("/uploads/%s/%s", deviceID, e.Name()),
			TakenAt:  takenAt,
		})
	}

	// 依時間降冪排序（最新的在前）
	sort.Slice(photos, func(i, j int) bool {
		return photos[i].TakenAt.After(photos[j].TakenAt)
	})

	if len(photos) > limit {
		photos = photos[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    photos,
	})
}

// parsePhotoTime 從檔名 capture_YYYYMMDD_HHMMSS.jpg 解析時間
func parsePhotoTime(filename string) time.Time {
	name := strings.TrimSuffix(filename, ".jpg")
	parts := strings.Split(name, "_")
	// 預期格式: ["capture", "20240101", "120000"]
	if len(parts) == 3 {
		t, err := time.ParseInLocation("20060102_150405", parts[1]+"_"+parts[2], time.Local)
		if err == nil {
			return t
		}
	}
	// 解析失敗時用檔案名稱字串排序仍正確，回傳零值
	return time.Time{}
}
