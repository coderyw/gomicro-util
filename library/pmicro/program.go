package pmicro

import (
	"context"
	"io"
	"net/http"
	_ "net/http/pprof"
	"sort"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	_ "github.com/coderyw/gomicro-util/library/automaxprocs"
	"github.com/coderyw/gomicro-util/library/component/storage"
	cron3 "github.com/coderyw/gomicro-util/library/cron"
	"github.com/coderyw/gomicro-util/library/db/cache/redis"
	"github.com/coderyw/gomicro-util/library/db/mysql"
	env2 "github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/mq"
	"github.com/coderyw/gomicro-util/library/mq/model"
	"github.com/coderyw/gomicro-util/library/pmicro/config"
	http2 "github.com/coderyw/gomicro-util/library/pmicro/http"
	"github.com/coderyw/gomicro-util/library/pmicro/metrics"
	"github.com/coderyw/gomicro-util/library/pmicro/option"
	poll_nacos "github.com/coderyw/gomicro-util/library/pmicro/selector/nacos"
	"github.com/coderyw/gomicro-util/library/single"
	"github.com/coderyw/gomicro-util/library/svc"
	srv_config "github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/cron"
	"github.com/coderyw/gomicro-util/model/cst"
	"github.com/coderyw/gomicro-util/model/micro"
	"go-micro.dev/v4/selector"
	"go.uber.org/zap"
)

const perfPort = ":18888"
const microPort = ":50051"
const httpPort = ":51001"

type Cron struct {
	Command  string
	Interval time.Duration
	CronFunc func()
}

func This() *Program {
	return this
}

func GetOtherCfg() interface{} {
	return this.cfg.GetOtherCfg()
}

func GetCfg() srv_config.ServerConfig {
	return this.cfg.GetCfg()
}

// GetPlatformCfg 获取平台名称
func GetPlatformCfg() string {
	return this.cfg.GetCfg().Platform
}

// GetTokenSaltCfg 获取token盐值
func GetTokenSaltCfg() string {
	return this.cfg.GetCfg().TokenSalt
}

var this *Program

type Program struct {
	MicroService *MicroService
	Options      *option.Options
	cfg          *config.ServerCfg

	closeF     []micro.CloseS
	httpServer []*http.Server
	// 链路追踪 默认通过配置设置，如果想自定义请传值
	Tracer       opentracing.Tracer
	TracerCloser io.Closer
}

func (p *Program) GetConfig() srv_config.ServerConfig {
	return p.cfg.GetCfg()
}

func NewProgram(optionFunc ...option.Option) *Program {
	options := &option.Options{
		//Selector: poll_etcd.PollNodeSelector, // 默认轮训select etcd
		Selector:       poll_nacos.PollNodeSelector,
		ConfigFile:     "etc/cfg.yaml",
		MQKafkaHandler: make([]model.Handler, 0),
	}
	for _, opt := range optionFunc {
		opt(options)
	}
	this = &Program{Options: options}
	return this
}

// 默认版本号
const LATEST = "latest"

//func (p *Program) initTracer() {
//	if len(p.cfg.Tracer.Host) > 0 {
//		tracer, closer, err := jaeger.NewTracer(p.Options.ServerName, fmt.Sprintf("%v:%v", p.cfg.Tracer.Host, p.cfg.Tracer.Port))
//		if err != nil {
//			log.Error("初始化链路追踪错误", zap.Any("host", p.cfg.Tracer), zap.Error(err))
//			return
//		}
//		p.TracerCloser = closer
//		p.Tracer = tracer
//	}
//}

func (p *Program) GetId() string {
	return p.MicroService.id
}

func (p *Program) Init() error {
	//初始化日志
	log.InitDefaultLogger(srv_config.LogCfg{
		Level:             0,
		StacktraceLevel:   nil,
		DisableStacktrace: false,
	}, p.Options.ServerName, env2.GetVersion())
	var err error
	// 非本地环境先等 sidecar 就绪，再做首跳出站（凭据、Nacos）
	if !env2.IsLocal() {
		logSidecarWait(waitSidecarReady(context.Background(), sidecarWaitFromEnv()))
	}
	// 初始化日志
	cfg, err := config.InitCfg(p.Options)
	if err != nil {
		return err
	}
	p.cfg = cfg
	//重新根据cfg初始化log
	log.InitDefaultLogger(p.cfg.GetCfg().Log, p.Options.ServerName, env2.GetVersion())
	logBuildInfo()
	//初始化tracer
	//p.initTracer()

	if p.cfg.GetCfg().Storage.Redis.Enable { //redis
		if stop, job, err := redis.InitRedis(p.cfg.GetCfg().Storage.Redis); err != nil {
			return err
		} else {
			p.setClose(stop)
			if job != nil {
				p.Options.CronJob = append(p.Options.CronJob, *job)
			}
		}
	}
	// redis要在mysql之前初始化
	if job, stop, err := mysql.InitConnect(p.Options.ServerName, p.cfg.GetCfg().Storage.Mysql); err != nil {
		return err
	} else {
		if stop != nil {
			p.setClose(*stop)
		}
		if job != nil {
			p.Options.CronJob = append(p.Options.CronJob, *job)
		}
	}

	//初始化 oss
	cl, err := storage.InitObjectStorage(p.cfg.GetCfg().Storage.ObjectStorage)
	if err != nil {
		return err
	}
	p.setClose(micro.CloseS{CloseFunc: cl})

	if p.Options.BeforeStartFunc != nil {
		if err := p.Options.BeforeStartFunc(); err != nil {
			return err
		}
	}
	return nil
}

func (p *Program) Start() (err error) {

	microService := new(MicroService)
	// 开启微服务
	err = microService.Start(p.cfg.GetCfg(), p.Options)
	if err != nil {
		return err
	}
	p.MicroService = microService

	//初始kafka
	var (
		kafkaClose func()
		kafkaJob   []cron.CronJob
	)
	if kafkaClose, kafkaJob, err = mq.InitKafka(p.Options.ServerName, p.cfg.GetCfg().KafkaCfg, p.Options.MQKafkaHandler); err != nil {
		return err
	}
	p.Options.CronJob = append(p.Options.CronJob, kafkaJob...)
	p.setCloseInfo(10, kafkaClose)

	// 监控指标上报（未开启时返回空 CloseS，安全）
	p.setClose(metrics.Init(p.cfg.GetCfg().Metrics, p.Options.ServerName))

	// 对关闭函数排序，降序
	sort.Slice(p.closeF, func(i, j int) bool {
		return p.closeF[i].Weight > p.closeF[j].Weight
	})

	// 基本组件启动后，可以执行 afterfunc
	if p.Options.AfterFunc != nil {
		if err = p.Options.AfterFunc(); err != nil {
			return err
		}
	}

	// 开启 Http 监听
	if httpServer, err := http2.HttpServer(p.cfg.GetCfg().Http, *p.Options); err != nil {
		return err
	} else {
		p.httpServer = append(p.httpServer, httpServer)
	}
	// 禁止暴露公网
	if httpServer, err := http2.LivenessServer(p.cfg.GetCfg().Http.Liveness); err != nil {
		return err
	} else {
		p.httpServer = append(p.httpServer, httpServer)
	}

	if len(p.Options.CronJob) > 0 {
		p.initCron()
	}
	single.SetReadinessProbe(true)
	return nil
}

func (p *Program) Stop() error {
	log.Info("Program开始停止服务...")
	single.SetReadinessProbe(false)
	log.Info("停止服务中...")

	// 注销微服务
	log.Info("ReadinessProbe 设置为 false")
	if p.MicroService != nil {
		p.MicroService.Stop()
	}
	if p.httpServer != nil {
		log.Info("开始停止HTTP服务器...")
		//创建超时上下文，Shutdown可以让未处理的连接在这个时间内关闭
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		//停止HTTP服务器
		for _, hs := range p.httpServer {
			if hs != nil {
				if err := hs.Shutdown(ctx); err != nil {
					log.Error("HTTP Server Shutdown:", zap.Error(err))
				}
			}
		}

	}
	log.Info("开始停止定时任务")
	cron3.CloseCron()

	if p.Options.BeforeStopFunc != nil {
		log.Info("开始执行BeforeStopFunc")
		p.Options.BeforeStopFunc()
	}
	log.Info("开始执行关闭函数")
	for _, closeF := range p.closeF {
		if closeF.CloseFunc != nil {
			log.Info("中间件关闭函数", zap.Int("weight", closeF.Weight))
			closeF.CloseFunc()
		}
	}
	if p.TracerCloser != nil {
		p.TracerCloser.Close()
	}
	log.Info("Program停止完成")
	return nil
}

// 获取服务的ip
func (p *Program) GetServerIps(serverName string) ([]string, error) {
	if srvs, err := p.MicroService.service.Options().Registry.GetService(serverName); err != nil {
		return nil, err
	} else {
		//过滤
		srvs = selector.FilterLabel(cst.ENV_ENVIRON, env2.GetEnv())(srvs)

		if len(srvs) == 0 {
			return nil, selector.ErrNotFound
		}
		resp := make([]string, 0)
		var (
			arr []string
		)
		for _, v := range srvs[0].Nodes {
			arr = strings.Split(v.Address, ":")
			if len(arr) > 0 {
				resp = append(resp, arr[0])
			}
		}

		return resp, nil
	}
}

func (p *Program) Run() error {
	return svc.Run(p)
}

func (p *Program) setCloseInfo(weight int, f func()) {
	if f == nil {
		return
	}
	p.closeF = append(p.closeF, micro.CloseS{
		Weight:    weight,
		CloseFunc: f,
	})
}

func (p *Program) setClose(c micro.CloseS) {
	p.closeF = append(p.closeF, c)
}
