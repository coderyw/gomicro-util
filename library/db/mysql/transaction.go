package mysql

import (
	"context"
	"testing"

	"github.com/coderyw/gomicro-util/library/mock"
	"gorm.io/gorm"
)

// Transaction 开启数据库事务（使用默认数据库连接）
// 自动处理事务的开始、提交和回滚，如果业务函数返回错误则自动回滚，否则自动提交
// 参数:
//   - ctx: 上下文对象，用于传递请求链路信息和控制事务生命周期
//   - txFunc: 事务处理函数，接收事务对象作为参数，返回错误则回滚事务
//
// 返回值:
//   - error: 事务处理过程中的错误，包括业务逻辑错误和数据库操作错误
//
// 使用示例:
//
//	err := Transaction(ctx, func(ctx context.Context,tx *gorm.DB) error {
//	    // 在事务中执行多个数据库操作
//	    if err := tx.Create(&user).Error; err != nil {
//	        return err // 返回错误会自动回滚事务
//	    }
//	    if err := tx.Create(&profile).Error; err != nil {
//	        return err
//	    }
//	    return nil // 返回nil会自动提交事务
//	})
func Transaction(ctx context.Context, txFunc func(ctx context.Context, tx *gorm.DB) error) error {
	if d := mock.CheckAndGetMockClient[*gorm.DB](ctx, func(t *testing.T) *gorm.DB {
		return new(gorm.DB)
	}); d != nil {
		return txFunc(ctx, d)
	}
	return DB(ctx).Transaction(func(tx *gorm.DB) error {
		// 确保事务对象携带上下文信息，用于链路追踪和超时控制
		tx = tx.WithContext(ctx)
		return txFunc(ctx, tx)
	})
}

// TransactionByDB 从指定的数据库连接中开启新事务
// 与Transaction函数的区别是可以指定使用哪个数据库连接，适用于多数据库场景
// 自动处理事务的开始、提交和回滚，如果业务函数返回错误则自动回滚，否则自动提交
// 参数:
//   - ctx: 上下文对象，用于传递请求链路信息和控制事务生命周期
//   - db: 指定的GORM数据库连接实例，通常通过DB(ctx, "database_name")获取
//   - txFunc: 事务处理函数，接收事务对象作为参数，返回错误则回滚事务
//
// 返回值:
//   - error: 事务处理过程中的错误，包括业务逻辑错误和数据库操作错误
//
// Returns:
//   - error
//
// 使用示例:
//
//	// 在指定数据库中执行事务
//	gameDB := DB(ctx, "game_database")
//	err := TransactionByDB(ctx, gameDB, func(tx *gorm.DB) error {
//	    // 在game_database的事务中执行操作
//	    if err := tx.Create(&gameRecord).Error; err != nil {
//	        return err // 返回错误会自动回滚事务
//	    }
//	    return nil // 返回nil会自动提交事务
//	})
func TransactionByDB(ctx context.Context, db *gorm.DB, txFunc func(ctx context.Context, tx *gorm.DB) error) error {
	if d := mock.CheckAndGetMockClient[*gorm.DB](ctx, func(t *testing.T) *gorm.DB {
		return new(gorm.DB)
	}); d != nil {
		return txFunc(ctx, d)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		// 确保事务对象携带上下文信息，用于链路追踪和超时控制
		tx = tx.WithContext(ctx)
		return txFunc(ctx, tx)
	})
}
