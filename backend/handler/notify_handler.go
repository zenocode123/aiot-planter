package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"aiot-planter/database"
	"aiot-planter/model"

	"github.com/gin-gonic/gin"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
)

type NotifyInput struct {
	DeviceId    string `json:"device_id"    binding:"required"`
	Recipient   string `json:"recipient"    binding:"required,email"`
	Sender      string `json:"sender"       binding:"required,email"`
	AppPassword string `json:"app_password" binding:"required"`
}

// SendNotification 取得最新感測資料、心情，並寄送 HTML Email（附上最新照片）
func SendNotification(c *gin.Context) {
	var input NotifyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "請填寫完整的通知設定：" + err.Error(),
		})
		return
	}

	// 1. 從資料庫查詢最新感測資料
	var sensor model.SensorRecord
	if err := database.DB.
		Where("device_id = ?", input.DeviceId).
		Order("created_at desc").
		First(&sensor).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "尚無該裝置的感測資料",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "查詢感測資料失敗"})
		}
		return
	}

	// 2. 查詢最新心情（允許沒有）
	var mood model.MoodRecord
	hasMood := true
	if err := database.DB.
		Where("device_id = ?", input.DeviceId).
		Order("created_at desc").
		First(&mood).Error; err != nil {
		hasMood = false
	}

	// 3. 找最新照片（允許沒有）
	latestPhoto := findLatestPhoto("./uploads")

	// 4. 組 HTML 郵件內容
	body := buildEmailHTML(input.DeviceId, sensor, mood, hasMood)

	// 5. 建立郵件
	m := gomail.NewMessage()
	m.SetHeader("From", input.Sender)
	m.SetHeader("To", input.Recipient)
	m.SetHeader("Subject", fmt.Sprintf("🌿 AIoT Planter 植物狀態報告 – %s", time.Now().Format("2006/01/02 15:04")))
	m.SetBody("text/html", body)

	if latestPhoto != "" {
		m.Attach(latestPhoto, gomail.Rename(filepath.Base(latestPhoto)))
		log.Printf("📎 附加照片: %s\n", latestPhoto)
	}

	// 6. Gmail SMTP 寄送（port 587 STARTTLS）
	d := gomail.NewDialer("smtp.gmail.com", 587, input.Sender, input.AppPassword)

	if err := d.DialAndSend(m); err != nil {
		log.Printf("❌ Email 寄送失敗: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Email 寄送失敗，請確認 Gmail 帳號與應用程式密碼是否正確",
		})
		return
	}

	log.Printf("✅ Email 已寄送至 %s\n", input.Recipient)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("報告已寄送至 %s", input.Recipient),
	})
}

// findLatestPhoto 在 uploads 目錄中找最新的 .jpg 檔
func findLatestPhoto(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return ""
	}

	var jpgs []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".jpg") {
			jpgs = append(jpgs, e)
		}
	}
	if len(jpgs) == 0 {
		return ""
	}

	// 檔名格式 capture_YYYYMMDD_HHMMSS.jpg，字母排序即時間排序
	sort.Slice(jpgs, func(i, j int) bool {
		return jpgs[i].Name() > jpgs[j].Name()
	})

	return filepath.Join(dir, jpgs[0].Name())
}

// buildEmailHTML 產生 HTML 郵件本文
func buildEmailHTML(deviceId string, s model.SensorRecord, m model.MoodRecord, hasMood bool) string {
	moodRow := ""
	if hasMood {
		moodRow = fmt.Sprintf(`
		<tr>
			<td style="padding:8px 12px;color:#6b7280;">植物心情</td>
			<td style="padding:8px 12px;font-weight:600;">%s</td>
		</tr>`, m.Mood)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-TW">
<head><meta charset="UTF-8"></head>
<body style="font-family:sans-serif;background:#f9fafb;padding:24px;">
  <div style="max-width:520px;margin:0 auto;background:#fff;border-radius:12px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
    <div style="background:#1f2937;color:#fff;padding:20px 24px;">
      <h2 style="margin:0;font-size:1.2rem;">🌿 AIoT Planter 植物狀態報告</h2>
      <p style="margin:4px 0 0;font-size:0.85rem;color:#9ca3af;">裝置：%s ・ %s</p>
    </div>
    <div style="padding:20px 24px;">
      <table style="width:100%%;border-collapse:collapse;font-size:0.95rem;">
        <tr style="background:#f3f4f6;">
          <td style="padding:8px 12px;color:#6b7280;">溫度</td>
          <td style="padding:8px 12px;font-weight:600;">%.1f °C</td>
        </tr>
        <tr>
          <td style="padding:8px 12px;color:#6b7280;">濕度</td>
          <td style="padding:8px 12px;font-weight:600;">%.1f %%</td>
        </tr>
        <tr style="background:#f3f4f6;">
          <td style="padding:8px 12px;color:#6b7280;">土壤濕度</td>
          <td style="padding:8px 12px;font-weight:600;">%d %%</td>
        </tr>
        <tr>
          <td style="padding:8px 12px;color:#6b7280;">光照強度</td>
          <td style="padding:8px 12px;font-weight:600;">%.0f lux</td>
        </tr>
        %s
      </table>
    </div>
    <div style="padding:12px 24px 20px;font-size:0.8rem;color:#9ca3af;">
      如有附件照片，為最新一張植物快照。
    </div>
  </div>
</body>
</html>`,
		deviceId,
		s.CreatedAt.Format("2006/01/02 15:04"),
		s.Temperature,
		s.Humidity,
		s.SoilMoisture,
		s.LightLevel,
		moodRow,
	)
}
