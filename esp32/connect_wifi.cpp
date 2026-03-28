#include "connect_wifi.h"
#include "env.h"
#include <WiFi.h>
#include <Arduino.h>

void init_wifi() {
  int timeout = 0;

  WiFi.begin(SSID_NAME, SSID_PASSWORD);

  Serial.printf("WiFi：%s 連線中", SSID_NAME);

  while (WiFi.status() != WL_CONNECTED && timeout < 10) {
    delay(500);
    Serial.print(".");
    timeout++;
  }

  Serial.println("");

  if (WiFi.status() == WL_CONNECTED) {
    Serial.printf("WiFi：%s %s 已連線！", SSID_NAME, WiFi.localIP().toString().c_str());
  } else {
    Serial.printf("連線失敗!");
  }

  Serial.println("");
}
