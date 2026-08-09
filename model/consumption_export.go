package model

import (
	"context"
	"errors"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

type ConsumptionExportFile struct {
	ID        int64  `json:"id" gorm:"primary_key"`
	TaskID    string `json:"task_id" gorm:"type:varchar(64);uniqueIndex"`
	UserID    int    `json:"user_id" gorm:"index"`
	FileName  string `json:"file_name" gorm:"type:varchar(255)"`
	Content   []byte `json:"-"`
	ExpiresAt int64  `json:"expires_at" gorm:"bigint;index"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
}

func (file *ConsumptionExportFile) BeforeCreate(_ *gorm.DB) error {
	if file.CreatedAt == 0 {
		file.CreatedAt = common.GetTimestamp()
	}
	return nil
}

func SaveConsumptionExportFile(file *ConsumptionExportFile) error {
	return DB.Create(file).Error
}

func GetConsumptionExportFile(taskID string, userID int, now int64) (*ConsumptionExportFile, error) {
	var file ConsumptionExportFile
	err := DB.Where("task_id = ? AND user_id = ? AND expires_at > ?", taskID, userID, now).First(&file).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &file, nil
}

func DeleteConsumptionExportFile(taskID string, userID int) error {
	return DB.Where("task_id = ? AND user_id = ?", taskID, userID).Delete(&ConsumptionExportFile{}).Error
}

func DeleteConsumptionExportTask(taskID string, userID int) (bool, error) {
	deleted := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("task_id = ? AND user_id = ? AND type = ? AND status NOT IN ?", taskID, userID, SystemTaskTypeConsumptionExport, activeSystemTaskStatuses()).
			Delete(&SystemTask{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if err := tx.Where("task_id = ? AND user_id = ?", taskID, userID).Delete(&ConsumptionExportFile{}).Error; err != nil {
			return err
		}
		deleted = true
		return nil
	})
	return deleted, err
}

func DeleteExpiredConsumptionExportFiles(ctx context.Context, now int64) (int64, error) {
	result := DB.WithContext(ctx).Where("expires_at <= ?", now).Delete(&ConsumptionExportFile{})
	return result.RowsAffected, result.Error
}

type ConsumptionExportLogQuery struct {
	UserID         int
	StartTimestamp int64
	EndTimestamp   int64
	TokenID        int
}

func CountConsumptionExportLogs(ctx context.Context, query ConsumptionExportLogQuery) (int64, error) {
	tx := consumptionExportLogQuery(ctx, query)
	var total int64
	err := tx.Model(&Log{}).Count(&total).Error
	return total, err
}

func GetConsumptionExportLogs(ctx context.Context, query ConsumptionExportLogQuery, offset int, limit int) ([]*Log, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 1000
	}
	tx := consumptionExportLogQuery(ctx, query)
	order := "created_at asc, id asc"
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		order = "created_at asc, request_id asc"
	}
	var logs []*Log
	err := tx.Order(order).Offset(offset).Limit(limit).Find(&logs).Error
	return logs, err
}

func consumptionExportLogQuery(ctx context.Context, query ConsumptionExportLogQuery) *gorm.DB {
	tx := LOG_DB.WithContext(ctx).
		Where("user_id = ? AND type IN ?", query.UserID, []int{LogTypeConsume, LogTypeRefund}).
		Where("created_at >= ? AND created_at <= ?", query.StartTimestamp, query.EndTimestamp)
	if query.TokenID > 0 {
		tx = tx.Where("token_id = ?", query.TokenID)
	}
	return tx
}
