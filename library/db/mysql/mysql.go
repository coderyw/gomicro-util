package mysql

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"hash/crc32"
	"os"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	_ "gitee.com/opengauss/openGauss-connector-go-pq"
	mysqlBase "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"gitlab.jkluck.vip/base/eonid/eonid-go/v2/eonid"
	"github.com/coderyw/gomicro-util/library/db/mysql/callback"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/library/mock"
	"github.com/coderyw/gomicro-util/library/pmicro/metrics"
	config2 "github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/cron"
	"github.com/coderyw/gomicro-util/model/micro"
	mysql "gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/dbresolver"

	"go.uber.org/zap"
)

var (
	// conn 默认的MySQL数据库连接实例
	conn *gorm.DB
	// dbCfg 数据库配置
	dbCfg config2.DbConfig
	// mulConn 多数据库连接映射表，用于管理多个数据库实例（并发安全）
	mulConn      sync.Map
	mulConnStats sync.Map
)

// NotFound GORM记录未找到错误的别名，方便外部使用
var NotFound = gorm.ErrRecordNotFound

// GetRDSErrDuplicatedKey 判断给定的错误是否为MySQL的唯一约束冲突错误（错误码1062）
// 用于处理插入重复数据时的错误判断
// 参数:
//   - err: 数据库操作返回的错误对象
//
// 返回值:
//   - bool: 如果错误是唯一约束冲突则返回true，否则返回false
func GetRDSErrDuplicatedKey(err error) bool {
	var mysqlErr *mysqlBase.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// 为了强制使用新的DB方法，暂时注释掉Conn方法
// Conn 获取数据库连接实例（已废弃）
// Deprecated: 获取实例时候没有携带上下文，所以无法使用traceId，建议使用DB()方法
// 参数:
//   - database: 可选的数据库名称，用于选择特定的数据库实例
//
// 返回值:
//   - *gorm.DB: GORM数据库连接实例
//func Conn(database ...string) *gorm.DB {
//	// 如果没有指定数据库名称，返回默认连接
//	if len(database) == 0 {
//		if conn == nil {
//			panic("获取mysql默认连接为空")
//		}
//		// 返回默认的mysql连接
//		return conn
//	}
//	// 根据数据库名称获取对应的连接实例
//	db := database[0]
//	if value, ok := mulConn.Load(db); ok {
//		return value.(*gorm.DB)
//	}
//	panic(fmt.Errorf("获取mysql连接为空: %v", database))
//}

// DB 获取数据库实例（推荐使用）
// 支持Mock模式，携带上下文信息，可以使用traceId进行链路追踪
// 参数:
//   - ctx: 上下文对象，用于传递请求链路信息
//   - database: 可选的数据库名称，用于选择特定的数据库实例
//
// 返回值:
//   - *gorm.DB: 带有上下文的GORM数据库连接实例
func DB(ctx context.Context, database ...string) *gorm.DB {
	// 检查是否为Mock测试模式，如果是则返回Mock数据库实例
	if d := mock.CheckAndGetMockClient[*gorm.DB](ctx, func(t *testing.T) *gorm.DB {
		return new(gorm.DB)
	}); d != nil {
		return d
	}
	// 如果没有指定数据库名称，返回带有上下文的默认连接
	if len(database) == 0 {
		if conn == nil {
			log.FatalCtx(ctx, "获取mysql连接为空", zap.Any("database", database))
			return nil
		}
		// 返回默认的mysql连接
		return conn.WithContext(ctx)
	}
	// 根据数据库名称获取对应的连接实例，并附加上下文
	db := database[0]
	if value, ok := mulConn.Load(db); ok {
		gdb := value.(*gorm.DB)
		return gdb.WithContext(ctx)
	}
	log.FatalCtx(ctx, "获取mysql连接为空", zap.Any("database", database))
	return nil
}

// 初始化 db 链接
func startDb(instance string, conf config2.DbBaseInstanceConfig) (db *gorm.DB, err error) {
	var sqlDb *sql.DB
	// 设置日志级别，Debug模式下使用Error级别避免过多日志输出
	var logLevel = logger.Info
	if conf.Debug {
		logLevel = logger.Error
	}
	// 默认使用MySQL驱动
	if len(conf.Driver) == 0 {
		conf.Driver = config2.DRIVER_MYSQL
	}
	dir, err := getDialector(instance, conf)
	if err != nil {
		return nil, err
	}
	// 连接数据库，配置GORM选项
	if db, err = gorm.Open(dir, &gorm.Config{
		// 跳过默认事物，如果设置false，gorm会自动提交我们的事物，导致手动事物被提交
		SkipDefaultTransaction: true,
		// 禁用外键约束检查，避免迁移时的问题
		DisableForeignKeyConstraintWhenMigrating: true,
		// 命名策略配置：不使用表前缀，使用单数表名
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: true,
			NameReplacer:  nil,
			NoLowerCase:   false,
		},
		// 使用自定义日志记录器
		Logger: log.NewGormLog(logLevel, conf.Driver),
		// 当前需要兼容华为云 ddm 没法在更新 decimal 时候会处理失败。
		// 比如，我使用字符串 1 更新 decimal，结果变成了 49，这个是因为字符串的二进制转成了 10进制
		// 同时第二个兼容点在连接数据库时候需要加上一个连接参数，详见下面的 mysql 驱动
		// 禁用预编译语句以兼容华为云DDM
		PrepareStmt: false,
	}); err != nil {
		return nil, errors.New(fmt.Sprintf("连接%s出错 - [%s]", conf.Driver, err.Error()))
	} else {

		// 设置连接池参数的默认值
		if conf.Idle == 0 {
			conf.Idle = 20 // 默认最大空闲连接数
		}

		if conf.Open == 0 {
			conf.Open = 200 // 默认最大打开连接数
		}

		// 最大空闲连接数不能大于最大打开连接数
		conf.Idle = min(conf.Idle, conf.Open)

		if conf.MaxLifeTime == 0 {
			conf.MaxLifeTime = 3600 // 默认一小时
		}
		if conf.MaxIdleTime == 0 {
			conf.MaxIdleTime = 5 * 60 //默认 5 分钟
		}

		if conf.MaxIdleTime == 0 {
			// 如果未指定，使用智能默认值：MaxLifeTime的1/6，但有合理的上下限
			conf.MaxIdleTime = min(max(conf.MaxLifeTime/6, 60), 600) // 限制在1分钟-10分钟范围内
		}
		// 最大空闲时间不能大于最大生存时间
		conf.MaxIdleTime = min(conf.MaxIdleTime, conf.MaxLifeTime)

		// 配置底层sql.DB连接池参数
		if sqlDb, err = db.DB(); err != nil {
			panic(err)
		} else {
			sqlDb.SetMaxIdleConns(int(conf.Idle))                                   // 设置最大空闲连接数
			sqlDb.SetMaxOpenConns(int(conf.Open))                                   // 设置最大打开连接数
			sqlDb.SetConnMaxLifetime(time.Duration(conf.MaxLifeTime) * time.Second) // 设置连接最大生存时间
			sqlDb.SetConnMaxIdleTime(time.Duration(conf.MaxIdleTime) * time.Second) // 设置连接最大空闲时间
		}

		// 测试数据库连接是否正常
		if err := sqlDb.Ping(); err != nil {
			return nil, errors.New(fmt.Sprintf("数据库Ping出错 - [%s]", err.Error()))
		}
		log.Info("load mysql success", zap.Object("conn", conf))

		return db, nil
	}
}

// initConnect 初始化数据库连接
// 功能包括：数据库连接建立、连接池配置、分表配置、雪花ID生成器配置等
// 参数:
//   - serverName: 服务名称，用于雪花ID生成器的标识
//   - conf: 数据库配置信息，包含连接参数、分表配置、雪花ID配置等
//
// 返回值:
//   - db: GORM数据库实例
//   - err: 错误信息
func initConnect(serverName string, instance string, conf config2.DbInstanceConfig) (db *gorm.DB, err error) {

	db, err = startDb(instance, conf.DbBaseInstanceConfig)
	if err != nil {
		return
	} else {
		// 配置分表功能
		if err = initSharding(db, conf); err != nil {
			return
		}
		// 配置雪花ID生成器（前置条件：必须开启Redis配置）
		if err = initSnowflake(db, serverName, conf); err != nil {
			return
		}
		// 初始化主从
		if err = initReplica(instance, db, conf); err != nil {
			return
		}
		// 注册连接日志打印插件
		//callback.RegisterConnectionLogger(db)
		// 注册多商户插件
		callback.NewPlugin(callback.Config{
			NamespaceResolver: func(ctx context.Context) string {
				return metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE)
			},
			EnableRawSQLRewrite: true,
		}).Initialize(db)
		log.Info("初始化数据库成功", zap.String("driver", conf.Driver), zap.String("host", conf.Host), zap.String("database", conf.Database))
	}
	return
}

// 初始化从库
func initReplica(instance string, db *gorm.DB, conf config2.DbInstanceConfig) error {
	for i, v := range conf.Replicas {
		// 设置连接池参数的默认值
		if v.Idle == 0 {
			v.Idle = 20 // 默认最大空闲连接数
		}

		if v.Open == 0 {
			v.Open = 200 // 默认最大打开连接数
		}
		// 最大空闲连接数不能大于最大打开连接数
		v.Idle = min(v.Idle, v.Open)

		if v.MaxLifeTime == 0 {
			v.MaxLifeTime = 3600 // 默认一小时
		}
		if v.MaxIdleTime == 0 {
			v.MaxIdleTime = 5 * 60 //默认 5 分钟
		}
		if v.MaxIdleTime == 0 {
			// 如果未指定，使用智能默认值：MaxLifeTime的1/6，但有合理的上下限
			v.MaxIdleTime = min(max(v.MaxLifeTime/6, 60), 600) // 限制在1分钟-10分钟范围内
		}
		// 最大空闲时间不能大于最大生存时间
		v.MaxIdleTime = min(v.MaxIdleTime, v.MaxLifeTime)
		// 获取 dir
		dir, err := getDialector(fmt.Sprintf("%v-%v", instance, i), conf.DbBaseInstanceConfig)
		if err != nil {
			return err
		}
		if err = db.Use(
			dbresolver.Register(dbresolver.Config{
				Replicas: []gorm.Dialector{dir},
			}).
				SetConnMaxIdleTime(time.Duration(v.MaxIdleTime)).
				SetConnMaxLifetime(time.Duration(v.MaxLifeTime)).
				SetMaxIdleConns(int(v.Idle)).
				SetMaxOpenConns(int(v.Open)),
		); err != nil {
			return err
		}
	}
	return nil
}

// InitConnect 初始化数据库连接管理器
// 负责初始化默认数据库连接和多个数据库连接，同时设置定时任务监控连接池状态
// 参数:
//   - serverName: 服务名称，用于雪花ID生成器标识
//   - conf: 数据库配置，包含主数据库和其他数据库的配置信息
//
// 返回值:
//   - cron.CronJob: 数据库连接池监控的定时任务
//   - micro.CloseS: 服务关闭时的清理函数
//   - error: 初始化过程中的错误信息
func InitConnect(serverName string, conf config2.DbConfig) (*cron.CronJob, *micro.CloseS, error) {
	if conf.Instance == nil || len(conf.Instance) == 0 {
		return nil, nil, nil
	}
	// 创建服务关闭时的清理结构体，权重设为0表示最后执行关闭
	stopHandler := micro.CloseS{
		Weight:    0, // 权重最低，最后执行关闭
		CloseFunc: stop,
	}

	// 保存数据库配置到全局变量
	dbCfg = conf

	// 初始化其他数据库连接
	for k, v := range conf.Instance {
		// 初始化额外的数据库连接
		db, err := initConnect(serverName, k, v)
		if conf.Default == k {
			// 默认数据库
			conn = db
		}
		if err != nil {
			return nil, nil, err
		}
		// 将连接存储到多连接映射中
		mulConn.Store(k, db)
	}

	// 连接池统计模式：-1 不打印不上报，0 默认（等同1），1 仅日志打印，2 仅指标上报，3 都开启
	// 用局部变量 + 闭包捕获（与 redis.InitRedis 的写法一致），避免包级变量在 cron goroutine
	// 与二次 InitConnect 之间形成无同步的并发读写 / 状态互相覆盖
	statsLogEnabled := config2.PoolStatsLogEnabled(conf.PoolStatsMode)
	statsReportEnabled := config2.PoolStatsReportEnabled(conf.PoolStatsMode)
	if conf.PoolStatsInterval == 0 {
		conf.PoolStatsInterval = 60
	}
	var cj *cron.CronJob
	if conf.PoolStatsInterval > 0 && (statsLogEnabled || statsReportEnabled) {
		cj = &cron.CronJob{
			Name: "Database Stats check", // 任务名称
			DoJob: func() {
				mysqlStatusCheck(statsLogEnabled, statsReportEnabled)
			},
			CronSpc:         fmt.Sprintf("@every %ds", conf.PoolStatsInterval), // 每隔PoolStatsInterval 秒执行
			DoBeforeDone:    false,                                             // 服务关闭前不执行
			DelayBeforeDone: false,                                             // 服务关闭前不延迟
		}
	}

	// 返回定时任务：用于定期检查和记录数据库连接池状态
	return cj, &stopHandler, nil
}

// mysqlStatusCheck 检查所有数据库连接池状态（默认库也在 mulConn 中）。
// 两个开关由 InitConnect 以闭包形式传入，不用包级变量。
func mysqlStatusCheck(logEnabled, reportEnabled bool) {
	mulConn.Range(func(key, value any) bool {
		emitDbStats(key.(string), value.(*gorm.DB), logEnabled, reportEnabled)
		return true
	})
}

// emitDbStats 输出单个数据库连接池的统计数据：
// logEnabled 时打印日志（过滤未发生变化的统计）；reportEnabled 时推送到 pmicro/metrics（metrics 未开启则由其内部丢弃）
func emitDbStats(name string, db *gorm.DB, logEnabled, reportEnabled bool) {
	sqlDb, err := db.DB()
	if err != nil {
		log.Error("获取 Mysql 统计错误", zap.String("DB", name), zap.Error(err))
		return
	}
	stats := sqlDb.Stats()
	if logEnabled {
		st, ok := mulConnStats.Load(name)
		if !ok || st.(sql.DBStats) != stats { //过滤一下没有发生变化的日志
			log.Info("连接池信息",
				zap.String("info", name),                                // 连接池标识
				zap.Int("MaxOpenConnections", stats.MaxOpenConnections), // 最大打开连接数
				zap.Int("InUse", stats.InUse),                           // 正在使用的连接数
				zap.Int("Idle", stats.Idle),                             // 空闲连接数
				zap.Int("Open", stats.OpenConnections),                  // 当前打开的连接数
				zap.Int64("WaitCount", stats.WaitCount),                 // 等待连接的总次数
				zap.Int64("MaxIdleClosed", stats.MaxIdleClosed),         // 因超过最大空闲连接数而关闭的连接数
				zap.Int64("MaxIdleTimeClosed", stats.MaxIdleTimeClosed), // 因超过最大空闲时间而关闭的连接数
				zap.Int64("MaxLifetimeClosed", stats.MaxLifetimeClosed), // 因超过最大生存时间而关闭的连接数
				zap.String("WaitDuration", stats.WaitDuration.String()), // 等待连接的总时间
			)
			mulConnStats.Store(name, stats)
		}
	}
	if reportEnabled {
		tags := metrics.Tags{"instance": name}
		metrics.SetGauge("mysql_pool_max_open_connections", tags, float64(stats.MaxOpenConnections))
		metrics.SetGauge("mysql_pool_in_use", tags, float64(stats.InUse))
		metrics.SetGauge("mysql_pool_idle", tags, float64(stats.Idle))
		metrics.SetGauge("mysql_pool_open_connections", tags, float64(stats.OpenConnections))
		metrics.SetGauge("mysql_pool_wait_count", tags, float64(stats.WaitCount))
		metrics.SetGauge("mysql_pool_wait_duration_ms", tags, float64(stats.WaitDuration.Milliseconds()))
		metrics.SetGauge("mysql_pool_max_idle_closed", tags, float64(stats.MaxIdleClosed))
		metrics.SetGauge("mysql_pool_max_idle_time_closed", tags, float64(stats.MaxIdleTimeClosed))
		metrics.SetGauge("mysql_pool_max_lifetime_closed", tags, float64(stats.MaxLifetimeClosed))
	}
}

var isSnowflakeInit bool

// stop 停止并清理数据库连接资源
// 首先等待所有事务提交完成，然后关闭数据库连接，最后停止雪花ID生成器
// 该函数在服务关闭时被调用，确保所有数据库资源被正确释放
func stop() {
	log.Info("开始停止数据库连接，等待事务完成...")

	// 等待所有数据库连接的事务完成
	waitForTransactionsToComplete()

	// 关闭默认数据库连接
	if conn != nil {
		if db, err := conn.DB(); err == nil {
			db.Close()
		}
	}

	// 关闭所有多数据库连接
	mulConn.Range(func(key, value any) bool {
		if v := value.(*gorm.DB); v != nil {
			if db, err := v.DB(); err == nil {
				db.Close()
			}
		}
		return true
	})
	if isSnowflakeInit {
		// 最后停止雪花ID生成器，确保在数据库操作完成前ID生成器仍可用
		eonid.Stop()
	}
	log.Info("停止数据库成功")
}

// waitForTransactionsToComplete 等待所有数据库连接的事务完成
// 通过检查连接池的InUse状态来判断是否还有活跃的事务
// 设置最大等待时间为30秒，避免无限等待
func waitForTransactionsToComplete() {
	const (
		maxWaitTime     = 30 * time.Second       // 最大等待时间
		checkInterval   = 500 * time.Millisecond // 检查间隔
		acceptableInUse = 0                      // 可接受的使用中连接数
	)

	startTime := time.Now()
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		allIdle := true
		totalInUse := 0

		// 检查默认数据库连接
		if conn != nil {
			if db, err := conn.DB(); err == nil {
				stats := db.Stats()
				totalInUse += stats.InUse
				if stats.InUse > acceptableInUse {
					allIdle = false
					log.Info("等待默认数据库事务完成",
						zap.Int("InUse", stats.InUse),
						zap.Int("Open", stats.OpenConnections),
						zap.Duration("等待时间", time.Since(startTime)))
				}
			}
		}

		// 检查所有多数据库连接
		mulConn.Range(func(key, value any) bool {
			name := key.(string)
			v := value.(*gorm.DB)
			if v != nil {
				if db, err := v.DB(); err == nil {
					stats := db.Stats()
					totalInUse += stats.InUse
					if stats.InUse > acceptableInUse {
						allIdle = false
						log.Info("等待多数据库事务完成",
							zap.String("database", name),
							zap.Int("InUse", stats.InUse),
							zap.Int("Open", stats.OpenConnections),
							zap.Duration("等待时间", time.Since(startTime)))
					}
				}
			}
			return true
		})

		// 如果所有连接都空闲了，或者超时了，则退出等待
		if allIdle {
			log.Info("所有数据库事务已完成，开始关闭连接")
			break
		}

		if time.Since(startTime) >= maxWaitTime {
			log.Warn("等待数据库事务完成超时，强制关闭连接",
				zap.Duration("等待时间", time.Since(startTime)),
				zap.Int("剩余使用中连接数", totalInUse))
			break
		}

		// 等待下一次检查
		<-ticker.C
	}
}

// isBlank 判断反射值是否为空或零值
// 支持多种Go基本类型的零值判断，用于雪花ID生成时判断主键是否需要赋值
// 参数:
//   - value: reflect.Value 反射值对象
//
// 返回值:
//   - bool: 如果值为空/零值则返回true，否则返回false
func isBlank(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.String, reflect.Array:
		// 字符串和数组类型：长度为0表示空
		return value.Len() == 0
	case reflect.Bool:
		// 布尔类型：false表示空
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// 有符号整数类型：0表示空
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		// 无符号整数类型：0表示空
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		// 浮点数类型：0.0表示空
		return value.Float() == 0
	case reflect.Ptr, reflect.Interface:
		// 指针和接口类型：nil表示空
		return value.IsNil()
	}
	// 其他类型默认返回false（不为空）
	return false
}

// ShardingAlgorithm 生成分表算法函数
// 根据分表数量生成相应的分表后缀算法，支持整数和字符串类型的分表键
// 参数:
//   - shardingNum: 分表数量，用于计算表后缀
//
// 返回值:
//   - func: 分表算法函数，接收列值并返回表后缀
func ShardingAlgorithm(shardingNum uint) func(columnValue any) (suffix string, err error) {
	return func(columnValue any) (suffix string, err error) {
		// 根据分表数量确定后缀格式，保证表名有序且长度一致
		var format string
		if shardingNum <= 10 {
			format = "_%01d" // 1位数字，如_0, _1, _2...
		} else if shardingNum <= 100 {
			format = "_%02d" // 2位数字，如_00, _01, _02...
		} else if shardingNum <= 1000 {
			format = "_%03d" // 3位数字，如_000, _001, _002...
		} else if shardingNum <= 10000 {
			format = "_%04d" // 4位数字，如_0000, _0001, _0002...
		}

		id := 0
		// 根据不同的列值类型进行处理
		switch value := columnValue.(type) {
		case int:
			id = value
		case int64:
			id = int(value) // 注意：可能存在溢出风险
		case uint64:
			id = int(value) // 注意：可能存在溢出风险
		case string:
			// 尝试将字符串转换为整数
			id, err = strconv.Atoi(value)
			if err != nil {
				// 如果转换失败，使用CRC32校验和作为hash值
				id = int(crc32.ChecksumIEEE([]byte(value)))
			}
		default:
			return "", errors.New("default algorithm only support integer and string column," +
				"if you use other type, specify you own ShardingAlgorithm")
		}

		// 使用模运算确定表后缀，确保分布均匀
		return fmt.Sprintf(format, id%int(shardingNum)), nil
	}
}

// getDialector 根据配置获取对应的数据库驱动器
// 支持MySQL、PostgreSQL、OpenGauss三种数据库驱动
// 参数:
//   - conf: 数据库配置信息，包含驱动类型、连接参数等
//
// 返回值:
//   - gorm.Dialector: GORM数据库驱动器接口实现
func getDialector(instance string, conf config2.DbBaseInstanceConfig) (gorm.Dialector, error) {

	var host string
	switch conf.Driver {
	case config2.DRIVER_MYSQL: // MySQL驱动配置
		// host := "ddm_test_user:L03kV2y7WUgXVwjga%v7@tcp(192.168.64.19:5066)/ca_game_all_order?charset=utf8mb4&parseTime=True&loc=Local&useServerPrepStmts=true&interpolateParams=true"
		// 构建MySQL DSN连接字符串
		// charset=utf8mb4: 支持完整的UTF-8字符集，包括emoji
		// parseTime=True: 自动解析MySQL的TIME和DATE类型为Go的time.Time
		// loc=Local: 使用本地时区
		// interpolateParams=true: 客户端插值参数，兼容华为云DDM
		return initMysqlDialector(instance, conf)
	case config2.DRIVER_PGSQL: // PostgreSQL驱动配置
		// 构建PostgreSQL连接字符串
		// sslmode=disable: 禁用SSL连接（根据实际需求可调整）
		//host = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", conf.Host, conf.Port, conf.User, conf.Password, conf.Database, "disable")
		//return postgres.Open(host)
		return initPostgresDialector(conf)
	case config2.DRIVER_OPEN_GAUSS: // OpenGauss驱动配置
		// host = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s search_path=%s sslmode=%s", conf.Host, conf.Port, conf.User, conf.Password, conf.Database, conf.PostgresqlGroup, "disable")
		// 构建OpenGauss连接字符串（基于PostgreSQL协议）
		host = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", conf.Host, conf.Port, conf.User, conf.Password, conf.Database, "disable")
		// 使用自定义的PostgreSQL配置以兼容OpenGauss
		return postgres.New(postgres.Config{
			DriverName:       "opengauss", // 指定使用OpenGauss驱动
			DSN:              host,        // 数据源名称
			WithoutReturning: true,        // OpenGauss不支持RETURNING子句
			//WithoutLastInsert: true,        // OpenGauss不支持LastInsertId
		}), nil
	default:
		// 不支持的数据库驱动类型，抛出panic
		panic("不支持的database驱动类型")
	}
}

// 初始化 mysql 驱动
func initMysqlDialector(instance string, conf config2.DbBaseInstanceConfig) (gorm.Dialector, error) {
	var fmtStr = "%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&interpolateParams=true"
	// 判断是否开启 ssl
	if conf.SSLConfig != nil && conf.SSLConfig.Enable {
		// 构建 CA
		rootPool := x509.NewCertPool()
		pem, err := os.ReadFile(conf.SSLConfig.Path)
		if err != nil {
			return nil, err
		}
		rootPool.AppendCertsFromPEM(pem)

		tlsConfig := &tls.Config{
			RootCAs:    rootPool,
			MinVersion: tls.VersionTLS12,
		}
		// 名称设置为实例名称
		if err = mysqlBase.RegisterTLSConfig(instance, tlsConfig); err != nil {
			return nil, err
		}

		fmtStr += fmt.Sprintf("&tls=%v", instance)
	}
	if len(conf.Port) > 0 {
		conf.Host += ":" + conf.Port
	}
	//dsn := "ddm_test_user:L03kV2y7WUgXVwjga%v7@tcp(192.168.64.19:5066)/ca_game_all_order?charset=utf8mb4&parseTime=True&loc=Local&useServerPrepStmts=true&interpolateParams=true"
	host := fmt.Sprintf(fmtStr, conf.User, conf.Password, conf.Host, conf.Database)
	return mysql.Open(host), nil
}

func initPostgresDialector(conf config2.DbBaseInstanceConfig) (gorm.Dialector, error) {
	var host string
	// 判断是否开启 ssl
	if conf.SSLConfig != nil && conf.SSLConfig.Enable {
		host = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s sslrootcert=%s", conf.Host, conf.Port, conf.User, conf.Password, conf.Database, "verify-full", conf.SSLConfig.Path)
	} else {
		host = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", conf.Host, conf.Port, conf.User, conf.Password, conf.Database, "disable")
	}
	return postgres.Open(host), nil
}
