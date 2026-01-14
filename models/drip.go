package models

import (
	"time"

	"gorm.io/gorm"
)

type Drip struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Recipient   string         `gorm:"size:42;not null;index:idx_recipient_token" json:"recipient"`
	TokenID     string         `gorm:"size:20;not null;index:idx_recipient_token" json:"tokenId"`
	Amount      string         `gorm:"size:30;not null" json:"amount"`
	TxHash      string         `gorm:"size:66;index" json:"txHash"`
	IPAddress   string         `gorm:"type:inet;index" json:"ipAddress"`
	Fingerprint string         `gorm:"size:64;index" json:"fingerprint"`
	Status      string         `gorm:"size:20;default:pending;index" json:"status"`
	Error       string         `gorm:"type:text" json:"error,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	CompletedAt *time.Time     `json:"completedAt,omitempty"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
