package database

import (
	"log"
	"os"

	"aiot-planter/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dbName := os.Getenv("DB_NAME")

	if dbName == "" {
		log.Fatal("❌ DB_NAME 未設定")
	}

	var openErr error
	DB, openErr = gorm.Open(sqlite.Open(dbName+".db"), &gorm.Config{})

	if openErr != nil {
		log.Fatalf("❌ 資料庫連線失敗: %v", openErr)
	}

	if err := DB.AutoMigrate(&model.SensorRecord{}, &model.MoodRecord{}, &model.User{}); err != nil {
		log.Fatalf("❌ 資料表遷移失敗: %v", err)
	}

	log.Printf("✅ 成功連線並初始化資料庫 %s.db", dbName)
	log.Println("")
}
