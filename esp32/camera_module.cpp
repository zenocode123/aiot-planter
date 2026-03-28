#include "camera_module.h"
#include "env.h"
#include <esp_camera.h>
#include <HTTPClient.h>
#include <WiFi.h>

void init_camera() {
  camera_config_t config;
  config.ledc_channel = LEDC_CHANNEL_0;
  config.ledc_timer = LEDC_TIMER_0;
  config.pin_d0 = Y2_GPIO_NUM;
  config.pin_d1 = Y3_GPIO_NUM;
  config.pin_d2 = Y4_GPIO_NUM;
  config.pin_d3 = Y5_GPIO_NUM;
  config.pin_d4 = Y6_GPIO_NUM;
  config.pin_d5 = Y7_GPIO_NUM;
  config.pin_d6 = Y8_GPIO_NUM;
  config.pin_d7 = Y9_GPIO_NUM;
  config.pin_xclk = XCLK_GPIO_NUM;
  config.pin_pclk = PCLK_GPIO_NUM;
  config.pin_vsync = VSYNC_GPIO_NUM;
  config.pin_href = HREF_GPIO_NUM;
  config.pin_sccb_sda = SIOD_GPIO_NUM;
  config.pin_sccb_scl = SIOC_GPIO_NUM;
  config.pin_pwdn = PWDN_GPIO_NUM;
  config.pin_reset = RESET_GPIO_NUM;
  config.xclk_freq_hz = 20000000;
  
  // 設定影像大小與格式，VGA 代表 640x480 (適合影像辨識與穩定傳輸)
  config.frame_size = FRAMESIZE_VGA;
  config.pixel_format = PIXFORMAT_JPEG; 
  
  // 關鍵修復：改用 CAMERA_GRAB_LATEST 並將 fb_count 設為 2。
  // 這樣相機驅動會永遠捨棄舊的未讀取畫面，確保每次呼叫 fb_get 時拿到的都是「最新鮮」的畫面。
  config.grab_mode = CAMERA_GRAB_LATEST;
  config.fb_location = CAMERA_FB_IN_PSRAM;
  config.jpeg_quality = 12;
  config.fb_count = 2;

  // 初始化相機
  esp_err_t err = esp_camera_init(&config);
  if (err != ESP_OK) {
    Serial.printf("相機初始化失敗 錯誤碼: 0x%x\n", err);
    return;
  }
  Serial.println("相機初始化成功！");
}

void take_and_upload_photo(const String& uploadUrl) {
  if (WiFi.status() != WL_CONNECTED) {
    Serial.println("WiFi 未連線，無法上傳照片。");
    return;
  }

  Serial.println("正在擷取影像...");
  camera_fb_t *fb = esp_camera_fb_get();
  if (!fb) {
    Serial.println("相機影像獲取失敗");
    return;
  }

  Serial.printf("影像擷取成功，大小：%u bytes. 準備上傳至 %s\n", fb->len, uploadUrl.c_str());

  HTTPClient http;
  http.begin(uploadUrl);
  
  // 使用 raw bytes 傳輸 JPEG 圖片，這在 ESP32 上記憶體效率最高
  http.addHeader("Content-Type", "image/jpeg");
  
  // 執行 HTTP POST
  int httpResponseCode = http.POST(fb->buf, fb->len);

  if (httpResponseCode > 0) {
    String response = http.getString();
    Serial.printf("照片上傳成功，HTTP 回應碼: %d\n", httpResponseCode);
    Serial.printf("伺服器回應: %s\n", response.c_str());
  } else {
    Serial.printf("照片上傳失敗，錯誤碼: %d, 訊息: %s\n", httpResponseCode, http.errorToString(httpResponseCode).c_str());
  }
  
  http.end();
  
  // 釋放記憶體非常重要，否則下一次無法拍照
  esp_camera_fb_return(fb);
}
