# AIoT Planter — 智慧盆栽監控系統

一個結合 **IoT 邊緣硬體、Go 後端 API、Python 機器學習與現代 Web 前端**的全端智慧盆栽系統。
ESP32 感測器每 5 分鐘上報環境數據，Python 以 Random Forest 模型推論「植物心情」，並透過 MQTT 即時回饋至 ESP32 的 OLED 表情動畫；Web 儀表板提供即時圖表、遠端拍照與生長紀錄時間軸。

## 目錄

- [系統架構](#系統架構)
- [技術棧](#技術棧)
- [快速開始](#快速開始)
  - [環境需求](#環境需求)
  - [環境變數設定](#環境變數設定)
  - [啟動 MQTT Broker](#啟動-mqtt-broker)
  - [啟動 Go 後端](#啟動-go-後端)
  - [啟動 Python 推論引擎](#啟動-python-推論引擎)
  - [ESP32 韌體燒錄](#esp32-韌體燒錄)
  - [存取前端](#存取前端)
- [API 參考](#api-參考)
- [MQTT 主題](#mqtt-主題)
- [目錄結構](#目錄結構)
- [Troubleshooting](#troubleshooting)

---

## 系統架構

```
┌─────────────────────────────────────────────────────────────────┐
│                          User Browser                           │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Frontend (HTML + Tailwind CSS + Alpine.js + ApexCharts) │   │
│  │  login.html / register.html / dashboard.html             │   │
│  └──────────────────┬──────────────────────────────────────┘   │
└─────────────────────│───────────────────────────────────────────┘
                      │ HTTP REST (JWT)
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│              Go Backend API  (Gin + GORM)  :3000                │
│                                                                 │
│  ┌──────────────┐  ┌────────────────┐  ┌───────────────────┐  │
│  │  REST Router │  │   Handlers     │  │   MQTT Broker     │  │
│  │  /api/...    │  │  sensor/mood/  │  │  (paho.mqtt.go)   │  │
│  │  JWT Auth    │  │  photos/notify │  │  Subscribe/Publish│  │
│  └──────┬───────┘  └───────┬────────┘  └────────┬──────────┘  │
│         │                  │                     │              │
│         └──────────────────▼─────────────────────┘             │
│                     ┌────────────┐                              │
│                     │  SQLite DB │                              │
│                     │ (GORM ORM) │                              │
│                     └────────────┘                              │
└───────────────────────────────────────────────────────────────-─┘
         ▲  MQTT (sensor data)          │  MQTT (mood / cmd)
         │  HTTP POST (photo upload)    │
         │                             ▼
┌────────┴────────────────────────────────────────────────────────┐
│                   ESP32-S3 Edge Device                          │
│                                                                 │
│  DHT11 ──► Temperature / Humidity                               │
│  Soil  ──► Soil Moisture %         ──► JSON ──► MQTT publish   │
│  BH1750──► Light Level (lux)                  every 5 minutes  │
│                                                                 │
│  OV2640──► JPEG capture ──► HTTP POST /api/upload              │
│  SSD1306 + RoboEyes ◄── MQTT subscribe (mood / cmd)            │
└─────────────────────────────────────────────────────────────────┘
         │  HTTP GET /api/sensors/latest
         ▼
┌─────────────────────────────────────────────────────────────────┐
│              Python ML Inference Engine                         │
│                                                                 │
│  schedule (every 5min)                                          │
│    ├── GET /api/sensors/latest                                  │
│    ├── Random Forest predict(temp, humidity, soil, light)       │
│    └── POST /api/mood  { mood: "happy" | "sad" | "sick" | ... } │
└─────────────────────────────────────────────────────────────────┘
```

### 端對端資料流

1. **感知** — ESP32 每 5 分鐘讀取 DHT11 / BH1750 / 土壤感測器
2. **上行** — 感測 JSON 透過 MQTT `aiot-planter/v1/devices/<id>/sensor` 送至 Broker
3. **儲存** — Go Backend 收到 MQTT，透過 GORM 寫入 SQLite
4. **推論** — Python 每 5 分鐘 GET 最新感測資料，餵給 Random Forest 模型，預測心情
5. **下行** — Python POST `/api/mood` → Go 透過 MQTT `aiot-planter/v1/devices/<id>/mood` 通知 ESP32
6. **表情** — ESP32 收到心情後，RoboEyes 在 OLED 呈現對應表情動畫
7. **拍照** — 前端按下「拍照」→ POST `/api/devices/<id>/capture` → Go 發送 MQTT `{"cmd":"capture"}` → ESP32 拍照並 HTTP POST 至 `/api/upload?device_id=<id>`
8. **通知** — 前端按下「寄送報告」→ POST `/api/notify/send` → Go 附上最新照片，Gmail SMTP 寄送 HTML 郵件

---

## 技術棧

| 層 | 技術 |
|---|---|
| **IoT 硬體** | ESP32-S3 (Arduino C++)、DHT11、BH1750 (I²C)、Soil ADC、OV2640 相機、SSD1306 OLED |
| **IoT 韌體庫** | PubSubClient (MQTT)、ArduinoJson、HTTPClient、Adafruit_SSD1306、FluxGarage_RoboEyes |
| **後端** | Go 1.21+、Gin、GORM、SQLite3、paho.mqtt.golang、golang-jwt、bcrypt、gomail |
| **機器學習** | Python 3.11+、scikit-learn (Random Forest Classifier)、pandas、joblib、schedule、pydantic |
| **前端** | Tailwind CSS (CDN)、Alpine.js v3、ApexCharts、DM Sans + DM Mono 字體 |
| **通訊協定** | MQTT (Mosquitto)、HTTP REST、I²C、WiFi |
| **資料庫** | SQLite (開發/生產皆適用) |

---

## 快速開始

### 環境需求

| 工具 | 最低版本 | 用途 |
|---|---|---|
| Go | 1.21+ | 後端 API |
| Python | 3.11+ | ML 推論引擎 |
| uv | 最新 | Python 套件管理 |
| Mosquitto | 2.x | MQTT Broker |
| Arduino IDE | 2.x | ESP32 韌體燒錄 |

```bash
# 安裝 Mosquitto（Raspberry Pi / Ubuntu）
sudo apt-get install mosquitto mosquitto-clients

# 安裝 uv（Python 套件管理器）
curl -LsSf https://astral.sh/uv/install.sh | sh
```

---

### 環境變數設定

在專案根目錄複製並填寫 `.env`：

```bash
cp .env.example .env
```

`.env` 完整範例：

```env
# 後端 Port
PORT=3000

# SQLite 資料庫名稱（不含 .db）
DB_NAME=sq_db

# MQTT Broker 連線資訊
MQTT_BROKER=tcp://localhost:1883
MQTT_CLIENT_ID=gin_client
MQTT_TOPIC=aiot-planter/v1/devices/+/sensor

# Python ML 推論目標裝置
TARGET_DEVICE_ID=esp32-s3-01

# JWT 簽章密鑰（請使用長且隨機的字串）
JWT_SECRET=your-secret-key-here
```

生成安全的 JWT_SECRET：
```bash
openssl rand -base64 32
```

---

### 啟動 MQTT Broker

```bash
# 啟動 Mosquitto（前景）
mosquitto

# 或以 systemd 背景服務執行
sudo systemctl start mosquitto
sudo systemctl enable mosquitto

# 驗證 Broker 正在運作
mosquitto_sub -t "aiot-planter/#" -v
```

---

### 啟動 Go 後端

```bash
cd backend

# 安裝依賴
go mod tidy

# 直接執行（開發模式）
go run main.go

# 或編譯後執行（生產建議）
go build -o ../aiot-planter .
cd ..
./aiot-planter
```

成功啟動時終端機輸出：
```
✅ 成功連線並初始化資料庫
✅ 成功連接到 MQTT Broker (tcp://localhost:1883)
✉️ 正在監聽主題: aiot-planter/v1/devices/+/sensor
[GIN-debug] Listening and serving HTTP on :3000
```

靜態檔案伺服：Go 會自動將 `./uploads/` 目錄掛載至 `/uploads`，前端可直接存取照片 URL。

---

### 啟動 Python 推論引擎

```bash
cd python

# 安裝依賴（使用 uv）
uv sync

# 執行推論引擎
uv run main.py
```

首次啟動時會立刻執行一次預測，之後每 5 分鐘排程執行。

若需要重新訓練模型：
```bash
# 以模擬資料重新訓練並儲存 planter_rf_model.joblib
uv run train_model.py
```

---

### ESP32 韌體燒錄

#### 1. 安裝 Arduino 依賴庫

在 Arduino IDE 的 Library Manager 安裝以下庫：

| 庫名稱 | 版本 |
|---|---|
| PubSubClient | 2.8+ |
| ArduinoJson | 7.x |
| DHT sensor library (Adafruit) | 1.4+ |
| BH1750 | 2.x |
| Adafruit SSD1306 | 2.5+ |
| FluxGarage RoboEyes | 最新 |

#### 2. 建立硬體設定檔

複製範例設定並填寫：

```bash
cp esp32/env.example.h esp32/env.h
```

`env.h` 必填項目：

```cpp
// WiFi 憑證
#define SSID_NAME     "你的WiFi名稱"
#define SSID_PASSWORD "你的WiFi密碼"

// 此盆植物的唯一裝置 ID（需與 .env 中 TARGET_DEVICE_ID 一致）
#define DEVICE_ID "esp32-s3-01"

// Go 後端伺服器的區網 IP（不能填 localhost）
#define MQTT_SERVER   "192.168.x.x"
#define BACKEND_PORT  3000

// MQTT 設定
#define MQTT_PORT   1883
#define MQTT_TOPIC  "aiot-planter/v1/devices/" DEVICE_ID "/sensor"
```

#### 3. 燒錄

1. 在 Arduino IDE 選擇開發板：`ESP32S3 Dev Module`
2. 連接 ESP32 至電腦，選擇對應 COM Port
3. 點擊 Upload

---

### 存取前端

後端啟動後，以瀏覽器開啟：

```
http://<伺服器IP>:3000/frontend/login.html
```

- 第一次使用先至 `register.html` 建立帳號
- 登入後自動跳轉至 `dashboard.html`

> **本機開發**：若後端與瀏覽器在同一台機器，使用 `http://localhost:3000/frontend/login.html`

---

## API 參考

### 公開路由（不需 JWT）

| Method | Endpoint | 說明 | Body |
|---|---|---|---|
| `POST` | `/api/user` | 註冊帳號 | `{ username, password, email }` |
| `POST` | `/api/login` | 登入，回傳 JWT | `{ username, password }` |
| `GET` | `/api/sensors/latest` | 取得最新感測值 | Query: `?device_id=<id>` |
| `GET` | `/api/sensors/history` | 取得歷史感測記錄 | Query: `?device_id=<id>&limit=<n>` |
| `POST` | `/api/upload` | ESP32 上傳照片（raw binary） | Query: `?device_id=<id>` |
| `POST` | `/api/mood` | Python ML 寫入心情預測 | `{ device_id, mood }` |
| `POST` | `/api/notify/send` | 寄送植物狀態 Email | `{ device_id, recipient, sender, app_password }` |

### 需要 JWT 驗證（Header: `Authorization: Bearer <token>`）

| Method | Endpoint | 說明 | Body/Query |
|---|---|---|---|
| `GET` | `/api/mood/latest` | 取得最新心情 | Query: `?device_id=<id>` |
| `GET` | `/api/mood/history` | 取得心情歷史 | Query: `?device_id=<id>&limit=<n>` |
| `POST` | `/api/devices/:id/capture` | 觸發 ESP32 拍照 | Path: `device_id` |
| `GET` | `/api/photos` | 取得裝置照片生長紀錄 | Query: `?device_id=<id>&limit=<n>` |
| `GET` | `/api/user` | 查詢所有使用者 | — |
| `PUT` | `/api/user` | 更新使用者資料 | `{ username, email, ... }` |

### 靜態資源

| 路徑 | 說明 |
|---|---|
| `GET /uploads/<device_id>/<filename>.jpg` | 直接存取 ESP32 上傳的照片 |

---

## MQTT 主題

| 方向 | 主題 | Payload | 說明 |
|---|---|---|---|
| ESP32 → Server | `aiot-planter/v1/devices/<id>/sensor` | `{ device_id, temperature, humidity, soil_moisture, light_level }` | 每 5 分鐘上報 |
| Server → ESP32 | `aiot-planter/v1/devices/<id>/mood` | `happy` \| `sad` \| `sick` \| `hot` \| `normal` | Python ML 預測結果 |
| Server → ESP32 | `aiot-planter/v1/devices/<id>/mood` | `{ "cmd": "capture" }` | 觸發 ESP32 拍照 |

心情分類（Random Forest 輸出）：

| 心情 | 說明 | OLED 表情 |
|---|---|---|
| `happy` | 環境適宜 | 😊 笑眼動畫 |
| `normal` | 普通狀態 | 😐 眨眼 |
| `sad` | 水分不足 | 😢 向下眼神 |
| `sick` | 多項數值異常 | 🤒 困惑動畫 |
| `hot` | 溫度過高 | 🥵 憤怒眼神 |

---

## 目錄結構

```
aiot-planter/
├── .env                        # 環境變數（不進 git）
├── .env.example                # 範例環境變數
│
├── backend/                    # Go 後端 API
│   ├── main.go                 # 應用程式入口：初始化 DB、MQTT、Gin
│   ├── go.mod / go.sum         # Go 模組依賴
│   ├── broker/
│   │   └── mqtt.go             # MQTT client 設定、訂閱感測資料
│   ├── database/
│   │   └── db.go               # SQLite 連線、GORM AutoMigrate
│   ├── handler/
│   │   ├── user_handler.go     # 帳號註冊、登入（bcrypt + JWT）
│   │   ├── sensor_handler.go   # 感測資料 GET
│   │   ├── mood_handler.go     # 心情讀寫 + MQTT 下行
│   │   ├── device_handler.go   # 裝置控制（觸發拍照）
│   │   ├── upload_handler.go   # ESP32 照片上傳（raw binary）
│   │   ├── photos_handler.go   # 照片生長紀錄 GET（依裝置分類）
│   │   └── notify_handler.go   # Email 通知（附最新照片）
│   ├── middleware/
│   │   └── auth.go             # JWT 驗證 Middleware
│   ├── model/
│   │   ├── user.go             # User GORM model
│   │   ├── sensor.go           # SensorRecord GORM model
│   │   └── mood.go             # MoodRecord GORM model
│   ├── router/
│   │   └── router.go           # Gin 路由統一設定（含 CORS）
│   └── uploads/                # ESP32 照片儲存目錄
│       └── <device_id>/        # 按裝置分類的子目錄
│           └── capture_YYYYMMDD_HHMMSS.jpg
│
├── esp32/                      # ESP32-S3 韌體
│   ├── esp32.ino               # 主程式（Non-Blocking 輪詢 + MQTT callback）
│   ├── env.h                   # 硬體設定（WiFi/MQTT 帳密，不進 git）
│   ├── env.example.h           # 設定檔範例
│   ├── camera_module/          # OV2640 初始化 + JPEG 上傳邏輯
│   ├── connect_wifi/           # WiFi 連線與重連
│   ├── connect_mqtt/           # MQTT 重連機制
│   └── soil/                   # 土壤感測器 ADC 校正與百分比換算
│
├── frontend/                   # Web 前端
│   ├── login.html              # 登入頁
│   ├── register.html           # 註冊頁
│   └── dashboard.html          # 主儀表板（感測圖表、生長紀錄、設定）
│
└── python/                     # ML 推論引擎
    ├── main.py                 # 排程推論入口（每 5 分鐘）
    ├── train_model.py          # 模型訓練腳本
    ├── generate_mock_data.py   # 模擬感測資料生成
    ├── logger_config.py        # 結構化 Log 設定
    ├── planter_rf_model.joblib # 已訓練的 Random Forest 模型
    ├── sensors_dataset.csv     # 訓練用資料集
    └── pyproject.toml          # uv 套件依賴定義
```

---

## Troubleshooting

**ESP32 螢幕一直重啟（Brownout）**

相機與 OLED 同時啟動瞬間電流過大。請確認供電為穩定 5V / 1A 以上（建議使用 USB 充電頭而非電腦 USB 埠）。

**ESP32 連上 WiFi 但無法連到 MQTT**

`env.h` 中的 `MQTT_SERVER` 必須填區網 IP（如 `192.168.1.100`），不能填 `localhost`（那是 ESP32 自己）。

**Go 後端啟動失敗：`MQTT_BROKER 未設定`**

確認 `.env` 存在於專案根目錄（`backend/` 的上一層），且 `MQTT_BROKER` 未空白。

**Python 推論結果沒有出現在儀表板**

1. 確認 Go 後端已啟動且 `PORT` 一致
2. 確認 `.env` 的 `TARGET_DEVICE_ID` 與 ESP32 `env.h` 的 `DEVICE_ID` 相同
3. 查看 Python log — `ModelServer` 是否成功載入 `planter_rf_model.joblib`

**前端圖表一片空白**

開啟 F12 > Network，確認 `/api/sensors/history` 是否有回應。若 401，表示 JWT Token 遺失；若 404，表示該裝置 ID 尚無資料。

**照片生長紀錄頁面看不到照片**

1. 確認 ESP32 已成功拍照上傳（Go log 會顯示 `✅ 照片已儲存`）
2. 確認 `backend/uploads/<device_id>/` 目錄下有 `.jpg` 檔案
3. 直接瀏覽 `http://localhost:3000/uploads/<device_id>/<filename>.jpg` 確認靜態伺服是否正常

**Email 寄送失敗**

Gmail 需要使用「應用程式密碼」（App Password），不是帳號密碼。前往 Google 帳戶 > 安全性 > 兩步驟驗證 > 應用程式密碼 產生專屬密碼。
