package mul_tenant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/tenant"
	"go.uber.org/zap"
)

const (
	// 商户配置命名空间
	ENV_TENANT_CFG_NACOS_NAMESPACE = "TENANT_CFG_NACOS_NAMESPACE"
	ENV_TENANT_CFG_NACOS_DATAID    = "TENANT_CFG_NACOS_DATAID"
	ENV_TENANT_CFG_NACOS_GROUP     = "TENANT_CFG_NACOS_GROUP"
	// 是否开启多租户模式 默认开启，如果配置 false，则不会区分商户
	ENV_ENABLE_MUL_TENANT = "ENABLE_MUL_TENANT"
)

const (
	// TenantStatusActive 租户状态：启用
	TenantStatusActive = "active"
	// TenantTypeExclusive 租户类型：独享
	TenantTypeExclusive = "exclusive"
)

var (
	tenantCfgNamespace string = "tenant"
	tenantCfgDataId    string = "saas-tenant-meta-mapping"
	tenantCfgGroup     string = "DEFAULT_GROUP"
	//enableMulTenant    bool   = true //默认是 true
)

// 默认租户，用于未配置商户命名空间的情况，租户信息需要配置为空，配合HasFallbackTenant使用
// TenantId 为占位值，不对应真实租户
var defaultTenant = tenant.Tenant{
	TenantId:   "0",
	TenantName: "",
	Namespace:  "",
	Type:       TenantTypeExclusive,
	Status:     TenantStatusActive,
	Ctx:        context.TODO(),
}

func init() {
	tenantCfgNamespace = env.GetOsEnv(ENV_TENANT_CFG_NACOS_NAMESPACE, tenantCfgNamespace)
	tenantCfgDataId = env.GetOsEnv(ENV_TENANT_CFG_NACOS_DATAID, tenantCfgDataId)
	tenantCfgGroup = env.GetOsEnv(ENV_TENANT_CFG_NACOS_GROUP, tenantCfgGroup)
	// 做一个本地调试
	if env.IsLocal() {
		// 如果是本地运行环境，则通过环境变量获取商户配置
		as := os.Getenv("NAMESPACE_TENANT")
		if as != "" {
			//使用，分割
			arr := strings.Split(as, ",")
			id := 0
			for _, v := range arr {
				if v == "" {
					continue
				}
				tenants = append(tenants, tenant.Tenant{
					TenantId:   strconv.Itoa(id),
					TenantName: v,
					Namespace:  v,
					Type:       TenantTypeExclusive,
					Status:     TenantStatusActive,
					Ctx:        metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, v),
				})
				id++
			}
		}
		if len(tenants) == 0 {
			tenants = append(tenants, defaultTenant)
		}
	}
}

var (
	tenants []tenant.Tenant // 租户列表
	lock    sync.RWMutex
)

var (
	changeListeners []func([]tenant.Tenant) // 租户列表变更监听器
	listenerLock    sync.RWMutex
)

// RegisterChangeListener 注册租户列表变更监听器
//
// 仅当运行期配置变更导致租户列表内容变化时触发（启动时的首次加载通常早于注册，不会触发；
// 需要初始状态的调用方注册后应自行读取一次 GetTenants 做对账）。
// 回调在 nacos 监听回调 goroutine 内同步依次执行，应快速返回、不可阻塞；
// 入参为变更后租户列表的浅拷贝，可安全持有。监听器注册后不可注销。
func RegisterChangeListener(fn func([]tenant.Tenant)) {
	if fn == nil {
		return
	}
	listenerLock.Lock()
	defer listenerLock.Unlock()
	changeListeners = append(changeListeners, fn)
}

// notifyChange 通知所有监听器租户列表已变更，需在持有 lock 之外调用
func notifyChange(ts []tenant.Tenant) {
	listenerLock.RLock()
	fns := make([]func([]tenant.Tenant), len(changeListeners))
	copy(fns, changeListeners)
	listenerLock.RUnlock()
	for _, fn := range fns {
		// 每个监听器一份拷贝，避免相互影响
		arr := make([]tenant.Tenant, len(ts))
		copy(arr, ts)
		// 隔离单个监听器的 panic：回调在 nacos 监听 goroutine 内执行，
		// 该 goroutine 无 recover，不能让业务侧监听器打崩配置监听
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error("租户变更监听器 panic", zap.Any("recover", r))
				}
			}()
			fn(arr)
		}()
	}
}

// tenantsEqual 判断两个租户列表内容是否一致（无序比较，nacos 配置是 map、解析顺序随机）
func tenantsEqual(a, b []tenant.Tenant) bool {
	if len(a) != len(b) {
		return false
	}
	keys := make(map[string]int, len(a))
	for _, t := range a {
		keys[tenantKey(t)]++
	}
	for _, t := range b {
		k := tenantKey(t)
		keys[k]--
		if keys[k] < 0 {
			return false
		}
	}
	return true
}

func tenantKey(t tenant.Tenant) string {
	return t.TenantId + "|" + t.TenantName + "|" + t.Namespace + "|" + t.Type + "|" + t.Status
}

// GetTenants 获取当前命名空间所在的商户
//
// Returns：
//   - []tenant.Tenant 商户列表（含各商户携带 namespace metadata 的 Ctx）。
//     返回内部列表的浅拷贝，调用方可安全持有与遍历；租户未加载时返回空切片
func GetTenants() []tenant.Tenant {
	lock.RLock()
	defer lock.RUnlock()
	arr := make([]tenant.Tenant, len(tenants))
	copy(arr, tenants)

	return arr
}

// StartListen 监听命名空间商户配置
//
//	不需要手动调用，会在服务启动时候自动执行。可以通过环境变量 ENABLE_LISTEN_TENANT=false去关闭，默认是开启
//
// Parameters:
//   - cfg config.RegistryCfg
//
// Returns：
//   - error
func StartListen(cfg config.RegistryCfg) error {
	if cfg.Hosts == nil {
		return errors.New("NACOS_URLS 有误")
	}
	cfg.NameId = tenantCfgNamespace
	if !env.GetEnableListenTenant() {
		return nil
	}
	clientConfig := constant.ClientConfig{
		NamespaceId:          cfg.NameId,
		Password:             cfg.Pwd,
		Username:             cfg.Account,
		NotLoadCacheAtStart:  true,
		UpdateCacheWhenEmpty: true,
		LogLevel:             "error",
	}
	serverConfigs := make([]constant.ServerConfig, 0)
	contextPath := "/nacos"
	for _, addr := range cfg.Hosts {
		// check we have a port
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return err
		}

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

	clientConfig.TimeoutMs = uint64(time.Millisecond * 3)
	var err error
	client, err := clients.CreateConfigClient(map[string]interface{}{
		constant.KEY_SERVER_CONFIGS: serverConfigs,
		constant.KEY_CLIENT_CONFIG:  clientConfig,
	})
	if err != nil {
		return err
	}
	// 加载一次配置
	if err = loadTenantCfg(client); err != nil {
		return err
	}

	// 开启监听
	return listen(client)
}

func loadTenantCfg(client config_client.IConfigClient) error {
	// 获取一次配置
	data, err := client.GetConfig(vo.ConfigParam{
		DataId: tenantCfgDataId,
		Group:  tenantCfgGroup,
	})
	if err != nil {
		return err
	}
	return parseTenantCfg(data)
}

func parseTenantCfg(data string) error {
	//过滤
	var ts []tenant.Tenant
	ns := env.GetNamespace()
	if data == "" {
		log.Warn("商户命名空间配置为空")
	} else {
		//解析
		obj := make(map[string]tenant.Tenant)
		err := json.Unmarshal([]byte(data), &obj)
		if err != nil {
			log.Error("解析商户命名空间错误", zap.Error(err))
			return err
		}
		for _, v := range obj {
			// 租户所在命名空间与当前 pod 配置的一致，且状态是 active才保留
			if v.Namespace == ns && v.Status == TenantStatusActive {
				v.Ctx = metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, v.TenantName)
				ts = append(ts, v)
			}
		}
	}
	if len(ts) == 0 {
		// 未匹配到租户（无配置或全部非 active）时，需要判断是否开启了空租户配置
		if env.HasFallbackTenant() {
			ts = append(ts, defaultTenant)
		} else {
			// 未匹配到租户，同时未开启空租户配置 需要返回错误
			return fmt.Errorf("命名空间 %s 未匹配到 active 租户且未配置兜底租户", ns)
		}
	}
	lock.Lock()
	changed := !tenantsEqual(tenants, ts)
	tenants = ts
	lock.Unlock()
	log.Info("商户命名空间加载完成", zap.Any("tenants", ts))
	// 列表内容有变化才通知监听器（在锁外回调，监听器内可安全调用 GetTenants）
	if changed {
		notifyChange(ts)
	}
	return nil
}

func listen(client config_client.IConfigClient) error {
	return client.ListenConfig(vo.ConfigParam{
		DataId: "saas-tenant-meta-mapping",
		Group:  "DEFAULT_GROUP",
		OnChange: func(namespace, group, dataId, data string) {
			log.Info("NacosTenantChange", zap.String("namespace", namespace),
				zap.String("group", group),
				zap.String("dataId", dataId))
			if err := parseTenantCfg(data); err != nil {
				log.Error("解析商户配置错误", zap.String("data", data), zap.Error(err))
				// todo 根据配置 确定是否需要重启服务
			}
		},
	})
}
