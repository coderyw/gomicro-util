package config

import (
	"github.com/coderyw/gomicro-util/util"
	"go.uber.org/zap/zapcore"
)

// 服务配置
/**

 */

var DefaultServerConfig = ServerConfig{
	Http: HttpCfg{
		Host: ":8080",
	},
	Storage: StorageCfg{
		Mysql: DbConfig{},
		Redis: RedisConfig{},
	},
}

type ServerConfig struct {
	// 平台名称
	Platform string `json:"platform" yaml:"platform"`
	// token签名盐值
	TokenSalt string `json:"token_salt" yaml:"token_salt"`
	// 服务名称 废弃
	ServerName string `yaml:"-" json:"-"`
	// 运行环境
	// local = 本地运行环境 dev = 测试运行环境 prod = 预发环境 res =线上环境
	ENVIRON string `yaml:"environ" json:"environ"`

	// 服务版本 废弃 使用环境变量
	Version string `yaml:"-" json:"-"`

	// 存储db配置
	Storage StorageCfg `yaml:"storage" json:"storage"`

	// 消息队列rabbit配置
	Rabbitmq RabbitMqCfg `json:"rabbitmq" yaml:"rabbitmq"` //是否启用rabbitmq

	// 消息队列kafka配置
	KafkaCfg KafkaCfg `json:"kafka" yaml:"kafka"`

	// http服务配置
	Http HttpCfg `yaml:"http" json:"http"`

	//注册中心配置
	Registry RegistryCfg `yaml:"registry" json:"registry"`

	//rpc服务配置
	RpcServer RpcServerCfg `yaml:"rpc_server" json:"rpc_server"`

	//日志相关配置
	Log LogCfg `yaml:"log" json:"log"`

	LarkCfg LarkCfg `json:"lark_cfg" yaml:"lark_cfg"`

	// 监控指标上报配置
	Metrics MetricsCfg `json:"metrics" yaml:"metrics"`

	//每个服务自定义配置
	Other interface{} `json:"other" yaml:"other"`
}

func (s ServerConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("server_name", s.ServerName)
	enc.AddString("environ", s.ENVIRON)
	enc.AddString("version", s.Version)
	if err := enc.AddObject("storage", s.Storage); err != nil {
		return err
	}
	if err := enc.AddObject("rabbitmq", s.Rabbitmq); err != nil {
		return err
	}
	if err := enc.AddObject("kafka", s.KafkaCfg); err != nil {
		return err
	}
	if err := enc.AddObject("http", s.Http); err != nil {
		return err
	}
	if err := enc.AddObject("registry", s.Registry); err != nil {
		return err
	}
	if err := enc.AddObject("rpc_server", s.RpcServer); err != nil {
		return err
	}
	if err := enc.AddObject("log", s.Log); err != nil {
		return err
	}
	if err := enc.AddObject("lark_cfg", s.LarkCfg); err != nil {
		return err
	}
	if s.Other != nil {
		if val, ok := s.Other.(zapcore.ObjectMarshaler); ok {
			if err := enc.AddObject("other", val); err != nil {
				return err
			}
		}
	}

	return nil
}

// http运行配置
type HttpCfg struct {
	// http监听端口
	Host string `yaml:"host" json:"host"`
	// 普罗米修斯调,不配置默认不开启性能监控
	EnablePrometheus bool `yaml:"enable_prometheus" json:"enable_prometheus"`

	CorsHeaders []string `yaml:"cors_headers" json:"cors_headers"`
	// 请求日志打印的 header 配置
	// 不配置则不打印 header；配置 * 打印所有 header；否则配置需要打印的 header 名，多个用 , 拼接，如 X-Request-Id,User-Agent
	LogHeaders string `yaml:"log_headers" json:"log_headers"`
	JwtSecret  string `json:"jwt_secret" yaml:"jwt_secret"`
	// jwt 过期时间,默认 14天
	JwtExpTime int64 `json:"jwt_exp_time" yaml:"jwt_exp_time"`
	// http server 配置
	HttpServerCfg HttpServer  `json:"http_server_cfg" yaml:"http_server_cfg"`
	Liveness      LivenessCfg `json:"liveness" yaml:"liveness"`
}

// 探活相关配置
type LivenessCfg struct {
	// 不配置则不启动探活服务
	Host string `json:"host" yaml:"host"`
	// 探活密钥(可以不配置)
	Secret string `json:"secret" yaml:"secret"`
}

func (s LivenessCfg) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("host", s.Host)
	return nil
}

type HttpServer struct {
	// 读超时 单位秒，默认 0，表示不超时
	//ReadTimeout 是读取整个请求（包括请求体）的最长时间。零或负值表示没有超时。
	//
	//由于 ReadTimeout 不允许处理程序针对每个请求的请求体可接受的截止时间或上传速率做出决策，因此大多数用户会更倾向于使用 ReadHeaderTimeout。两者可以同时使用。
	ReadTimeoutS int `json:"read_timeout_s" yaml:"read_timeout_s"`
	//ReadHeaderTimeout 是读取请求头的允许时间。连接读取请求头的截止时间会在读取完毕后重置，
	//
	//此时处理程序可以判断读取请求体的时间是否过长。
	//
	//如果 ReadHeaderTimeout 为零，则使用 ReadTimeout 的值。
	//
	//如果 ReadHeaderTimeout 为负数，或者 ReadHeaderTimeout 为零且 ReadTimeout 为零或负数，则没有超时限制。
	ReadHeaderTimeoutS int `json:"read_header_timeout_s" yaml:"read_header_timeout_s"`
	//WriteTimeout 是写入整个响应（包括响应头）的最长时间。零或负值表示没有超时时间。
	// WriteTimeout 是响应写入超时前的最大持续时间。每当读取到新请求的头部时，它都会被重置。与 ReadTimeout 类似，它不允许处理程序针对每个请求做出决策。
	//
	//零或负值表示没有超时。
	WriteTimeoutS int `json:"write_timeout_s" yaml:"write_timeout_s"`
	// IdleTimeout 是启用 keep-alive 机制时等待下一个请求的最长时间。
	// 如果为零，则使用 ReadTimeout 的值。如果为负数，或者如果 IdleTimeout 为零且 ReadTimeout 为零或负数，则没有超时。
	IdleTimeoutS int `json:"idle_timeout_s" yaml:"idle_timeout_s"`
	// MaxHeaderBytes 是请求头最大字节数。如果为零，则使用默认值 1<<20。
	// MaxHeaderBytes 控制服务器解析请求头键值对（包括请求行）时读取的最大字节数。它不限制请求体的大小。如果为零，则使用 http.DefaultMaxHeaderBytes。
	MaxHeaderBytes int `json:"max_header_bytes" yaml:"max_header_bytes"`
}

func (s HttpServer) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt("read_timeout_s", s.ReadTimeoutS)
	enc.AddInt("read_header_timeout_s", s.ReadHeaderTimeoutS)
	enc.AddInt("write_timeout_s", s.WriteTimeoutS)
	enc.AddInt("idle_timeout_s", s.IdleTimeoutS)
	enc.AddInt("max_header_bytes", s.MaxHeaderBytes)
	return nil
}

func (s HttpCfg) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("host", s.Host)
	enc.AddBool("enable_prometheus", s.EnablePrometheus)
	enc.AddArray("cors_headers", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.CorsHeaders {
			encoder.AppendString(v)
		}
		return nil
	}))
	enc.AddString("log_headers", s.LogHeaders)
	enc.AddString("jwt_secret", util.Mask(s.JwtSecret, 2, 3))
	enc.AddInt64("jwt_exp_time", s.JwtExpTime)
	if err := enc.AddObject("http_server_cfg", s.HttpServerCfg); err != nil {
		return err
	}
	if err := enc.AddObject("liveness", s.Liveness); err != nil {
		return err
	}

	return nil
}

type RegistryCfg struct {
	// 注册类型 etcd和nacos、k8s
	Type    string   `yaml:"type" json:"type"`
	Hosts   []string `yaml:"hosts" json:"hosts"`
	Account string   `yaml:"account" json:"account"`
	Pwd     string   `yaml:"pwd" json:"pwd"`
	// 如果是nacos，这个是命名空间id，如果是istio这个是namespace名称,这个参数支持环境变量NAMESPANCE_ID
	NameId string `yaml:"name_id" json:"name_id"`
	// 是否开启istio方式服务调用
	// 可同时与其他类型注册同时使用
	// 如果配置了其他注册中心（type有值），并且开启了istio，则会默认使用istio方式调用
	// 如果没有配置其他注册中心，只开启了istio，只会通过istio去调用
	EnableIstio bool `yaml:"enable_istio" json:"enable_istio"`
}

func (s RegistryCfg) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("type", s.Type)
	enc.AddArray("hosts", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.Hosts {
			encoder.AppendString(v)
		}
		return nil
	}))
	enc.AddString("account", s.Account)
	enc.AddString("pwd", util.Mask(s.Pwd, 2, 3))
	enc.AddString("name_id", s.NameId)
	enc.AddBool("enable_istio", s.EnableIstio)

	return nil
}

type RpcServerCfg struct {
	ListenOn string `yaml:"listen_on" json:"listen_on"`
	// 服务端相关配置
	Server RpcServer `json:"server" yaml:"server"`
	Client RpcClient `yaml:"client" json:"client"`
}

type RpcServer struct {
	Auths []AuthServer `yaml:"auths" json:"auths"`
}

func (s RpcServer) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddArray("auths", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.Auths {
			encoder.AppendObject(v)
		}
		return nil
	}))
	return nil
}

type RpcClient struct {
	// 请求失败重试 默认 3，-1表示不开启重试
	Attempts int `yaml:"attempts" json:"attempts"`
	// 重试间隔，默认 500ms
	AttemptIntervalMs uint `yaml:"attempt_interval_ms" json:"attempt_interval_ms"`
	// 连接池连接最大存活时间，单位 s。默认 3600（1小时），-1 表示不过期（坏连接由连接状态检查剔除）
	PoolTTLS int          `yaml:"pool_ttl_s" json:"pool_ttl_s"`
	Auths    []AuthClient `yaml:"auths" json:"auths"`
}

func (s RpcClient) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddArray("auths", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.Auths {
			encoder.AppendObject(v)
		}
		return nil
	}))
	return nil
}
func (s RpcServerCfg) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("listen_on", s.ListenOn)

	if err := enc.AddObject("server", s.Server); err != nil {
		return err
	}
	if err := enc.AddObject("client", s.Client); err != nil {
		return err
	}
	return nil
}

type AuthClient struct {
	// 开启认证的服务端端点。默认是[]默认为空，表示所有服务都需要认证（支持正则匹配）
	// 假定当前服务为 ca-delay-center
	//
	// ca-delay-center.DelayCenterService.PushDelayData 表示对 delayCenter.DelayCenterService.PushDelayData进行认证
	//
	// ca-delay-center.* 表示对 当前服务的多少有 service 的 method 都进行认证
	//
	// ca-delay-center.DelayCenterService.* 表示对 delayCenter.DelayCenterService的所有method进行认证
	AuthEndpoint []string `yaml:"auth_endpoint" json:"auth_endpoint"`
	// 认证方式 jwt | basic (basic标识明文密码认证)。认证方式为空时候，表示不开启认证
	AuthType string `yaml:"auth_type" json:"auth_type"`
	// 认证密钥
	AuthSecret string `json:"auth_secret" yaml:"auth_secret"`
	// 验证版本 可能对应着不同类型认证方式版本，如jwt的版本号
	AuthVersion string `json:"auth_version" yaml:"auth_version"`
}

func (s AuthClient) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddArray("auth_endpoint", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.AuthEndpoint {
			encoder.AppendString(v)
		}
		return nil
	}))
	enc.AddString("auth_type", s.AuthType)
	enc.AddString("auth_secret", util.Mask(s.AuthSecret, 2, 3))
	enc.AddString("auth_version", s.AuthVersion)

	return nil
}

type AuthServer struct {
	// 开启认证的服务端端点。默认是[]默认为空，表示所有服务都需要认证（支持正则匹配）
	// 假定当前服务为 ca-delay-center
	//
	// ca-delay-center.DelayCenterService.PushDelayData 表示对 delayCenter.DelayCenterService.PushDelayData进行认证
	//
	// ca-delay-center.* 表示对 当前服务的多少有 service 的 method 都进行认证
	//
	// ca-delay-center.DelayCenterService.* 表示对 delayCenter.DelayCenterService的所有method进行认证
	AuthEndpoint []string `yaml:"auth_endpoint" json:"auth_endpoint"`
	// 认证方式 jwt | basic (basic标识明文密码认证)。认证方式为空时候，表示不开启认证
	AuthType string `yaml:"auth_type" json:"auth_type"`
	// 认证密钥
	AuthSecret []string          `json:"auth_secret" yaml:"auth_secret"`
	AuthMap    map[string]string `json:"-" yaml:"-"`
	// 验证版本 可能对应着不同类型认证方式版本，如jwt的版本号
	AuthVersion string `json:"auth_version" yaml:"auth_version"`
	// 是否开启强制验证
	//
	// 强制认证，如果开启强制认证，则根据配置的 AuthEndpoint，对每个进行验证。
	//
	// 非强制认证，如果没有开启强制认证，则根据请求传递的 metadata，如果元数据不存在验证信息则不验证，如果存在则会去验证
	ForceAuth bool `json:"force_auth" yaml:"force_auth"`
}

func (s AuthServer) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddArray("auth_endpoint", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.AuthEndpoint {
			encoder.AppendString(v)
		}
		return nil
	}))
	enc.AddString("auth_type", s.AuthType)
	enc.AddArray("auth_secret", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.AuthSecret {
			encoder.AppendString(util.Mask(v, 2, 3))
		}
		return nil
	}))
	enc.AddString("auth_version", s.AuthVersion)
	enc.AddBool("force_auth", s.ForceAuth)

	return nil
}

type LogCfg struct {
	//DebugLevel  - 1
	//InfoLevel 0
	//WarnLevel 1
	//ErrorLevel 2
	//DPanicLevel 3
	//PanicLevel  4
	//FatalLevel 5
	// 会打印Level及以上级别的日志
	Level int8 `json:"level" yaml:"level"`
	// 开启堆栈的level级别，会开启StacktraceLevel及以上的日志级别的堆栈打印,默认是2
	StacktraceLevel *int8 `json:"stacktrace_level" yaml:"stacktrace_level"`
	//DisableStacktrace完全禁用自动堆栈跟踪捕获。
	//通过默认情况下，在开发中捕获WarnLevel及以上日志，在生产中捕获ErrorLevel及以上
	DisableStacktrace bool `json:"disableStacktrace" yaml:"disableStacktrace"`
}

func (s LogCfg) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt8("level", s.Level)
	if s.StacktraceLevel != nil {
		enc.AddInt8("stacktrace_level", *s.StacktraceLevel)
	}
	enc.AddBool("disableStacktrace", s.DisableStacktrace)
	return nil
}
