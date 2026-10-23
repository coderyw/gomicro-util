// Package option
// @Author: yinwei
// @File: options
// @Version: 1.0.0
// @Date: 2024/7/2 14:28

package option

import (
	"github.com/gin-gonic/gin"
	"github.com/coderyw/gomicro-util/library/mq/model"
	"github.com/coderyw/gomicro-util/model/cron"
	"go-micro.dev/v4"
	"go-micro.dev/v4/selector"
)

type Option func(*Options)

type Flag struct {
	Name  string
	Value string
	Usage string
}

type Options struct {
	// service名称
	ServerName string
	// 配置文件地址，默认etc/cfg.yaml 不推荐线上使用配置文件
	ConfigFile string //配置文件地址
	// 注册微服务处理函数
	RegisterHandlerFunc func(service micro.Service) error
	// 程序初始化完成后，micro运行之前执行
	BeforeStartFunc func() error
	// micro运行之后执行
	AfterFunc func() error
	// 程序停止前执行
	BeforeStopFunc func() error
	// 默认的自定义配置
	CustomDefaultCfg any
	// 自定义配置变动处理方法
	CustomCfgChangeFunc func(cfy any)
	// 消息队列 kafka
	MQKafkaHandler []model.Handler
	// rabbit mq
	MQRabbitHandler map[string]model.HandlerFunc

	// 注册http处理函数
	RegisterHttpHandler func(*gin.Engine)
	// 是否使用默认的信息返回处理
	HttpHandlerDefaultResp bool

	// 服务选择器，默认轮询，如果想自定义请传值
	Selector func(opts ...selector.Option) selector.Selector
	// cron定时任务
	CronJob []cron.CronJob
	// micro的option
	Flags []Flag
}

// Flags 使用自定义flag
//
// Parameters:
//   - opts: ...Flag
//
// Returns:
//   - Option
func Flags(opts ...Flag) Option {
	return func(o *Options) {
		o.Flags = append(o.Flags, opts...)
	}
}

// ServiceName 设置服务名称
//
// Parameters:
//   - name: string 服务名
//
// Returns:
//   - Option
func ServiceName(name string) Option {
	return func(options *Options) {
		options.ServerName = name
	}
}

// ConfigFile 设置配置文件路径
//
// 默认在项目根目录下 etc/cfg.yaml
//
// Parameters:
//   - file: string 文件路径
//
// Returns:
//   - Option
func ConfigFile(file string) Option {
	return func(options *Options) {
		options.ConfigFile = file
	}
}

// RegisterHandlerFunc 注册微服务处理函数
//
// Parameters:
//   - f func(service micro.Service) error
//
// Returns：
//   - Option
func RegisterHandlerFunc(f func(service micro.Service) error) Option {
	return func(o *Options) {
		o.RegisterHandlerFunc = f
	}
}

// BeforeStartFunc 程序初始化后，micro 运行之前执行
//
// 初始化后表明是在配置加载完成和 mysql初始化完成 以及 redis初始化完成 之后。micro 运行之前是 grpc 和 http 还未开启监听之前
//
// Parameters:
//   - f func() error 执行函数
//
// Returns：
//   - Option
func BeforeStartFunc(f func() error) Option {
	return func(o *Options) {
		o.BeforeStartFunc = f
	}
}

// AfterFunc micro运行之后执行
//
// Parameters:
//   - f func() error
//
// Returns：
//   - Option
func AfterFunc(f func() error) Option {
	return func(o *Options) {
		o.AfterFunc = f
	}
}

// BeforeStopFunc 程序停止前执行
//
// Parameters:
//   - f func() error
//
// Returns：
//   - Option
func BeforeStopFunc(f func() error) Option {
	return func(o *Options) {
		o.BeforeStopFunc = f
	}
}

// CustomDefaultCfg 配置自定义配置的默认配置以及当自定义配置变动时候的处理函数
//
// Parameters:
//   - cfg any 默认配置
//   - cfgChangeFunc func(cfy any) 配置变动处理函数
//
// Returns：
//   - Option
func CustomDefaultCfg(cfg any, cfgChangeFunc func(cfy any)) Option {
	return func(options *Options) {
		options.CustomDefaultCfg = cfg
		options.CustomCfgChangeFunc = cfgChangeFunc
	}
}

// MQHandler 注册 MQ 消息处理函数
//
// 注册的
//
// Parameters:
//   - consumerInstance string 消费实例名
//   - topic string 主题
//   - handlerFunc mq.HandlerFunc 处理函数
//   - groupId ...string 消费组 id，默认服务名
//
// Returns：
//   - Option 新的 option
func MQHandler(consumerInstance, topic string, handlerFunc model.HandlerFunc, groupId ...string) Option {
	return func(o *Options) {

		//if len(rabbit) > 0 && rabbit[0] {
		//	if o.MQRabbitHandler == nil {
		//		o.MQRabbitHandler = make(map[string]mq.HandlerFunc)
		//	}
		//	o.MQRabbitHandler[topic] = handlerFunc
		//} else {
		if o.MQKafkaHandler == nil {
			o.MQKafkaHandler = make([]model.Handler, 0)
		}
		var gi string
		if len(groupId) != 0 {
			gi = groupId[0]
		}
		o.MQKafkaHandler = append(o.MQKafkaHandler, model.Handler{
			InstanceName: consumerInstance,
			GroupId:      gi,
			HandlerFunc:  handlerFunc,
			Topics:       topic,
		})
		//}
	}
}

// MQMulHandler MQ批量消费注册函数
//
// Parameters:
//   - consumerInstance string
//   - topic string
//   - handlerFunc mq.HandlerFuncs
//   - groupId ...string
//
// Returns：
//   - Option
func MQMulHandler(consumerInstance, topic string, handlerFunc model.HandlerFuncs, groupId ...string) Option {
	return func(o *Options) {

		var gi string
		if len(groupId) != 0 {
			gi = groupId[0]
		}
		if o.MQKafkaHandler == nil {
			o.MQKafkaHandler = make([]model.Handler, 0)
		}
		o.MQKafkaHandler = append(o.MQKafkaHandler, model.Handler{
			InstanceName: consumerInstance,
			GroupId:      gi,
			HandlerFuncs: handlerFunc,
			Topics:       topic,
		})
	}
}

// 使用新的注册方法
func RegisterHttpHandler(f func(*gin.Engine), useDefaultResp ...bool) Option {
	return func(o *Options) {
		o.RegisterHttpHandler = f
		if len(useDefaultResp) > 0 {
			o.HttpHandlerDefaultResp = useDefaultResp[0]
		}
	}
}

func Selector(f func(opts ...selector.Option) selector.Selector) Option {
	return func(o *Options) {
		o.Selector = f
	}
}

func Cron(jobs ...cron.CronJob) Option {
	return func(o *Options) {
		o.CronJob = append(o.CronJob, jobs...)
	}
}
