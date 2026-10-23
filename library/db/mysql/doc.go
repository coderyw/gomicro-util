// Package mysql GORM 多实例数据库封装。支持驱动：mysql、postgresql、opengauss。
//
// pmicro 启动时按 storage.mysql 配置自动初始化，支持分表与雪花主键（replicas 从库配置当前有缺陷，见文档）。
// 多租户下会按 ctx 中的租户自动改写库名，因此务必传入请求链路上的 ctx。
//
// 使用方式：
//
//  1. 创建数据库实例
//
//     cronJob,close,err := InitConnect(serverName, config.DbConfig)
//
//  2. 数据库实例
//
//     获取默认数据库实例
//     db := DB(context.Context)
//
//     获取指定实例
//     db:= DB(context.Context, instanceName)
//
//  3. 数据库操作
//
//     创建
//     db.Create(&entity)
//
//     查询
//     db.Model(Entity{}).First(&Entity, id)
//
//     更新
//     db.Model(&entity).Where("active = ?", true).Update("name", "hello")
//
//     删除
//     db.Where("name = ?", "hello").Delete(&Entity{})
//
//  4. 事务
//
//     db.Transaction(func(tx *gorm.DB) error {
//     })
//
// 文档：docs/components/mysql.md
package mysql
