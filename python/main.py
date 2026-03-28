import sys
import time
import requests
import joblib
import logging
import schedule
import os
import pandas as pd
from pathlib import Path
from dotenv import load_dotenv
from pydantic import BaseModel, Field, ValidationError

load_dotenv(Path(__file__).parent.parent / ".env")

from logger_config import setup_logger
logger = setup_logger(__name__)

# -- 系統參數 (自動從 .env 讀取) --
PORT      = os.getenv("PORT")
DEVICE_ID = os.getenv("TARGET_DEVICE_ID")

# 啟動時驗證必要的 .env 變數
_missing = [k for k, v in {"PORT": PORT, "TARGET_DEVICE_ID": DEVICE_ID}.items() if not v]
if _missing:
    logger.error(f"❌ .env 缺少必要變數: {', '.join(_missing)}，請確認 .env 設定後重新啟動。")
    sys.exit(1)

API_BASE      = f"http://localhost:{PORT}"
SENSOR_URL    = f"{API_BASE}/api/sensors/latest?device_id={DEVICE_ID}"
MOOD_POST_URL = f"{API_BASE}/api/mood"
MODEL_PATH    = "planter_rf_model.joblib"


class SensorData(BaseModel):
    device_id:     str
    temperature:   float = Field(..., description="環境溫度")
    humidity:      float = Field(..., description="環境濕度")
    soil_moisture: int   = Field(..., description="土壤濕度")
    light_level:   float = Field(..., description="光照度")


class ModelServer:
    def __init__(self, model_path: str):
        logger.info(f"載入訓練模型：{model_path} ...")
        self.model = joblib.load(model_path)

    def predict_mood(self, sensor: SensorData) -> str:
        features_df = pd.DataFrame([{
            "temperature":   sensor.temperature,
            "humidity":      sensor.humidity,
            "soil_moisture": sensor.soil_moisture,
            "light_level":   sensor.light_level,
        }])
        return self.model.predict(features_df)[0]


def fetch_and_predict(server: ModelServer) -> None:
    try:
        # 1. 取得最新感測資料（公開路由，不需 JWT）
        logger.info("獲取最新感測器數據...")
        resp = requests.get(SENSOR_URL, timeout=5)
        resp.raise_for_status()
        data_json = resp.json()

        if not data_json.get("success"):
            logger.error(f"❌ API 錯誤：{data_json}")
            return

        # 2. 資料清洗與型態防禦
        sensor_record = SensorData(**data_json["data"])

        # 3. 模型預測
        mood = server.predict_mood(sensor_record)
        logger.info(f"預測結果：{mood} (Temp: {sensor_record.temperature}, Soil: {sensor_record.soil_moisture})")

        # 4. 回傳心情至 Go（公開路由，不需 JWT）
        post_resp = requests.post(
            MOOD_POST_URL,
            json={"device_id": sensor_record.device_id, "mood": mood},
            timeout=5,
        )
        post_resp.raise_for_status()
        logger.info("✅ 成功將心情 Post 至後端！")

    except ValidationError as ve:
        logger.error(f"❌ 異常資料格式：{ve.errors()}")
    except requests.RequestException as re:
        logger.error(f"❌ 網路連線錯誤：{re}")
    except Exception as e:
        logger.error(f"❌ 發生未預期的錯誤：{e}", exc_info=True)


def main():
    logger.info("AIoT Planter 推論引擎啟動...")
    server = ModelServer(MODEL_PATH)

    # 第一次立刻執行
    fetch_and_predict(server)

    # 每 5 分鐘排程執行
    schedule.every(5).minutes.do(fetch_and_predict, server=server)
    logger.info("⏱️ 已排程完成，每 5 分鐘執行一次！(按 Ctrl+C 結束)")

    while True:
        schedule.run_pending()
        time.sleep(1)


if __name__ == "__main__":
    main()
