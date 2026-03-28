#include "connect_mqtt.h"
#include "env.h"
#include <PubSubClient.h>
#include <WiFi.h>
#include <Arduino.h>

extern PubSubClient client;

void reconnect_mqtt() {
  // 使用非阻塞設計：每 5 秒才嘗試重連一次 MQTT
  static unsigned long lastMqttRetry = 0;

  if (millis() - lastMqttRetry < 5000) {
    return;
  }

  lastMqttRetry = millis();

  Serial.printf("正在嘗試連線 MQTT [%s] ", MQTT_SERVER);

  if (client.connect(DEVICE_ID)) {
    Serial.println("\n[成功] MQTT 已連接！");
    client.subscribe(PROJECT_PREFIX DEVICE_ID "/mood");
    client.subscribe(PROJECT_PREFIX DEVICE_ID "/cmd");
  } else {
    Serial.printf("\n[失敗] 狀態碼%d\n", client.state());
  }
}
