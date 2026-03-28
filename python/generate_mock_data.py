import csv
import random
import logging
from pathlib import Path

from logger_config import setup_logger
logger = setup_logger(__name__)

# --- MLOps Data Pipeline: 產生假資料 ---
def generate_succulent_data(num_samples: int = 2000) -> str:
    file_path = Path("./sensors_dataset.csv")    

    with open(file_path, mode="w", newline="") as file:
        writer = csv.writer(file)
        writer.writerow(["temperature", "humidity", "soil_moisture", "light_level", "mood"])

        for _ in range(num_samples):
            temp = round(random.uniform(10.0, 45.0), 1)     
            humidity = round(random.uniform(20.0, 95.0), 1)
            moisture = random.randint(0, 100)               
            light = round(random.uniform(0.0, 65000.0), 1) 

            # --- 決定多肉的心情 ---
            if temp > 35.0:
                mood = "hot"          
            elif moisture > 50:
                mood = "sick"        
            elif light < 800.0 and moisture > 30:
                mood = "sad"
            elif moisture < 20 and light >= 1500.0:
                mood = "happy"       
            else:
                mood = "normal"      
            
            writer.writerow([temp, humidity, moisture, light, mood])
    
    logger.info(f"✅：成功產生 {num_samples} 筆訓練資料，存於 {file_path}")

    return str(file_path)

if __name__ == "__main__":
    generate_succulent_data()
