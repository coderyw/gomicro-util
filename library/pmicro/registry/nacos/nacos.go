// Package nacos 提供基于Nacos的服务注册与发现功能
// @Author: yinwei
// @File: nacos
// @Version: 1.0.0
// @Date: 2024/6/26 17:36

package nacos

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/cron"
	"go-micro.dev/v4/registry"
	"go.uber.org/zap"
)

// client 全局nacos命名客户端实例
var client naming_client.INamingClient

// nacosRegistry nacos注册中心实现
type nacosRegistry struct {
	client         naming_client.INamingClient // nacos客户端
	opts           registry.Options            // 注册中心配置选项
	sync.RWMutex                               // 读写锁，保护并发访问
	closed         atomic.Bool                 // 标识注册中心是否已关闭
	unhealthyCount int64                       // 不健康连续次数
	last           *registry.Service
}

// NewRegistry 创建新的nacos注册中心实例
// serviceName: 服务名称
// opts: 注册中心配置选项
func NewRegistry(serviceName string, opts ...registry.Option) (registry.Registry, cron.CronJob) {
	log.Info("开始启动 Nacos 注册", zap.String("serviceName", serviceName))
	n := &nacosRegistry{
		opts: registry.Options{
			Context: context.Background(),
		},
	}
	// 配置nacos客户端
	if err := configure(n, opts...); err != nil {
		log.Fatal("Nacos 初始化错误", zap.Error(err))
	}

	// 监听 一致性 hash
	watch(serviceName)
	return n, cron.CronJob{
		Name: "Nacos健康检查",
		DoJob: func() {
			n.check()
		},
		CronSpc:         "0/10 * * * * ?",
		DoBeforeDone:    false,
		DelayBeforeDone: false,
	}
}

// configure 配置nacos注册中心
// n: nacos注册中心实例
// opts: 配置选项
func configure(n *nacosRegistry, opts ...registry.Option) error {
	// 设置nacos日志器
	logger.SetLogger(log.GetNacosLogger())

	// 关闭已存在的客户端连接
	if n.client != nil {
		n.client.CloseClient()
	}

	// 应用配置选项
	for _, o := range opts {
		o(&n.opts)
	}

	clientConfig := clientConfigFrom(n.opts.Context)
	serverConfigs := make([]constant.ServerConfig, 0)
	contextPath := "/nacos"

	// 获取nacos服务器地址
	addrs, ok := n.opts.Context.Value(addressKey{}).([]string)
	if !ok {
		addrs = n.opts.Addrs
		if len(addrs) == 0 {
			return errors.New("缺失nacos地址")
		}
	}

	// 解析服务器地址并构建配置
	for _, addr := range addrs {
		// 检查是否包含端口
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return err
		}

		// 转换端口号
		p, err := strconv.ParseUint(port, 10, 64)
		if err != nil {
			return err
		}

		serverConfigs = append(serverConfigs, constant.ServerConfig{
			IpAddr:      host,
			Port:        p,
			ContextPath: contextPath,
		})
	}

	// 设置默认超时时间
	if n.opts.Timeout == 0 {
		n.opts.Timeout = time.Second * 3
	}

	// 设置客户端超时时间
	clientConfig.TimeoutMs = uint64(n.opts.Timeout.Milliseconds())

	// 创建nacos命名客户端
	var err error
	client, err = clients.CreateNamingClient(map[string]interface{}{
		constant.KEY_SERVER_CONFIGS: serverConfigs,
		constant.KEY_CLIENT_CONFIG:  clientConfig,
	})
	if err != nil {
		return err
	}

	// 等待 nacos 就绪
	waitTimes := 30
	for {
		waitTimes -= 3
		if waitTimes < 0 {
			log.Fatal("Nacos 未就绪")
		}
		if client.ServerHealthy() {
			break
		}
		time.Sleep(3 * time.Second)
	}

	// 线程安全地设置客户端
	n.Lock()
	n.client = client
	n.Unlock()

	return nil
}

// clientConfigFrom 取客户端配置：上下文里有 WithClientConfig 传入的配置就用它，否则用默认配置。
//
// 无论哪种来源都强制 NotLoadCacheAtStart=true。SDK 的 GetService 本地缓存命中就不问服务端，
// 若启动时加载了上次运行留在 CacheDir 的服务列表，第一次调用会打到旧地址：
// 容器原地重启后 IP 可能被别的容器占用（甚至是自己），表现为 Unimplemented / unknown service。
// 启动时 Nacos 不可用本来就会 Fatal，加载磁盘缓存换不来可用性。
// WithClientConfig 是整份替换，调用方只填账密时其余字段都是零值，所以不能依赖调用方设置。
func clientConfigFrom(ctx context.Context) constant.ClientConfig {
	clientConfig := constant.ClientConfig{
		UpdateCacheWhenEmpty: true,    // 缓存为空时更新缓存
		LogLevel:             "debug", // 日志级别
	}
	if ctx != nil {
		if cfg, ok := ctx.Value(configKey{}).(constant.ClientConfig); ok {
			clientConfig = cfg
		}
	}
	clientConfig.NotLoadCacheAtStart = true
	return clientConfig
}

// getNodeIPPort 从服务中提取节点的IP和端口
// s: 注册服务实例
// 返回: 主机地址、端口号、错误信息
func getNodeIPPort(s *registry.Service) (host string, port int, err error) {
	if len(s.Nodes) == 0 {
		return "", 0, errors.New("you must deregister at least one node")
	}
	node := s.Nodes[0]
	host, pt, err := net.SplitHostPort(node.Address)
	if err != nil {
		return "", 0, err
	}
	port, err = strconv.Atoi(pt)
	if err != nil {
		return "", 0, err
	}
	return
}

// check 注册中心状态校验，定期检查nacos服务器健康状态
func (n *nacosRegistry) check() {
	if !n.client.ServerHealthy() {
		log.Warn("注册中心检查错误")
		if atomic.AddInt64(&n.unhealthyCount, 1) > 5 {
			log.Fatal("注册中心未就绪")
		}
	} else {
		atomic.StoreInt64(&n.unhealthyCount, 0)
	}
}

// Init 初始化注册中心，重新配置客户端
func (n *nacosRegistry) Init(opts ...registry.Option) error {
	_ = configure(n, opts...)
	return nil
}

// Options 获取注册中心配置选项
func (n *nacosRegistry) Options() registry.Options {
	return n.opts
}

// Register 注册服务实例到nacos
// s: 要注册的服务
// opts: 注册选项
func (n *nacosRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	// 执行注册操作
	n.RLock()
	defer n.RUnlock()
	if n.closed.Load() {
		log.Warn("实例已经注销，不再进行注册")
		// 已经停止注册不再去注册
		return nil
	}
	n.last = s
	var options registry.RegisterOptions
	// 应用注册选项
	for _, o := range opts {
		o(&options)
	}

	withContext := false
	param := vo.RegisterInstanceParam{}

	// 检查是否通过上下文传递了注册参数
	if options.Context != nil {
		if p, ok := options.Context.Value("register_instance_param").(vo.RegisterInstanceParam); ok {
			param = p
			withContext = ok
		}
	}

	// 如果没有通过上下文传递参数，则构建默认参数
	if !withContext {
		host, port, err := getNodeIPPort(s)
		if err != nil {
			return err
		}
		// 设置版本信息到元数据
		s.Nodes[0].Metadata["version"] = s.Version
		param.Ip = host
		param.Port = uint64(port)
		// 复制元数据
		param.Metadata = make(map[string]string)
		for k, v := range s.Nodes[0].Metadata {
			param.Metadata[k] = v
		}
		param.ServiceName = s.Name
		param.Enable = true    // 启用实例
		param.Healthy = true   // 实例健康
		param.Weight = 1.0     // 权重
		param.Ephemeral = true // 临时实例
	}

	// 执行注册操作（保留 NS 的并发锁保护 + SAAS 的注册结果校验）
	n.RLock()
	succ, err := n.client.RegisterInstance(param)
	n.RUnlock()
	if err != nil {
		log.Error("注册中心注册错误", zap.String("serviceName", s.Name), zap.Error(err))
		return err
	}
	if !succ {
		return errors.New("注册中心注册失败")
	}
	return nil
}

// Deregister 从nacos注销服务实例
// s: 要注销的服务
// opts: 注销选项
func (n *nacosRegistry) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {
	n.Lock()
	defer n.Unlock()
	if s == nil {
		s = n.last
	}
	if s == nil { //参数不对，不去做
		return nil
	}
	log.Info("从 nacos 注销实例")
	// 标记注册中心为关闭状态
	if !n.closed.CompareAndSwap(false, true) {
		log.Warn("已经注销过实例")
		return nil
	}

	var options registry.DeregisterOptions
	// 应用注销选项
	for _, o := range opts {
		o(&options)
	}

	withContext := false
	param := vo.DeregisterInstanceParam{}

	// 检查是否通过上下文传递了注销参数
	if options.Context != nil {
		if p, ok := options.Context.Value("deregister_instance_param").(vo.DeregisterInstanceParam); ok {
			param = p
			withContext = ok
		}
	}

	// 如果没有通过上下文传递参数，则构建默认参数
	if !withContext {
		host, port, err := getNodeIPPort(s)
		if err != nil {
			return err
		}
		param.Ip = host
		param.Port = uint64(port)
		param.ServiceName = s.Name
	}
	log.Info("注销 nacos 实例", zap.Any("param", param))
	_, err := n.client.DeregisterInstance(param)
	return err
}

// GetService 从nacos获取指定名称的服务实例列表
// name: 服务名称
// opts: 获取选项
func (n *nacosRegistry) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	var options registry.GetOptions
	// 应用获取选项
	for _, o := range opts {
		o(&options)
	}

	withContext := false
	param := vo.GetServiceParam{}

	// 检查是否通过上下文传递了查询参数
	if options.Context != nil {
		if p, ok := options.Context.Value("select_instances_param").(vo.GetServiceParam); ok {
			param = p
			withContext = ok
		}
	}

	// 如果没有通过上下文传递参数，则使用服务名称
	if !withContext {
		param.ServiceName = name
	}

	// 从nacos获取服务信息
	n.RLock()
	service, err := n.client.GetService(param)
	n.RUnlock()
	if err != nil {
		return nil, err
	}

	// 构建服务实例列表
	services := make([]*registry.Service, 0)
	for _, v := range service.Hosts {
		// 只返回健康、启用且权重大于0的实例
		if !v.Healthy || !v.Enable || v.Weight <= 0 {
			continue
		}

		// 构建节点信息
		nodes := make([]*registry.Node, 0)
		nodes = append(nodes, &registry.Node{
			Id:       v.InstanceId,
			Address:  net.JoinHostPort(v.Ip, fmt.Sprintf("%d", v.Port)),
			Metadata: v.Metadata,
		})

		// 构建服务信息
		s := registry.Service{
			Name:     v.ServiceName,
			Version:  v.Metadata["version"],
			Metadata: v.Metadata,
			Nodes:    nodes,
		}
		services = append(services, &s)
	}

	return services, nil
}

// ListServices 从nacos获取所有服务列表
// opts: 列表选项
func (n *nacosRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	var options registry.ListOptions
	// 应用列表选项
	for _, o := range opts {
		o(&options)
	}

	withContext := false
	param := vo.GetAllServiceInfoParam{}

	// 检查是否通过上下文传递了查询参数
	if options.Context != nil {
		if p, ok := options.Context.Value("get_all_service_info_param").(vo.GetAllServiceInfoParam); ok {
			param = p
			withContext = ok
		}
	}

	n.RLock()
	defer n.RUnlock()

	// 如果没有通过上下文传递参数，则先获取服务总数设置分页参数
	if !withContext {
		services, err := n.client.GetAllServicesInfo(param)
		if err != nil {
			return nil, err
		}
		param.PageNo = 1                        // 页码
		param.PageSize = uint32(services.Count) // 页大小
	}

	// 获取所有服务信息
	services, err := n.client.GetAllServicesInfo(param)
	if err != nil {
		return nil, err
	}

	// 构建注册服务列表
	var registryServices []*registry.Service
	for _, v := range services.Doms {
		registryServices = append(registryServices, &registry.Service{Name: v})
	}
	return registryServices, nil
}

// Watch 创建服务监听器，用于监听服务变化
// opts: 监听选项
func (n *nacosRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	return newWatcher(n, opts...)
}

// String 返回注册中心类型名称
func (n *nacosRegistry) String() string {
	return "nacos"
}
