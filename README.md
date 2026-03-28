# 🌱 AIoT Planter (智慧盆栽系統)

一個結合物聯網 (IoT)、邊緣硬體 (ESP32)、機器學習 (Machine Learning) 與現代化 Web 技術的智慧盆栽監控系統。
本專案不僅能從遠端全天候監控植物的生長環境 (溫濕度、光照、土壤濕度)、即時拍照，還能根據機器學習推論出「植物的心情」，並生動地顯示在實體的 OLED 螢幕上！

## ✨ 主要特色 (Key Features)

- **邊緣硬體感知與互動**：基於 ESP32S3-EYE，整合 DHT 溫濕度計、土壤濕度計、BH1750 光照計，並搭載 OLED 螢幕 (RoboEyes) 即時呈現豐富表情動畫。
- **IoT 雙向通訊機制**：使用 MQTT 協定，每 5 分鐘固定推播感測資料，並隨時監聽捕捉伺服器下達的指令 (例如：拍照、改變心情)。
- **即時影像擷取**：整合 ESP32 相機模組，收到 `capture` 指令時，能立即拍攝植物影像並直接透過 HTTP POST 上傳至後端伺服器。
- **機器學習推論**：(Python 端) 建立 Random Forest (隨機森林) 模型，分析多維度的植物環境數據並預測環境對於植物的健康壓力，轉化為「心情」。
- **高併發微服務 API**：使用 Go & Gin 搭建符合 Clean Architecture 規範的「前後端分離」高延展性 RESTful 伺服器。
- **無狀態安全認證**：採用 JWT (JSON Web Token) 配合 bcrypt 以保障連線終端與 Web 系統的使用者安全。

---

## 🛠️ 技術棧 (Tech Stack)

### IoT 硬體端 (ESP32 / C++)
- **核心開發板**: ESP32S3-EYE
- **感測器組件**: DHT11/22 (溫濕度)、Analog Soil Moisture (土壤水份)、BH1750 (I2C 光照)
- **多媒體組件**: OV2640 相機模組 (支援 JPEG 壓縮)、SSD1306 OLED
- **通訊與協定**: `WiFi.h`, `PubSubClient` (MQTT), `HTTPClient`, `ArduinoJson`
- **UI/動畫庫**: `Adafruit_SSD1306`, `FluxGarage_RoboEyes`

### 後端 API 伺服器 (Go)
- **架構設計**: 前後端分離 (Decoupled)，單一入口點，高內聚 Controller 設計
- **框架與 ORM**: Gin Web Framework, GORM
- **資料庫**: SQLite3
- **IoT 依賴**: `paho.mqtt.golang`

### 前端客戶端 (Web / HTML)
- **UI 框架**: Pico CSS (極簡優雅的 Classless CSS)
- **互動邏輯**: Alpine.js (直接在 HTML 中撰寫的輕量級資料綁定)

### 資料科學與機器學習 (Python)
- **語言**: Python 3.12+ (使用 `uv` 依賴管理)
- **核心模型**: `scikit-learn` (Random Forest Classifier)

---

## 🚀 系統架構與資料流 (Architecture & Data Flow)

### 1. 目錄結構

```text
├── backend/                  # Go 後端 API 核心
│   ├── broker/               # MQTT 初始化與連線封裝
│   ├── database/             # SQLite 連線與 ORM 模型遷移
│   ├── handler/              # 獨立的控制器邏輯 (User, Sensor, Mood)
│   ├── model/                # 資料庫 Schema 與 DTO 定義
│   ├── router/               # Gin API 路由統一管理器
│   └── main.go               # 輕量級應用程式啟動點
├── esp32/                    # IoT 邊緣裝置 C++ 程式碼
│   ├── env.h                 # (需建立) 硬體腳位配置與連線帳密
│   ├── camera_module/        # ESP32 鏡頭初始化與 HTTP 上傳邏輯
│   ├── connect_wifi/mqtt     # 連線保持與重新連線機制
│   └── esp32.ino             # 主迴圈 (Non-Blocking 輪詢、表情更新)
├── frontend/                 # 客戶端網頁
│   ├── login.html            # 支援 Token LocalStorage 儲存的登入介面
│   └── register.html         # 帳號註冊介面
└── script/                   # Python 資料收集與機器學習
```

### 2. 生態系資料流向 (End-to-End Flow)
1. **[感知]**：ESP32 持續讀取 (`DHT`, `Soil`, `BH1750`) 數據。
2. **[上拋]**：每 5 分鐘，ESP32 將這包感測據轉為 JSON，透過 MQTT (`aiot-planter/v1/devices/<id>/sensor`) 送出。
3. **[儲存]**：Go 伺服器的 Broker 收到 MQTT 推播，透過 GORM 立即寫入 SQLite，並開放提供給 `/api/sensors/latest` 查詢。
4. **[推論]**：Python 腳本拿著這些歷史數據送入 Random Forest，預測出心情 (例如 `"happy"`, `"sick"` , `"hot"`)，透過 HTTP POST 打向 Go 的 `/api/mood`。
5. **[回饋]**：Go 伺服器將心情轉送至 MQTT `aiot-planter/v1/devices/<id>/mood`。ESP32 收到後，利用 RoboEyes 庫即時轉變 OLED 螢幕上這盆植物的「眼神」。
6. **[互動]**：如果使用者想看植物現況，可透過 MQTT 發送 `{"cmd": "capture"}` 給 ESP32。ESP32 擷取即時相機畫面，利用 HTTP POST 實體圖片給 Go 的 `/api/upload`。

---

## ⚙️ 環境配置與啟動 (Getting Started)

### 第一部分：硬體設備 (ESP32) 燒錄設置

1. 請使用 Arduino IDE 或 PlatformIO 打開 `esp32/esp32.ino`。
2. 複製設定檔：在 `esp32/` 目錄中複製 `env.example` 並更名為 `env.h`。
3. 填寫 `env.h` 必填資訊：
    ```cpp
    #define SSID_NAME "您的_WiFi名稱"
    #define SSID_PASSWORD "您的_WiFi密碼"
    
    // 設定這盆植物獨一無二的 ID!
    #define DEVICE_ID "esp32-s3-01"  
    
    #define MQTT_SERVER "192.168.x.x" // 您跑 Go 伺服器或 Broker 的內網 IP
    #define BACKEND_PORT 3000         // Go 伺服器 API Port
    ```
4. 安裝依賴庫：`PubSubClient`, `ArduinoJson`, `DHT sensor library`, `BH1750`, `Adafruit SSD1306`, `FluxGarage_RoboEyes`。
5. 連接所有 Sensor 到 ESP32 上設定好的腳位，然後燒錄 (Upload)！

### 第二部分：伺服器啟動 (Go Backend)

在開始安裝前，請確保您的開發機或樹莓派已安裝 **Go 1.21+** 以及 **Mosquitto** 或任何 MQTT Broker。

```bash
# 1. 複製環境變數範本並填寫 (記得替換 MQTT_BROKER 與 JWT_SECRET)
cp .env.example .env

# 2. 抓取依賴並編譯
cd backend
go mod tidy
go build -o ../aiot-planter

# 3. 回到主目錄執行 API 系統
cd ..
./aiot-planter
```
> 若終端機印出 `✅ 成功連線並初始化資料庫` 以及 `✅ 成功連接到 MQTT Broker`，代表伺服器與硬體的橋樑已搭建完成。

### 第三部分：存取前端 Web
1. 打開瀏覽器，輸入 `http://<您的伺服器IP>:3000/frontend/login.html`。
2. 點擊「註冊」建立您的專屬帳號。
3. 登入成功後，系統將根據登入時 API 派發的 URL，自動引導您到 Node-RED Dashboard 等視覺化面板。

---

## 📡 API 與 MQTT 通訊手冊 

### MQTT 主題設計 (Topics)
- **上行資料 (ESP32 -> Server)**
  - `aiot-planter/v1/devices/+/sensor`: 定時傳送 `{ "temperature": "25.0", "humidity": "60.0", "soil_moisture": 45, "light_level": "300.0" }`
- **下行控制 (Server -> ESP32)**
  - `aiot-planter/v1/devices/<id>/mood`: 下達指令 (`happy`, `sad`, `sick`, `hot` 或 `{ "cmd": "capture" }`)

### Go HTTP REST API
- `POST /api/user`: 系統註冊。
- `POST /api/login`: 使用者登入，成功後核發 `JWT Token` 與前往跳轉的 `node_red_url` 資訊。
- `GET /api/sensors/latest?device_id=xxx`: 給前端呼叫，取得資料表中該盆栽最新鮮的一筆感測記錄。
- `POST /api/mood`: 接收 Python 的心情分析結果，轉派 MQTT 給 ESP32。

---

## 👨‍💻 Troubleshooting (常見問題排解)

**Q1: ESP32 的螢幕一直重複閃爍或重啟？**
**A**: 發生此情況通常是 Brownout，相機與 OLED 同時啟動時瞬間電流消耗太大。請確認您的供電 (USB 或電池) 足以提供穩定至少 5V/1A 的電流。

**Q2: ESP32 成功連線 Wifi 卻連不上 MQTT？**
**A**: 請檢查 `env.h` 中的 `MQTT_SERVER` IP 是否為伺服器的區網 IP (如 192.168.x.x)。如果是填寫 `localhost` 會指向 ESP32 自己。

**Q3: 註冊帳號時出現 "無效的資料格式"？**
**A**: 請開啟瀏覽器 `F12` > Network，檢查您送往後端的 JSON 格式是否確實包含了: `username`, `password`, `email`。

**Q4: 相機拍照失敗並顯示鏡頭獲取錯誤？**
**A**: OV2640 的線路接觸不良是很常發生的情況，請斷電後將鏡頭排線拔出重插一次。同時也請確認 ESP32S3-EYE 的腳位在 `env.h` 中是否沒有跟其他感測器衝突。
