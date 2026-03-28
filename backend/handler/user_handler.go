package handler

import (
	"aiot-planter/database"
	"aiot-planter/model"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// 註冊用 DTO (Data Transfer Object)
type RegisterInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required"`
}

// 登入用 DTO (Data Transfer Object)
type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func FindAllUsers(c *gin.Context) {
	var users []model.User

	r := database.DB.Find(&users)

	if r.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   r.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
	})
}

func CreateUser(c *gin.Context) {
	var input RegisterInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "無效的資料格式: " + err.Error(),
		})
		return
	}

	// 2. 使用 bcrypt 進行密碼雜湊，14 是一個不錯的安全強度 (數字越大越慢但越難被破解)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), 14)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "密碼加密失敗",
		})
		return
	}

	// 3. 把收到的文字跟「加密後的密碼」塞回真正的 User Model
	user := model.User{
		Username: input.Username,
		Email:    input.Email,
		Password: string(hashedPassword),
	}

	// 4. 存入資料庫
	r := database.DB.Create(&user)

	if r.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "無法儲存使用者 (可能是帳號或信箱已重複): " + r.Error.Error(),
		})
		return
	}

	// 5. 回傳建立成功的物件
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    user,
	})
}

func LoginUser(c *gin.Context) {
	var input LoginInput

	// 1. 確認前端傳來的 JSON 格式正不正確
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "無效的登入資料格式: " + err.Error(),
		})
		return
	}

	var user model.User

	// 2. 用帳號去資料庫尋找使用者
	if err := database.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "帳號或密碼錯誤", // 為防暴力破解，找不到帳號也要統一回覆「帳密錯誤」
		})
		return
	}

	// 3. 將資料庫的加密密碼與使用者輸入的明碼進行比對
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "帳號或密碼錯誤",
		})
		return
	}

	// 4. 準備產生 JWT 通關密語
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("❌ JWT_SECRET 未設定")
	}

	// 宣告 Token 裡面要包什麼資料 (Payload)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // 設定 24 小時後過期
	})

	// 5. 將 Payload 與金鑰簽名打包成字串
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "產生登入憑證失敗",
		})
		return
	}

	// 6. 登入成功！
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"token":        tokenString,
			"user":         user,
			"node_red_url": os.Getenv("NODE_RED_URL"),
		},
	})
}
