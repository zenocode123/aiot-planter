package broker

import (
	"encoding/json"
	"log"
	"os"

	"aiot-planter/database"
	"aiot-planter/model"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTT Setup
func SetupMQTT() mqtt.Client {
	mqttBroker := os.Getenv("MQTT_BROKER")
	mqttClientID := os.Getenv("MQTT_CLIENT_ID")
	mqttTopic := os.Getenv("MQTT_TOPIC")

	if mqttBroker == "" {
		log.Fatal("❌ MQTT_BROKER 未設定")
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttBroker)
	opts.SetClientID(mqttClientID)

	// 設定全域的訊息接收 Handler
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("📩 收到 Topic: %s 的 MQTT 訊息", msg.Topic())
		var record model.SensorRecord

		// 解析 JSON
		if err := json.Unmarshal(msg.Payload(), &record); err != nil {
			log.Printf("❌ JSON 解析失敗: %v\n", err)
			return
		}

		// 儲存至資料庫
		if err := database.DB.Create(&record).Error; err != nil {
			log.Printf("❌ 寫入資料庫失敗: %v\n", err)
			return
		}

		log.Printf("✅ 成功儲存感測器 %s 的數據", record.DeviceId)
	})

	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("❌ 連接 MQTT Broker 失敗: %v", token.Error())
	}

	log.Printf("✅ 成功連接到 MQTT Broker (%s)\n", mqttBroker)

	// 訂閱設定的主題
	if token := mqttClient.Subscribe(mqttTopic, 0, nil); token.Wait() && token.Error() != nil {
		log.Fatalf("❌ 訂閱主題 %s 失敗: %v", mqttTopic, token.Error())
	}

	log.Printf("✉️ 正在監聽主題: %s\n", mqttTopic)

	return mqttClient
}
