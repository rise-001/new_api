package model

import (
	"context"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

func removeLegacyConsumptionExportStorage() error {
	const legacyTaskType = "consumption_export"
	if err := DB.Where("type = ?", legacyTaskType).Delete(&SystemTaskLock{}).Error; err != nil {
		return err
	}
	if err := DB.Where("type = ?", legacyTaskType).Delete(&SystemTask{}).Error; err != nil {
		return err
	}
	return DB.Exec("DROP TABLE IF EXISTS consumption_export_files").Error
}

type ConsumptionExportLogQuery struct {
	UserID         int
	StartTimestamp int64
	EndTimestamp   int64
	TokenID        int
	TokenName      *string
}

type ConsumptionExportLogCursor struct {
	CreatedAt int64
	ID        int
	Offset    int
}

func CountConsumptionExportLogs(ctx context.Context, query ConsumptionExportLogQuery) (int64, error) {
	tx := consumptionExportLogQuery(ctx, query)
	var total int64
	err := tx.Model(&Log{}).Count(&total).Error
	return total, err
}

func GetConsumptionExportLogsAfter(ctx context.Context, query ConsumptionExportLogQuery, cursor ConsumptionExportLogCursor, limit int) ([]*Log, error) {
	if limit <= 0 {
		limit = 1000
	}
	tx := consumptionExportLogQuery(ctx, query)
	order := "created_at asc, id asc"
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		order = "created_at asc, request_id asc"
		tx = tx.Offset(cursor.Offset)
	} else if cursor.CreatedAt > 0 {
		tx = tx.Where("created_at > ? OR (created_at = ? AND id > ?)", cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
	}
	var logs []*Log
	err := tx.Order(order).Limit(limit).Find(&logs).Error
	return logs, err
}

func consumptionExportLogQuery(ctx context.Context, query ConsumptionExportLogQuery) *gorm.DB {
	tx := LOG_DB.WithContext(ctx).
		Where("user_id = ? AND type IN ?", query.UserID, []int{LogTypeConsume, LogTypeRefund}).
		Where("created_at >= ? AND created_at <= ?", query.StartTimestamp, query.EndTimestamp)
	if query.TokenID > 0 {
		tx = tx.Where("token_id = ?", query.TokenID)
	}
	if query.TokenName != nil {
		tx = tx.Where("token_name = ?", *query.TokenName)
	}
	return tx
}
