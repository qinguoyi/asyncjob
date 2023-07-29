package repo

import (
	"github.com/qinguoyi/asyncjob/app/models"
	"gorm.io/gorm"
)

type uidRepo struct{}

func NewUidRepo() *uidRepo { return &uidRepo{} }

// GetAllBusinessID .
func (u *uidRepo) GetAllBusinessID(db *gorm.DB) ([]models.Uid, error) {
	var ret []models.Uid
	if err := db.Where("status = ?", 1).Find(&ret).Error; err != nil {
		return ret, err
	}
	return ret, nil
}

// GetByBusinessID .
func (u *uidRepo) GetByBusinessID(db *gorm.DB, bizId string) (*models.Uid, error) {
	ret := &models.Uid{}
	if err := db.Where("business_id = ? and status = ?", bizId, 1).First(ret).Error; err != nil {
		return ret, err
	}
	return ret, nil
}

// Updates .
func (u *uidRepo) Updates(db *gorm.DB, bizId string, columns map[string]interface{}) error {
	err := db.Model(&models.Uid{}).Where("business_id = ?", bizId).Updates(columns).Error
	return err
}
