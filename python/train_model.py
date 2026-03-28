import pandas as pd
import logging
from typing import Tuple
from sklearn.ensemble import RandomForestClassifier
from sklearn.model_selection import train_test_split
from sklearn.metrics import classification_report, accuracy_score
import joblib

from logger_config import setup_logger
logger = setup_logger(__name__)

def train_planter_model(data_path: str = "sensors_dataset.csv", model_out: str = "planter_rf_model.joblib") -> None:
    logger.info("開始訓練...")

    # Importing the dataset
    try:
        df = pd.read_csv(data_path)
        logger.info(f"✅：成功載入 Dataset，共 {len(df)} 筆")
    except Exception as e:
        logger.error(f"❌：無法讀取 {data_path}：{e}")

        return

    X = df.drop(['mood'], axis=1)
    y = df["mood"]

    # 20% test, 80% train
    X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=0)

    # Training Random Forest
    model = RandomForestClassifier(n_estimators=100, max_depth=10, random_state=0, oob_score=True)
    model.fit(X_train, y_train)

    # 預測
    y_pred = model.predict(X_test)

    # 驗證
    acc = accuracy_score(y_test, y_pred)

    # 實驗追蹤
    logger.info(f"testset 準確度：{acc * 100:.2f}%")
    logger.info(f"內部驗證 OOB Score：{model.oob_score_ * 100:.2f}%")

    # 儲存模型 Registry (File System)
    joblib.dump(model, model_out)
    logger.info(f"模型已成功封裝並儲存至：{model_out}")

if __name__ == "__main__":
    train_planter_model()
