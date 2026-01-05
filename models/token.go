package models

import (
	"time"

	"gorm.io/gorm"
)

type Token struct {
	ID            string         `gorm:"primaryKey;size:20" json:"id"`
	Name          string         `gorm:"size:50;not null" json:"name"`
	Symbol        string         `gorm:"size:10;not null" json:"symbol"`
	Address       string         `gorm:"size:42" json:"address"` // Empty for native ETH
	DripAmount    string         `gorm:"size:30;not null" json:"dripAmount"`
	CooldownHours int            `gorm:"not null;default:24" json:"cooldownHours"`
	Decimals      int            `gorm:"not null;default:18" json:"decimals"`
	LogoURL       string         `gorm:"size:200" json:"logoUrl"`
	IsActive      bool           `gorm:"default:true" json:"isActive"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type TokenStats struct {
	TokenID    string `json:"tokenId"`
	TotalDrips int64  `json:"totalDrips"`
	Balance    string `json:"balance"`
}
