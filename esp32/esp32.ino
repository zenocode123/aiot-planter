#include "env.h"
#include <WiFi.h>
#include "connect_wifi.h"
#include "connect_mqtt.h"
#include "camera_module.h"
#include "DHT.h"
#include "soil.h"
#include <BH1750.h>
#include <ArduinoJson.h>
#include <PubSubClient.h>
#include <Adafruit_SSD1306.h>
#include <FluxGarage_RoboEyes.h>

DHT dht_sensor(DHT_PIN, DHT_TYPE);
Soil soil_sensor(SOIL_PIN, DRY, WET);
BH1750 light_sensor;

WiFiClient esp_client;
PubSubClient client(esp_client);

Adafruit_SSD1306 display(SCREEN_WIDTH, SCREEN_HEIGHT, &Wire, OLED_RESET);
RoboEyes<Adafruit_SSD1306> roboEyes(display);

// 執行 MQTT 回調
void mqttCallback(char *topic, byte *payload, unsigned int length);

// Set CAM False
bool should_capture_photo = false;

// 馬達控制狀態
bool pump_is_on       = false;
unsigned long pump_start_ms = 0;

void setup() {
  Serial.begin(115200);
  // --- 初始化 WiFi ---
  init_wifi();
  // --- 初始化 MQTT ---
  client.setServer(MQTT_SERVER, MQTT_PORT);
  client.setCallback(mqttCallback);
  // --- 初始化 I2C ---
  Wire.begin(I2C2_SDA, I2C2_SCL);
  // --- 初始化 OLED 與 RoboEyes ---
  if (!display.begin(SSD1306_SWITCHCAPVCC, 0x3C)) {
    Serial.println(F("SSD1306 設定失敗！"));
  }
  // Max framerate 100fps
  roboEyes.begin(SCREEN_WIDTH, SCREEN_HEIGHT, 100);
  roboEyes.setPosition(DEFAULT);
  roboEyes.setAutoblinker(true);
  roboEyes.setIdleMode(true);
  roboEyes.close();
  // --- 初始化 DHT ---
  dht_sensor.begin();
  // --- 初始化 BH1750 ---
  light_sensor.begin(BH1750::CONTINUOUS_HIGH_RES_MODE, 0x23, &Wire);
  // --- 初始化繼電器（Active-LOW 模組：HIGH = 關閉）---
  pinMode(RELAY_PIN, OUTPUT);
  digitalWrite(RELAY_PIN, HIGH);
  // --- 初始化 CAM ---
  init_camera();
}

void loop() {
  // 保持動畫更新
  roboEyes.update();

  // 1. 檢查 WiFi Connect & Reconnect
  if (WiFi.status() != WL_CONNECTED) {
    static unsigned long lastWifiRetry = 0;
    // Each 10s Init Wifi
    if (millis() - lastWifiRetry > 10000 || lastWifiRetry == 0) {
      Serial.println("WiFi 斷開，嘗試重新連線...");
      init_wifi();
      lastWifiRetry = millis();
    }
    return;
  }

  // 2. 檢查 MQTT Connect
  if (!client.connected()) {
    reconnect_mqtt();
    return;
  }

  client.loop();

  // 馬達安全計時器：超過 PUMP_MAX_ON_SEC 自動關閉
  if (pump_is_on && (millis() - pump_start_ms > (unsigned long)PUMP_MAX_ON_SEC * 1000UL)) {
    digitalWrite(RELAY_PIN, HIGH);  // Active-LOW：HIGH = 關閉
    pump_is_on = false;
    Serial.println("⚠️ 馬達安全保護：超時自動關閉");
  }

  // CAM 觸發邏輯
  if (should_capture_photo) {
    // 1. reset
    should_capture_photo = false;  
    // 2. API URL http://172.20.10.5:3000/api/upload?device_id=esp32-s3-01
    String apiUrl = String("http://") + MQTT_SERVER + ":" + String(BACKEND_PORT) + "/api/upload?device_id=" + DEVICE_ID;
    // 3. 執行拍照
    take_and_upload_photo(apiUrl);
  }

  // --- Non-Blocking 5m---
  constexpr unsigned long UPDATE_INTERVAL_MS = 5 * 60 * 1000UL;

  static unsigned long lastUpdate = 0;

  if (millis() - lastUpdate < UPDATE_INTERVAL_MS) {
    return;
  }
  lastUpdate = millis();

  float h = dht_sensor.readHumidity();
  float t = dht_sensor.readTemperature();
  int s = soil_sensor.readPercent();
  float l = light_sensor.readLightLevel();

  if (isnan(t) || isnan(h)) {
    Serial.println("DHT 讀取失敗！");
    return;
  }

  Serial.printf("大氣溫度：%.1f°C，大氣濕度：%.1f%%，土壤濕度：%d%%，光照度：%.1f lx", t, h, s, l);
  Serial.println("");

  JsonDocument doc;
  doc["device_id"] = DEVICE_ID;
  doc["temperature"] = serialized(String(t, 1));
  doc["humidity"] = serialized(String(h, 1));
  doc["soil_moisture"] = s;
  doc["light_level"] = serialized(String(l, 1));

  char jsonBuffer[200];
  serializeJson(doc, jsonBuffer);

  client.publish(MQTT_TOPIC, jsonBuffer);

  Serial.println(jsonBuffer);
}

void mqttCallback(char *topic, byte *payload, unsigned int length) {
  String message = "";

  for (int i = 0; i < length; i++) {
    message += (char)payload[i];
  }

  Serial.printf("收到 MQTT 訊息：%s\n", message.c_str());

  // 解析 JSON
  JsonDocument doc;
  DeserializationError error = deserializeJson(doc, message);

  if (!error && doc.containsKey("cmd")) {
    // 宣告並初始化
    String cmd = doc["cmd"].as<String>();
    if (cmd == "capture") {
      should_capture_photo = true;
      Serial.println("收到拍照指令，將於主迴圈中拍攝。");
      return;
    }
    if (cmd == "pump_on") {
      digitalWrite(RELAY_PIN, LOW);   // Active-LOW：LOW = 繼電器導通
      pump_is_on    = true;
      pump_start_ms = millis();
      Serial.println("💧 馬達啟動");
      return;
    }
    if (cmd == "pump_off") {
      digitalWrite(RELAY_PIN, HIGH);  // Active-LOW：HIGH = 繼電器斷開
      pump_is_on = false;
      Serial.println("🛑 馬達停止");
      return;
    }
  }

  if (!error && doc.containsKey("mood")) {
    // 如果是 JSON 格式且包含 mood 欄位，重新賦值。
    message = doc["mood"].as<String>();
    Serial.printf("解析出心情指令：%s\n", message.c_str());
  } else {
    // 如果解析失敗或沒有 mood/cmd，猜測可能是後端直接傳純文字，保留原 message 檢查
    message.trim();
  }

  // 重置
  roboEyes.setPosition(DEFAULT);
  roboEyes.open();

  // hot、sick、normal、happy、sad
  // 先給基底心情
  if (message == "happy") {
    roboEyes.setMood(HAPPY);
    roboEyes.anim_laugh();
  } else if (message == "sad") {
    roboEyes.setMood(TIRED);
    roboEyes.setPosition(S);
  } else if (message == "sick") {
    roboEyes.setMood(DEFAULT);
    roboEyes.anim_confused();
  } else if (message == "hot") {
    roboEyes.setMood(ANGRY);
    roboEyes.blink();
  } else {
    roboEyes.setMood(DEFAULT);
    roboEyes.blink();
  }
}
