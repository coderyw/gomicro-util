/**
 * @Author: yinwei
 * @Description:
 * @File: sharding
 * @Version: 1.0.0
 * @Date: 2023/11/1 09:19
 */
package mysql

import (
	"github.com/pkg/errors"
	"github.com/coderyw/gomicro-util/library/db/mysql/sharding"
	"github.com/coderyw/gomicro-util/library/log"
	config2 "github.com/coderyw/gomicro-util/model/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Sharding struct {
	suffixFunc func(columnValue any) (suffix string, err error)
}

func (s *Sharding) Name() string {
	return "sharding_plus"
}

func (s *Sharding) Initialize(db *gorm.DB) error {
	if s.suffixFunc == nil {
		return nil
	}
	db.Callback().Create().Before("*").Register("gorm:sharding", s.switchConn)
	db.Callback().Query().Before("*").Register("gorm:sharding", s.switchConn)
	db.Callback().Update().Before("*").Register("gorm:sharding", s.switchConn)
	db.Callback().Delete().Before("*").Register("gorm:sharding", s.switchConn)
	db.Callback().Row().Before("*").Register("gorm:sharding", s.switchConn)
	db.Callback().Raw().Before("*").Register("gorm:sharding", s.switchConn)

	return nil
}

func (s *Sharding) switchConn(db *gorm.DB) {

	//s.suffixFunc()
	//db.Statement.Table = db.Get()
}

// 初始化分表
func initSharding(db *gorm.DB, conf config2.DbInstanceConfig) error {
	if len(conf.Sharding) > 0 {
		sh := new(sharding.Sharding)
		// 遍历所有分表配置
		for _, v := range conf.Sharding {
			log.Info("初始化分表", zap.Any("sharding", v))
			cfg := sharding.Config{
				DoubleWrite:       false,                            // 不启用双写
				ShardingKey:       v.ShardingKey,                    // 分表键
				NumberOfShards:    v.ShardingNum,                    // 分表数量
				ShardingAlgorithm: ShardingAlgorithm(v.ShardingNum), // 分表算法
			}

			// 根据主键生成策略配置不同的生成器
			switch v.PKGenerator {
			case sharding.PKSnowflake: // 雪花算法使用自定义的
				if conf.SnowflakeGen == nil || len(conf.SnowflakeGen.Snowflakes) == 0 {
					return errors.New("雪花id未配置")
				}
				cfg.PrimaryKeyGeneratorFn = func(tableIndex int64) int64 {
					return 0 // 实际ID生成在GORM回调中处理
				}
				cfg.PrimaryKeyGenerator = sharding.PKCustom
			case sharding.PKPGSequence, sharding.PKMySQLSequence:
				// 使用数据库自带的序列
				cfg.PrimaryKeyGenerator = int(v.PKGenerator)
			case sharding.PKCustom:
				// 自定义主键生成器
				cfg.PrimaryKeyGenerator = int(v.PKGenerator)
				cfg.PrimaryKeyGeneratorFn = func(tableIndex int64) int64 {
					return 0
				}
			}

			// 注册分表配置到分表处理器
			sh = sh.Register(cfg, v.TableName)
			db.Attrs()
		}
		// 将分表插件注册到GORM
		if err := db.Use(sh); err != nil {
			return err
		}
	}
	return nil
}
