// Package callback
// @Author: yinwei
// @File: db_info
// @Version: 1.0.0
// @Date: 2025/4/14 22:57

package callback

import (
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func RegisterConnectionLogger(db *gorm.DB) {
	db.Callback().Create().Before("gorm:create").Register("connlog:before", printConnInfo)
	db.Callback().Query().Before("gorm:query").Register("connlog:before", printConnInfo)
	db.Callback().Update().Before("gorm:update").Register("connlog:before", printConnInfo)
	db.Callback().Delete().Before("gorm:delete").Register("connlog:before", printConnInfo)
	db.Callback().Row().Before("gorm:row").Register("connlog:before", printConnInfo)
	db.Callback().Raw().Before("gorm:raw").Register("connlog:before", printConnInfo)
}

func printConnInfo(db *gorm.DB) {
	// 打印连接池状态
	sqlDB, err := db.DB()
	if err == nil {
		stats := sqlDB.Stats()
		log.DebugCtx(db.Statement.Context, "连接池信息",
			zap.Int("InUse", stats.InUse),
			zap.Int("Idle", stats.Idle),
			zap.Int("Open", stats.OpenConnections),
			zap.Int64("WaitCount", stats.WaitCount),
			zap.Int64("MaxIdleClosed", stats.MaxIdleClosed),
			zap.Int64("MaxIdleTimeClosed", stats.MaxIdleTimeClosed),
			zap.Int64("MaxLifetimeClosed", stats.MaxLifetimeClosed),
			zap.String("WaitDuration", stats.WaitDuration.String()),
		)
	} else {
		log.WarnCtx(db.Statement.Context, "[连接池] 获取失败", zap.Error(err))
	}
}
