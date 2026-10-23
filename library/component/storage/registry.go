package storage

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	storage2 "github.com/coderyw/gomicro-util/internal/storage"
	huaweiprovider "github.com/coderyw/gomicro-util/internal/storage/huawei"
	s3provider "github.com/coderyw/gomicro-util/internal/storage/s3"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/namespace"
	"github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/storage"
	"go.uber.org/zap"
)

// registryPtr 存放当前生效的 registry，未初始化时为 nil。
//
// 用 atomic.Pointer 而不是裸指针：Upload / HasInstance 在请求路径上读它，
// InitObjectStorage 在启动路径上写它，裸指针的读写在 Go 内存模型下是数据竞争
// （即便实际只在启动时写一次，也会被 -race 检出，且未来若接热更新会真的出问题）。
var registryPtr atomic.Pointer[storage2.Registry]

func InitObjectStorage(cfg *config.ObjectStorage) (func(), error) {
	if cfg == nil {
		return nil, nil
	}

	// 实例名在所有 provider 之间必须唯一：同名实例分属不同的云会导致分派歧义。
	// 这里提前查重是为了给出明确的错误信息，Registry.Register 本身也会兜住重复注册。
	mp := make(map[string]bool)
	for instance := range cfg.AwsS3Config {
		mp[instance] = true
	}
	for instance := range cfg.HuaweiObsConfig {
		if mp[instance] {
			return nil, fmt.Errorf("duplicated object storage instance: %s", instance)
		}
		mp[instance] = true
	}

	// 默认实例必须真实存在。指向不存在实例的默认值比没有默认值更糟：
	// 业务侧留空 Instance 时会以为「跟随部署方默认」，实际每次上传都 instance not found。
	// 这里复用查重时建好的 mp 先做一次早退判断，给出比 SetDefault 更具体的错误信息。
	defaultInstance := strings.TrimSpace(cfg.DefaultInstance)
	if defaultInstance != "" && !mp[defaultInstance] {
		return nil, fmt.Errorf("storage.object_storage.default_instance=%q not found in aws_s3_config / huawei_obs_config",
			defaultInstance)
	}

	// 先在本地把 registry 装配完整，全部成功后再发布到 registryPtr。
	// 原先是先把包级变量赋成空壳再逐个注册，中途任一 provider 构造失败就会留下
	// 一个「已发布但只注册了一半」的 registry。
	r := storage2.NewRegistry()
	// 对每个云进行初始化
	for instance, cfg := range cfg.AwsS3Config {
		if cfg == nil {
			log.Warn("aws s3 config is nil", zap.Any("instance", instance))
			continue
		}
		prodS3, err := s3provider.New(instance, *cfg)
		if err != nil {
			return nil, err
		}
		if err = r.Register(instance, prodS3); err != nil {
			return nil, err
		}
	}
	for instance, cfg := range cfg.HuaweiObsConfig {
		if cfg == nil {
			log.Warn("huawei obs config is nil", zap.Any("instance", instance))
			continue
		}
		prodObs, err := huaweiprovider.New(instance, *cfg)
		if err != nil {
			return nil, err
		}
		if err = r.Register(instance, prodObs); err != nil {
			return nil, err
		}
	}

	// 在发布之前设默认：此刻 r 还是本 goroutine 独占的，与后续只读的请求路径之间
	// 由 registryPtr.Swap 建立 happens-before。
	//
	// 上面 mp 的检查放行了「实例名在配置里、但那份 cfg 是 nil 被跳过注册」的情况，
	// 这里会兜住——默认实例没真正注册成功，宁可启动失败也不能带病上线。
	if err := r.SetDefault(defaultInstance); err != nil {
		return nil, err
	}

	// 重复初始化不是预期用法：本组件由 pmicro 在启动时调用一次，不支持热更新。
	// 真发生了就把旧的关掉，避免上一份 registry 的连接池无人回收。
	if old := registryPtr.Swap(r); old != nil {
		log.Warn("object storage re-initialized, closing the previous registry")
		old.Close()
	}

	log.Info("object storage init success", zap.Any("objectCfg", cfg))
	return close, nil
}

// Upload 上传文件到对象存储
//
// Parameters:
//   - ctx context.Context
//   - req *storage.UploadRequest
//
// Returns：
//   - *storage.UploadResult
//   - error
func Upload(ctx context.Context, req *storage.UploadRequest) (*storage.UploadResult, error) {
	if req == nil {
		return nil, storage2.ErrInvalidArgument
	}
	// 未配置 storage.object_storage 时 InitObjectStorage 会直接返回，registryPtr 保持为 nil。
	// 这里显式判断，避免调用方拿到一个 nil 解引用的 panic。
	reg := registryPtr.Load()
	if reg == nil {
		return nil, fmt.Errorf("%w: object storage is not initialized, check storage.object_storage config",
			storage2.ErrInstanceNotFound)
	}
	// 在副本上加前缀，不碰调用方传进来的 req。
	//
	// 原先是就地改写 req.Key：调用方若复用同一个 UploadRequest 重试，第二次会在
	// 已带前缀的 key 上再加一层，变成 tenantok/tenantok/xxx。provider 内部同理
	// 不再改写 req.Bucket。浅拷贝足够——Body / Metadata / Tags 在下游都是只读的。
	r := *req
	r.Key = getPrefix(ctx, strings.TrimLeft(req.Key, "/"))
	return reg.Upload(ctx, &r)
}

// HasInstance 判断实例名是否已注册。
//
// 用途：other.xxx_instance 这类「跨配置段的字符串引用」在配置层面没有任何约束，
// 名字写错、或实例漏挂在 aws_s3_config / huawei_obs_config 下，都要等到真正上传时
// 才报 instance not found。业务方应在启动阶段（pmicro 的 BeforeStartFunc，
// 该钩子在 InitObjectStorage 之后执行）用本函数把这类错配前移到发布时暴露。
//
// 面向启动阶段：不要在请求路径上用它做上传前探测——
// Upload 本身已经会返回 ErrInstanceNotFound，多一次探测只是徒增开销。
func HasInstance(instance string) bool {
	reg := registryPtr.Load()
	if reg == nil {
		return false
	}
	_, ok := reg.Get(instance)
	return ok
}

// DefaultInstance 返回 storage.object_storage.default_instance 解析出的默认实例名，
// 未配置（或组件未初始化）时返回空串。
//
// 用途：业务侧「配置里没写实例名」时判断是「跟随部署方默认」还是「压根没接入对象存储」。
// 不需要区分这两者的调用方无需理会本函数——直接把 UploadRequest.Instance 留空即可，
// Upload 内部会自行回落到默认实例。
func DefaultInstance() string {
	reg := registryPtr.Load()
	if reg == nil {
		return ""
	}
	return reg.Default()
}

// Instances 返回所有已注册的实例名，顺序不保证稳定。
//
// 主要用于校验失败时把可选值打进错误信息，省去到 yaml 里逐段比对的功夫。
func Instances() []string {
	reg := registryPtr.Load()
	if reg == nil {
		return nil
	}
	return reg.Instances()
}

func close() {
	reg := registryPtr.Load()
	if reg == nil {
		return
	}
	reg.Close()
}

// getPrefix 给对象 key 加租户前缀，结果形如 tenant{ns}/{key}。
//
// ⚠️ 已知问题，暂按现状保留（改动会影响线上既有对象的 key 形态，需单独评估）：
//
// 最后那个 "ns" 是「ctx 里没有租户标识时使用的默认命名空间」。它有两处不妥：
//
//  1. 全库唯一。redis / local-cache / mysql 调 FmtByTypeOnCtx 时都不传默认值，
//     即无租户上下文时原样返回 key、不加前缀。只有对象存储会兜底成 "ns"。
//  2. 它对 ns 线并不生效，只作用于 saas 线——而且作用是错的。
//     ns 是独享型部署，pod 设了 TENANT_FALLBACK，FmtByType 第一行的
//     env.HasFallbackTenant() 就短路返回原始 key 了（见 library/env/env.go 的
//     HasFallbackTenant 注释）。真正会走到这个默认值的，是「未设 TENANT_FALLBACK
//     的多租户 pod + ctx 里没有租户标识」的组合，也就是 saas 线丢了租户上下文的
//     调用路径（后台任务、异步 goroutine 等），结果是把 saas 的对象写进 tenantns/ 目录。
//
// 修的方向是把默认命名空间做成 ObjectStorage 的配置项、默认留空（与库内其它
// 存储类型对齐）。在那之前，调用方务必保证 ctx 带着租户标识。
func getPrefix(ctx context.Context, key string) string {

	return namespace.FmtByTypeOnCtx(ctx, namespace.NAMESPACE_TYPE_OSS, key, "ns")
}
