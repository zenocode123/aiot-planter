package main

import (
	"log"
	"os"

	"aiot-planter/broker"
	"aiot-planter/database"
	"aiot-planter/router" // 引入抽離的路由模組

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load Env
	err := godotenv.Load("../.env")

	if err != nil {
		log.Println("❌ 載入 .env 失敗")
	}

	// 2. Init DB
	database.InitDB()

	port := os.Getenv("PORT")

	// 3. 設定並連接 MQTT
	mqttClient := broker.SetupMQTT()
	defer mqttClient.Disconnect(250)

	// 4. 啟動 Gin Server
	r := gin.Default()

	router.Setup(r, mqttClient)

	r.Run(":" + port)
}
