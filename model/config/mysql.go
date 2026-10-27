/**
 * @Author: yinwei
 * @Description: 数据库配置文件
 * @File: mysql
 * @Version: 1.0.0
 * @Date: 2023/10/24 16:14
 */
package config

import (
	"github.com/coderyw/gomicro-util/util"
	"go.uber.org/zap/zapcore"
)

const (
	// mysql驱动
	DRIVER_MYSQL = "mysql"
	// postgresql驱动
	DRIVER_PGSQL = "postgresql"
	// 华为云gauss db驱动
	DRIVER_OPEN_GAUSS = "opengauss"
)

// DbConfig 数据库配置
/**
  default: 'core'
  instance:
    core:
      # 连接地址
      host: 'mysql.example.com'
      # 连接端口
      port: '3306'
      # 用户名
      user: datamanager
      # 密码
      password: 'password'
      # 默认数据库名
      database: 'db_game_color'
      # 是否开启debug模式（目前默认都是开启的）
      debug: true
      # 数据库驱动，支持 mysql/postgresql/opengauss 默认 msyql
      driver: 'mysql'
      # 最大连接数， -1 表示不限制，默认 200
      open: 100
      # 最大空闲连接数，默认 50
      idle: 20
      # 最大空闲时间(单位 s):设置可以重复使用连接的最长时间,过期的连接可能会在重新使用前延迟关闭。如果<=0,则连接不会因连接的年龄而关闭。 默认 20s
      max_life_time: 3600
      # ssl配置
      ssl_config:
        enable: true
        #  证书位置 如果是本地运行，证书需要放在本地。如果是云上运行，推荐证书放在k8s 的K8s Secret
        #  kubectl create secret generic rds-ca-secret --from-file=global-bundle.pem //创建证书
        #  或者 yaml 方式
        #apiVersion: v1
        #kind: Secret
        #metadata:
        #  name: rds-ca-secret
        #type: Opaque
        #data:
        #  global-bundle.pem: <base64内容>
        # 然后 deployment 挂载证书
        #apiVersion: apps/v1
        #kind: Deployment
        #metadata:
        #  name: my-app
        #spec:
        #  template:
        #    spec:
        #      containers:
        #        - name: app
        #          image: my-app:latest
        #          volumeMounts:
        #            - name: rds-ca
        #              mountPath: /etc/ssl/rds
        #              readOnly: true
        #      volumes:
        #        - name: rds-ca
        #          secret:
        #            secretName: rds-ca-secret
        path: '/etc/ssl/global-bundle.pem'
      # 如果需要配置分表，则需要下面的配置
      sharding:
        - table_name: 'b_color_order' # 分表名
          sharding_key: 'sys_user_id' # 分表键
          sharding_num: 100 # 分表数量
      # 雪花id生产规则配置（当前是全局配置）
      snowflake_gen:
        #  是否开启默认的数据中心(如果开启，没有配置数据中心的表都会使用这个数据中心) 默认 true
        enable_default: true
        # 雪花中心类型 0 表名级别 1 数据库名为雪花中心（中心名称为库名） 2 服务级别
        snowflake_type: 0
        snowflakes:
          - worker_id_bit: 10
            data_center: 'ca_user'
      # 从库配置
      replicas:
        - host: 'mysql-replica.example.com'
          port: '3306'
          user: datamanager1
          password: 'password'
          database: 'db_game_color'
          debug: true
          driver: 'mysql'
          open: 100
          idle: 20
          max_life_time: 3600
          ssl_config:
            enable: true
            path: '/etc/ssl/global-bundle.pem'
          sharding:
            - table_name: 'b_color_order' # 分表名
              sharding_key: 'sys_user_id' # 分表键
              sharding_num: 100 # 分表数量
    # 游戏库配置
    game:
      host: 'mysql.example.com'
      port: '3306'
      user: datamanager
      password: 'password'
      database: 'db_game_color'
      debug: true
      driver: 'mysql'
      open: 100
      idle: 20
      max_life_time: 3600
      ssl_config:
        enable: true
        path: '/etc/ssl/global-bundle.pem'
      sharding:
        - table_name: 'b_color_order' # 分表名
          sharding_key: 'sys_user_id' # 分表键
          sharding_num: 100 # 分表数量
      snowflake_gen:
        enable_default: true
        snowflake_type: 0
        snowflakes:
          - worker_id_bit: 10
            data_center: 'ca_user'
      replicas:
        - host: 'mysql-replica.example.com'
          port: '3306'
          user: datamanager1
          password: 'password'
          database: 'db_game_color'
          debug: true
          driver: 'mysql'
          open: 100
          idle: 20
          max_life_time: 3600
          ssl_config:
            enable: true
            path: '/etc/ssl/global-bundle.pem'
          sharding:
            - table_name: 'b_color_order' # 分表名
              sharding_key: 'sys_user_id' # 分表键
              sharding_num: 100 # 分表数量

*/
type DbConfig struct {
	// 默认数据库实例
	Default string `json:"default" yaml:"default"`
	// 连接池统计模式：-1 不打印不上报，0 默认（等同1），1 仅日志打印，2 仅指标上报，3 日志和上报都开启。默认 1
	PoolStatsMode int `json:"pool_stats_mode" yaml:"pool_stats_mode"`
	// 链接池统计打印间隔。默认 60s 一次。单位 s。-1标识不开启
	PoolStatsInterval int                         `json:"pool_stats_interval" yaml:"pool_stats_interval"`
	Instance          map[string]DbInstanceConfig `json:"instance" yaml:"instance"`
}

func (s DbConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("default", s.Default)
	enc.AddInt("pool_stats_mode", s.PoolStatsMode)
	enc.AddInt("pool_stats_interval", s.PoolStatsInterval)
	enc.AddObject("instance", zapcore.ObjectMarshalerFunc(func(encoder zapcore.ObjectEncoder) error {
		for k, v := range s.Instance {
			encoder.AddObject(k, v)
		}
		return nil
	}))
	return nil
}

type DbInstanceConfig struct {
	// 主库基础配置
	DbBaseInstanceConfig `yaml:",inline"`
	// 分表配置，用于配置数据库分表策略
	Sharding []ShardingTable `json:"sharding" yaml:"sharding"`
	// 雪花id配置
	SnowflakeGen *SnowflakeGen `json:"snowflake_gen" yaml:"snowflake_gen"`
	// 从库配置
	Replicas []DbReplicas `json:"replicas" yaml:"replicas"`
}

func (s DbInstanceConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("host", s.Host)
	enc.AddString("port", s.Port)
	enc.AddString("user", s.User)
	enc.AddString("password", util.Mask(s.Password, 2, 3))
	enc.AddString("database", s.Database)
	enc.AddBool("debug", s.Debug)
	enc.AddInt64("open", s.Open)
	enc.AddInt64("idle", s.Idle)
	enc.AddInt64("max_life_time", s.MaxLifeTime)
	enc.AddInt64("max_idle_time", s.MaxIdleTime)
	enc.AddString("driver", s.Driver)
	enc.AddArray("sharding", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.Sharding {
			encoder.AppendObject(v)
		}
		return nil
	}))
	if s.SSLConfig != nil {
		if err := enc.AddObject("ssl_config", s.SSLConfig); err != nil {
			return err
		}
	}
	enc.AddArray("replicas", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.Replicas {
			encoder.AppendObject(v)
		}
		return nil
	}))
	if s.SnowflakeGen != nil {
		if err := enc.AddObject("snowflake_gen", s.SnowflakeGen); err != nil {
			return err
		}
	}
	return nil
}

// 数据库从库配置
type DbReplicas struct {
	DbBaseInstanceConfig `yaml:",inline"`
	// 分表配置，用于配置数据库分表策略
	Sharding []ShardingTable `json:"sharding" yaml:"sharding"`
}

func (s DbReplicas) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("host", s.Host)
	enc.AddString("port", s.Port)
	enc.AddString("user", s.User)
	enc.AddString("password", util.Mask(s.Password, 2, 3))
	enc.AddString("database", s.Database)
	enc.AddBool("debug", s.Debug)
	enc.AddInt64("open", s.Open)
	enc.AddInt64("idle", s.Idle)
	enc.AddInt64("max_life_time", s.MaxLifeTime)
	enc.AddInt64("max_idle_time", s.MaxIdleTime)
	enc.AddString("driver", s.Driver)
	enc.AddArray("sharding", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.Sharding {
			encoder.AppendObject(v)
		}
		return nil
	}))
	if s.SSLConfig != nil {
		if err := enc.AddObject("ssl_config", s.SSLConfig); err != nil {
			return err
		}
	}
	return nil
}

type DbBaseInstanceConfig struct {
	// 数据库主机地址
	Host string `json:"host" yaml:"host"`
	// 数据库端口号
	Port string `json:"port" yaml:"port"`
	// 数据库用户名
	User string `json:"user" yaml:"user"`
	// 数据库密码
	Password string `json:"password" yaml:"password"`
	// 数据库名称
	Database string `json:"database" yaml:"database"`
	// 是否开启debug模式，开启后会打印SQL日志
	Debug bool `json:"debug" yaml:"debug"`
	// 最大打开连接数，即同时活跃的连接数上限，0表示无限制
	Open int64 `json:"open" yaml:"open"`
	// 最大空闲连接数，即连接池中保持的空闲连接数量
	// 建议设置为Open的10%-20%，避免频繁创建/销毁连接
	Idle int64 `json:"idle" yaml:"idle"`
	// 连接最大生存时间（秒）
	// 连接从创建到强制关闭的最长时间，无论连接是否在使用
	// 用于防止连接老化、适应数据库服务器重启等场景
	// 建议设置为几小时，如3600秒(1小时)
	// 如果值<=0，则连接不会因为存活时间而被关闭
	// 最大空闲时间
	//SetConnMaxLifetime设置可以重复使用连接的最长时间。
	//
	//过期的连接可能会在重新使用前延迟关闭。
	//
	//如果d<=0，则连接不会因连接的年龄而关闭。
	// 默认 1 小时
	MaxLifeTime int64 `json:"max_life_time" yaml:"max_life_time"`
	// 连接最大空闲时间（秒）
	// 超过此时间的空闲连接将被关闭以释放资源
	// 只影响空闲状态的连接，正在使用的连接不受影响
	// 建议设置为几分钟到十几分钟，如300秒(5分钟)
	// 如果值<=0，则空闲连接不会因为空闲时间而被关闭
	MaxIdleTime int64 `json:"max_idle_time" yaml:"max_idle_time"`
	// 数据库驱动类型，支持：mysql, opengauss, postgresql
	Driver    string     `json:"driver" yaml:"driver"`
	SSLConfig *SSLConfig `json:"ssl_config" yaml:"ssl_config"`
}

func (s DbBaseInstanceConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("host", s.Host)
	enc.AddString("port", s.Port)
	enc.AddString("user", s.User)
	enc.AddString("password", util.Mask(s.Password, 2, 3))
	enc.AddString("database", s.Database)
	enc.AddBool("debug", s.Debug)
	enc.AddInt64("open", s.Open)
	enc.AddInt64("idle", s.Idle)
	enc.AddInt64("max_life_time", s.MaxLifeTime)
	enc.AddInt64("max_idle_time", s.MaxIdleTime)
	enc.AddString("driver", s.Driver)
	if s.SSLConfig != nil {
		if err := enc.AddObject("ssl_config", s.SSLConfig); err != nil {
			return err
		}
	}
	return nil
}

type SSLConfig struct {
	// 是否开启
	Enable bool `json:"enable" yaml:"enable"`
	// 证书位置 如果是本地运行，证书需要放在本地。如果是云上运行，推荐证书放在k8s 的K8s Secret
	// kubectl create secret generic rds-ca-secret --from-file=global-bundle.pem //创建证书
	// 或者 yaml 方式
	//apiVersion: v1
	//kind: Secret
	//metadata:
	//  name: rds-ca-secret
	//type: Opaque
	//data:
	//  global-bundle.pem: <base64内容>
	// 然后 deployment 挂载证书
	//apiVersion: apps/v1
	//kind: Deployment
	//metadata:
	//  name: my-app
	//spec:
	//  template:
	//    spec:
	//      containers:
	//        - name: app
	//          image: my-app:latest
	//          volumeMounts:
	//            - name: rds-ca
	//              mountPath: /etc/ssl/rds
	//              readOnly: true
	//      volumes:
	//        - name: rds-ca
	//          secret:
	//            secretName: rds-ca-secret
	Path string `json:"path" yaml:"path"`
}

func (s SSLConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddBool("enable", s.Enable)
	enc.AddString("path", s.Path)
	return nil
}

// 雪花id配置
type SnowflakeGen struct {
	// 是否开启默认的数据中心(如果开启，没有配置数据中心的表都会使用这个数据中心) 默认 true
	EnableDefault *bool `json:"enable_default" yaml:"enable_default"`
	// 雪花中心类型 0 表名级别 1 数据库名为雪花中心（中心名称为库名） 2 服务级别
	SnowflakeType uint8 `json:"snowflake_type" yaml:"snowflake_type"`
	// 是否使用雪花算法生成主键id,开启需要依赖redis
	// key=数据中心名称，value=雪花算法配置，支持多个表共用一个数据中心
	Snowflakes []Snowflake `yaml:"snowflakes" json:"snowflakes"`
}

func (s *SnowflakeGen) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if s.EnableDefault != nil {
		enc.AddBool("enable_default", *s.EnableDefault)
	}
	enc.AddArray("snowflakes", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.Snowflakes {
			encoder.AppendObject(v)
		}
		return nil
	}))
	return nil
}

// 雪花算法配置
type Snowflake struct {
	// 机器id位数，默认10,取值范围[1, 15]
	WorkerIdBit uint8 `json:"worker_id_bit" yaml:"worker_id_bit"`
	// data center
	DataCenter string `json:"data_center"  yaml:"data_center"`
}

func (s Snowflake) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("data_center", s.DataCenter)
	enc.AddUint8("worker_id_bit", s.WorkerIdBit)
	//enc.AddArray("tables", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
	//	for _, v := range s.Tables {
	//		encoder.AppendString(v)
	//	}
	//	return nil
	//}))

	return nil
}

type ShardingTable struct {
	// 表名
	TableName string `json:"table_name" yaml:"table_name"`
	// 分表key
	ShardingKey string `json:"sharding_key" yaml:"sharding_key"`
	// 分表数量
	ShardingNum uint `json:"sharding_num" yaml:"sharding_num"`
	// 分表主键生成策略 0 雪花算法(依赖于snowflake_gen这个配置) 1postgresql主键生成策略 2 mysql主键生成策略 3自增主键
	// 雪花id依赖redis
	PKGenerator uint `json:"p_k_generator" yaml:"p_k_generator"`
}

func (s ShardingTable) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("table_name", s.TableName)
	enc.AddString("sharding_key", s.ShardingKey)
	enc.AddUint("sharding_num", s.ShardingNum)
	enc.AddUint("p_k_generator", s.PKGenerator)
	return nil
}
