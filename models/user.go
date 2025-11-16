package models

import _ "gorm.io/gorm"

type User struct {
	UserId   string `gorm:"primaryKey;column:user_id" json:"user_id"`
	UserName string `gorm:"column:username" json:"username"`
	TeamName string `gorm:"not null" json:"team_name"`
	IsActive bool   `json:"is_active"`
}
