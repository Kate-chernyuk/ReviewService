package models

import _ "gorm.io/gorm"

type Team struct {
	TeamName string `gorm:"primaryKey" json:"team_name"`
	Members  []User `gorm:"foreignKey:TeamName;references:TeamName" json:"members"`
}

func (Team) TableName() string {
	return "teams"
}
