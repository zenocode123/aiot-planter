package model

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `json:"username" gorm:"unique;not null"`
	Password string `json:"-"` // 表示在任何情況下轉成 JSON 時，都不要把此欄位回傳！
	Email    string `json:"email" gorm:"unique;not null"`
}
