package models

import "time"

type Uid struct {
	ID         int        `gorm:"column:id;primaryKey;not null;autoIncrement;comment:自增ID"`
	BusinessId string     `json:"businessId" gorm:"column:business_id;type:varchar(255)"` // 业务ID
	MaxId      int64      `json:"maxId" gorm:"column:max_id"`                             // 当前的最大ID
	Step       int64      `json:"step" gorm:"column:step"`                                // 步进ID
	Status     int        `json:"status" gorm:"column:status"`
	CreatedAt  *time.Time `gorm:"column:createdAt;not null;comment:创建时间"`
	UpdatedAt  *time.Time `gorm:"column:updatedAt;not null;comment:更新时间"`
}
