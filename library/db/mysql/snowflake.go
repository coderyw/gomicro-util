package mysql

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/coderyw/goeonid/v2/eonid"
	"github.com/coderyw/goeonid/v2/eonid/option"
	"github.com/coderyw/gomicro-util/library/db/cache/redis"
	"github.com/coderyw/gomicro-util/library/env"
	config2 "github.com/coderyw/gomicro-util/model/config"
	"gorm.io/gorm"
)

const (
	snowflakeTypeTable    = 0
	snowflakeTypeDatabase = 1
	snowflakeTypeServer   = 2
)

var gotDefaultSnowflake = false

// 初始化雪花 id
func initSnowflake(db *gorm.DB, serverName string, conf config2.DbInstanceConfig) (err error) {
	centerFilter := make(map[string]struct{})
	var t uint8
	if conf.SnowflakeGen != nil {
		t = conf.SnowflakeGen.SnowflakeType
		// 初始化所有配置的雪花ID数据中心
		for _, v := range conf.SnowflakeGen.Snowflakes {
			// 避免重复初始化同一个数据中心
			if _, ok := centerFilter[v.DataCenter]; ok {
				return errors.Errorf("存在重复的数据中心 %v", v.DataCenter)
			}
			centerFilter[v.DataCenter] = struct{}{}
			// 开始初始化
			if err := initSnowflakeInstance(serverName, v, 2); err != nil {
				return err
			}
		}
		// 默认数据中心只初始化一个
		if !gotDefaultSnowflake && (conf.SnowflakeGen.EnableDefault == nil || *conf.SnowflakeGen.EnableDefault) {
			// 确认是否已经初始化过默认的
			// 初始化默认数据中心
			if err := initSnowflakeInstance(serverName, config2.Snowflake{
				WorkerIdBit: 10, //默认的使用 10
				DataCenter:  "",
			}, 1); err != nil {
				return err
			}
			gotDefaultSnowflake = true
		}
	}
	// 不管有没有注册雪花 id，都要注册回调插件
	if err = db.Callback().Create().Before("gorm:create").Register("snowflake:before_create", snowflake(serverName, centerFilter, t)); err != nil {
		return errors.WithStack(err)
	}
	return err
}

// 初始化雪花 id 实例
func initSnowflakeInstance(serverName string, v config2.Snowflake, isDefault uint8) (err error) {
	disableHashTag, err := env.GetDisableEonidHashTag()
	if err != nil {
		return errors.WithStack(err)
	}
	// 创建ID生成器选项
	op := option.NewDefaultIdGeneratorOptions(serverName, v.DataCenter, 0)
	op.DisableHashTag = disableHashTag
	if v.WorkerIdBit > 0 {
		op.WorkerIdBitLength = v.WorkerIdBit // 设置WorkerID位长度
	}
	op.DefaultCenter = isDefault // 设置是否为默认数据中心

	// 初始化ID生成器，需要Redis客户端支持
	if err = eonid.InitIdGenerator(op, redis.SnowflakeOrDefault().Client()); err != nil {
		return errors.WithStack(err)
	}
	isSnowflakeInit = true
	return nil
}

func snowflake(serverName string, tbs map[string]struct{}, t uint8) func(db *gorm.DB) {

	return func(db *gorm.DB) {
		// 要求必须传入实体model，
		if db.Statement.Schema != nil {

			if len(db.Statement.Schema.PrimaryFields) == 0 {
				//db.Error = fmt.Errorf("找不到主键id")
				// 主键 id 不存在，则直接返回走默认的逻辑
				return
			}
			var center []string
			if len(tbs) != 0 { //存在非默认的
				var key string
				// 根据配置查找是否配置了这个表的雪花id
				switch t {
				case snowflakeTypeDatabase: // 库名基础
					arr := strings.SplitN(db.Statement.Schema.Table, ".", 2)
					if len(arr) <= 1 {
						db.Error = fmt.Errorf("表名格式错误：%v", db.Statement.Schema.Table)
						return
					}
					key = arr[0]
				case snowflakeTypeServer:
					key = serverName
				default: //默认使用表名
					key = db.Statement.Table
				}

				_, ok := tbs[key]
				if ok {
					center = []string{key}
				}
			}
			if len(center) == 0 && !gotDefaultSnowflake {
				//没有注册雪花 id
				return
			}
			snowflakeId(db, center)
		} else {
			db.Error = fmt.Errorf("gorm的schema为空")
			return
		}
	}
}

func snowflakeId(db *gorm.DB, center []string) {
	// 约定只能有一个主键，并且主键注释一定要要添加
	for _, field := range db.Statement.Schema.PrimaryFields {
		switch db.Statement.ReflectValue.Kind() { // 根据要创建的实体类型判断处理
		case reflect.Slice, reflect.Array:
			// 遍历切片或则数组
			for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
				rv := db.Statement.ReflectValue.Index(i)
				rv = reflect.Indirect(rv)
				if _, ok := field.ValueOf(db.Statement.Context, rv); !ok {
					continue
				}
				// 根据主键id类型，生成不同类型的id
				var id interface{} = eonid.NextId(center...)
				//  使用雪花id
				if err := field.Set(db.Statement.Context, rv, id); err != nil {
					db.Error = err
					return
				}
			}
		case reflect.Struct:
			rv := reflect.Indirect(db.Statement.ReflectValue)
			if _, ok := field.ValueOf(db.Statement.Context, rv); !ok {
				continue
			}
			// 根据主键id类型，生成不同类型的id
			var id interface{} = eonid.NextId(center...)
			//  使用雪花id
			if err := field.Set(db.Statement.Context, rv, id); err != nil {
				db.Error = err
				return
			}
		default:
			// 如果非上述两种类型，需要在判断是否是map类型
			switch values := db.Statement.Dest.(type) {
			case map[string]interface{}:
				if val, ok := values[field.DBName]; !ok || val == nil {
					values[field.DBName] = eonid.NextId(center...)
				}

			case *map[string]interface{}:
				if val, ok := (*values)[field.DBName]; !ok || val == nil {
					(*values)[field.DBName] = eonid.NextId(center...)
				}
				//(*values)[field.DBName] = eonid.NextId(center...)
			case []map[string]interface{}, *[]map[string]interface{}:
				mapValues, ok := values.([]map[string]interface{})
				if !ok {
					if v, ok := values.(*[]map[string]interface{}); ok {
						if *v != nil {
							mapValues = *v
						}
					}
				}
				for _, mapValue := range mapValues {
					if mapValue != nil {
						if val, ok := mapValue[field.DBName]; !ok || val == nil {
							mapValue[field.DBName] = eonid.NextId(center...)
						}
						//mapValue[field.DBName] = eonid.NextId(center...)
					}
				}
			default:
				db.Error = fmt.Errorf("未知的实体类型：%v", db.Statement.Dest)
				return
			}
		}
	}
}
