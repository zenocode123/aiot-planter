package model

import "gorm.io/gorm"

type SensorRecord struct {
	gorm.Model
	DeviceId     string  `json:"device_id"`
	Temperature  float64 `json:"temperature"`
	Humidity     float64 `json:"humidity"`
	SoilMoisture int     `json:"soil_moisture"`
	LightLevel   float64 `json:"light_level"`
}
