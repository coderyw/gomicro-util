// Package pmicro
// @Author: yinwei
// @File: micro
// @Version: 1.0.0
// @Date: 2023/11/2 22:35

package pmicro

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/urfave/cli/v2"
	cron2 "github.com/coderyw/gomicro-util/library/cron"
	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/pmicro/option"
	grpccli "github.com/coderyw/gomicro-util/library/pmicro/plugin/client_grpc"
	grpcsrv "github.com/coderyw/gomicro-util/library/pmicro/plugin/server_grpc"
	registry2 "github.com/coderyw/gomicro-util/library/pmicro/registry"
	"github.com/coderyw/gomicro-util/library/pmicro/registry/nacos"
	etcd_selector "github.com/coderyw/gomicro-util/library/pmicro/selector/etcd"
	"github.com/coderyw/gomicro-util/library/pmicro/selector/istio"
	poll_nacos "github.com/coderyw/gomicro-util/library/pmicro/selector/nacos"
	"github.com/coderyw/gomicro-util/library/pmicro/wait_stop"
	"github.com/coderyw/gomicro-util/library/pmicro/wrapper/auth"
	"github.com/coderyw/gomicro-util/library/pmicro/wrapper/metadata"
	"github.com/coderyw/gomicro-util/library/pmicro/wrapper/monitoring/prometheus"
	server2 "github.com/coderyw/gomicro-util/library/pmicro/wrapper/recovery"
	"github.com/coderyw/gomicro-util/library/pmicro/wrapper/response"
	"github.com/coderyw/gomicro-util/library/pmicro/wrapper/tracer"
	"github.com/coderyw/gomicro-util/library/pmicro/wrapper/validator"
	"github.com/coderyw/gomicro-util/library/pmicro/wrapper/wrapper_client"
	etcd_config "github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/cron"
	"github.com/coderyw/gomicro-util/model/cst"
	"github.com/coderyw/gomicro-util/util"
	"go-micro.dev/v4"
	"go-micro.dev/v4/client"
	config2 "go-micro.dev/v4/config"
	cli2 "go-micro.dev/v4/config/source/cli"
	logger2 "go-micro.dev/v4/logger"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/selector"
	"go-micro.dev/v4/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	//"github.com/opentracing/opentracing-go/ext"
	//"github.com/uber/jaeger-client-go"
	grpcTransport "github.com/go-micro/plugins/v4/transport/grpc"
)

type Servicer interface {
	GrpcServer() *grpc.Server
}

type InitFunc func(client client.Client)

var initFuncArr []InitFunc

var service micro.Service

func RegisterInitClient(fs []InitFunc) {
	initFuncArr = append(initFuncArr, fs...)
}

func GetService() micro.Service {
	return service
}

func SetService(m micro.Service) {
	service = m
}

type MicroService struct {
	cancelFunc context.CancelFunc
	service    micro.Service
	wg         util.WaitGroupWrapper
	id         string
	registry   registry.Registry
}

// 根据配置获取具体的注册中心
func (m *MicroService) getRegOpt(ctx context.Context, config etcd_config.ServerConfig, microLog logger2.Logger, cfg etcd_config.RegistryCfg, options *option.Options) (context.Context, []registry.Option, error) {
	regOpt := []registry.Option{}

	switch cfg.Type {
	case cst.REGISTRY_TYPE_ETCD: // etcd
		regOpt = append(regOpt, func(options *registry.Options) {
			options.Addrs = cfg.Hosts
			options.Context = ctx
			options.Logger = microLog
		})
		options.Selector = etcd_selector.PollNodeSelector
	case cst.REGISTRY_TYPE_NACOS: // nacos
		regOpt = append(regOpt, func(options *registry.Options) {
			options.Logger = microLog
			options.Context = ctx
			options.Addrs = cfg.Hosts
		},
			nacos.WithAddress(cfg.Hosts),
			nacos.WithClientConfig(constant.ClientConfig{
				Username:    cfg.Account,
				Password:    cfg.Pwd,
				NamespaceId: cfg.NameId,
			}),
		)
		options.Selector = poll_nacos.PollNodeSelector

	case cst.REGISTRY_TYPE_K8S: // k8s
		regOpt = append(regOpt, func(options *registry.Options) {
			options.Logger = microLog
			options.Context = ctx
		})
	default:
		if len(cfg.Type) != 0 {
			return nil, nil, errors.New("error of registry type")
		}
	}
	if cfg.EnableIstio {
		// 不进行配置，使用默认的
		regOpt = append(regOpt, func(options *registry.Options) {
			if options.Context == nil {
				options.Context = context.Background()
			}
			// ctx传入一些参数
			ctx := context.WithValue(options.Context, "env", config.ENVIRON)
			ctx = context.WithValue(ctx, "grpcport", config.RpcServer.ListenOn)
			options.Context = ctx
		})
		// 修改selector
		options.Selector = istio.IstioSelector
	}

	return ctx, regOpt, nil
}

func (m *MicroService) Start(config etcd_config.ServerConfig, options *option.Options) (err error) {
	m.id = env.GetId()

	microLog := log.NewMicroLog()

	// 注册中心配置
	var (
		regOpt []registry.Option
		ctx    context.Context
	)
	ctx, regOpt, err = m.getRegOpt(context.Background(), config, microLog, config.Registry, options)
	if err != nil {
		return
	}
	ctx, cancelFunc := context.WithCancel(ctx)
	defer func() {
		if err != nil {
			cancelFunc()
		}
	}()
	m.cancelFunc = cancelFunc
	var cjob cron.CronJob
	m.registry, cjob, err = registry2.NewRegistry(registry2.RegDis{ServiceName: options.ServerName, RDType: config.Registry.Type, Opts: regOpt, Istio: config.Registry.EnableIstio})
	if err != nil {
		return err
	}
	// 定时任务配置
	if cjob.CronSpc != "" {
		cron2.AddCron(cjob)
	}

	// service配置
	opts := []micro.Option{
		micro.Name(options.ServerName),
		micro.Version(env.GetVersion()),
		micro.Address(config.RpcServer.ListenOn),
		micro.Registry(m.registry),
		micro.RegisterTTL(time.Second * 5),
		micro.RegisterInterval(time.Second * 10),
		micro.Context(ctx),
		micro.Metadata(map[string]string{
			cst.ENV_ENVIRON: env.GetEnv(),
			"id":            m.id,
		}),
		micro.Logger(microLog),
		// 这块使用grpc的server插件，为了兼容其他微服务框架调用本框架，比如gzero或者java的grpc，如果
		// 只是gomicro调用gomicro，则可以不使用
		micro.Server(grpcsrv.NewServer()),
		micro.Client(grpccli.NewClient()),
		micro.WrapClient(wrapper_client.ClientWrapper(This().cfg)),
		// 改用 grpc 的 transport
		micro.Transport(grpcTransport.NewTransport()),
	}
	if len(options.Flags) > 0 {
		var ff []cli.Flag
		for _, v := range options.Flags {
			ff = append(ff, &cli.StringFlag{
				Name:  v.Name,
				Usage: v.Usage,
				Value: v.Value,
			})
		}
		opts = append(opts, micro.Flags(ff...))
	}

	// 创建service
	serv := micro.NewService(opts...)

	// service初始化
	serv.Init()
	// server初始化  配置
	serv.Server().Init(
		server.Registry(m.registry),
		server.Context(ctx),
		server.Metadata(map[string]string{
			cst.ENV_ENVIRON: env.GetEnv(),
			"id":            m.id,
		}),
		server.Name(options.ServerName),
		server.Version(env.GetVersion()),
		server.Address(config.RpcServer.ListenOn),
		server.WithLogger(microLog),

		// recover panic中间件
		server.WrapHandler(server2.RecoveryHandler),
		// trace中间件
		server.WrapHandler(tracer.NewHandlerWrapper()),
		// 消息回复中间件
		server.WrapHandler(response.RespHandlerWrapper()),
		// auth 校验
		server.WrapHandler(auth.NewHandlerWrapper(This().cfg)),
		// 参数校验
		server.WrapHandler(validator.NewHandlerWrapper()),
		// 普罗米修斯
		server.WrapHandler(prometheus.NewHandlerWrapper(func(op *prometheus.Options) {
			op.Name = options.ServerName
		})),
		server.WrapHandler(metadata.NewHandlerWrapper()),

		// 注册过期时间
		server.RegisterTTL(time.Second*10),
		// 注册时间间隔
		server.RegisterInterval(time.Second*5),
	)
	// 本地开启grpc反射服务
	if config.ENVIRON == cst.RUNTIME_ENV_LOCAL {
		if srv := grpcsrv.GetGrpcServer(serv.Server()); srv != nil {
			reflection.Register(srv)
		} else {
			log.Error("获取grpc server错误")
		}
	}

	// call 中间件
	wrapCall := make([]client.CallWrapper, 0)
	// call recovery
	wrapCall = append(wrapCall, server2.RecoveryClient())
	// 客户端调用指标统计
	wrapCall = append(wrapCall, prometheus.NewCallWrapper(func(op *prometheus.Options) {
		op.Name = options.ServerName
	}))
	// 消息回复
	wrapCall = append(wrapCall, response.NewCallWrapper())
	// validator中间层
	wrapCall = append(wrapCall, validator.NewCallWrapper())
	wrapCall = append(wrapCall, metadata.NewCallWrapper())
	atp := 0
	// 重试配置
	if config.RpcServer.Client.Attempts == 0 {
		atp = 2
	} else if config.RpcServer.Client.Attempts > 0 {
		atp = config.RpcServer.Client.Attempts - 1
	}
	atpInt := uint(0)
	if config.RpcServer.Client.AttemptIntervalMs > 0 {
		atpInt = config.RpcServer.Client.AttemptIntervalMs
	} else {
		atpInt = 500
	}
	// 连接池连接存活时间配置
	poolTTL := time.Hour
	if config.RpcServer.Client.PoolTTLS > 0 {
		poolTTL = time.Duration(config.RpcServer.Client.PoolTTLS) * time.Second
	} else if config.RpcServer.Client.PoolTTLS < 0 {
		// 不过期，坏连接由连接池的状态检查剔除
		poolTTL = math.MaxInt64
	}
	// client初始化配置
	serv.Client().Init(
		func(options *client.Options) {
			options.CallOptions.Retries = atp //重试 atp 次，重试规则如下
			// 教研是否可以重试
			options.CallOptions.Retry = func(ctx context.Context, req client.Request, retryCount int, err error) (bool, error) {
				if err == nil {
					return false, nil
				}
				if strings.Contains(err.Error(), "invalid connection") {
					log.WarnCtx(ctx, "[GRPC] call failed, will retry", zap.Error(err))
					time.Sleep(time.Duration(atpInt) * time.Millisecond) // 等 atpInt ms 再进行重试
					return true, nil
				}
				// 判断是否是特定错误，由于使用 Envoy 代理方式请求，在上游服务发布期间，可能会出现请求不通服务问题
				// 所以为了兼容这个问题，可以做出重试机制.
				st, ok := status.FromError(err)
				if ok {
					// 请求不通，并且错误是下游找不到情况。
					if st.Code() == codes.Unavailable {
						log.WarnCtx(ctx, "[GRPC] call failed, will retry", zap.Any("retryCount", retryCount), zap.Error(err))
						time.Sleep(time.Duration(atpInt) * time.Millisecond) // 等 atpInt ms 再进行重试
						return true, nil
					}
					return false, nil

				}
				// 不能gomicro 这块报错。因为 gomicro 的错误不能确认服务端是否收到消息。
				return false, err
			}
		},
		client.Selector(options.Selector(func(options *selector.Options) {
			options.Registry = m.registry
			options.Logger = microLog
			// 设置selector的ttl（缓存时间）
			options.Context = context.WithValue(ctx, "selector_ttl", time.Second*3)
		})),
		client.WithLogger(microLog),
		client.Registry(m.registry),
		client.PoolSize(50),
		client.PoolTTL(poolTTL),
		client.DialTimeout(time.Second*3),
		client.RequestTimeout(time.Second*30),
		client.WrapCall(wrapCall...),
	)

	service = serv
	m.service = serv
	// 注册handler函数
	if options.RegisterHandlerFunc != nil {
		if err = options.RegisterHandlerFunc(serv); err != nil {
			return err
		}
	}

	// 初始化自定义配置i
	err = config2.Load(cli2.NewSource())
	if err != nil {
		log.Error("load config error", zap.Error(err))
	}

	m.wg.Wrap(func() {
		if err := service.Run(); err != nil {
			log.Error("Micro 服务启动错误 ", zap.Error(err))
		}
	})
	for _, f := range initFuncArr {
		f(service.Client())
	}
	return nil
}

func (m *MicroService) Stop() {
	log.Info("停止Micro...")
	m.cancelFunc()
	wait_stop.Stop()
	//  注意这里不需要再次停止，因为这里已经在micro中停止了
	// m.service.Server().Stop()
	wait_stop.Wait()
	m.wg.Wait()
	log.Info("停止Micro成功")
}
