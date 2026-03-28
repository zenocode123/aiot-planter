package model

import "gorm.io/gorm"

type MoodRecord struct {
	gorm.Model
	DeviceId string `json:"device_id"`
	Mood     string `json:"mood"`
}
