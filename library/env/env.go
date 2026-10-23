package env

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/coderyw/gomicro-util/model/cst"
)

// GetEnv 获取运行时环境
//
// Returns：
//   - string 运行时环境值： server-local,server-dev,server-test,server-pre,server-prod
func GetEnv() string {
	return env
}

func GetNacosNamespaceId() string {
	return nacosNamespaceId
}

// IsLocal 判断运行时环境是否本地环境
//
// Returns：
//   - bool true=是，false=不是
func IsLocal() bool {
	return env == cst.RUNTIME_ENV_LOCAL
}

// IsTest 检测当前环境是否是测试环境
func IsTest() bool {
	return env == cst.RUNTIME_ENV_TEST
}

// IsDev 检测当前环境是否是开发环境
//
// Returns：
//   - bool true=是，false=不是
func IsDev() bool {
	return env == cst.RUNTIME_ENV_DEV
}

// GetVersion 获取服务版本
//
// Returns：
//   - string 版本
func GetVersion() string {
	return version
}

// IsRelease 是否是生产环境
//
// Returns：
//   - bool true=是，false=不是
func IsRelease() bool {
	return env == cst.RUNTIME_ENV_PROD
}

// IsPre 是否是预发环境
//
// Returns：
//   - bool true=是，false=不是
func IsPre() bool {
	return env == cst.RUNTIME_ENV_PRE
}

// IsTestAuto 返回当前环境是否是自动化测试环境
func IsTestAuto() bool {
	return env == cst.RUNTIME_ENV_TEST_AUTO
}

// IsPreAuto 返回当前环境是否是自动化预发环境
func IsPreAuto() bool {
	return env == cst.RUNTIME_ENV_PRE_AUTO
}

func IsFormal() bool {
	return IsPre() || IsRelease()
}

// 获取服务 id
func GetId() string {
	return id
}

var (
	env     string
	etcd    string
	nacos   string
	version string
	id      string
	// 服务所属的命名空间
	namespace string
	// 是否开启监听商户命名空间配置
	enableListenTenant bool
	// 兜底租户名（来自环境变量 TENANT_FALLBACK）。非空表示本 pod 配置了兜底租户，数据层走原始命名（不加租户前缀）
	fallbackTenantName string
	// 指定对应的nacos namespace id
	nacosNamespaceId string
)

// SyncEnv 加载环境变量
//
// Returns：
//   - error 返回加载错误信息
func SyncEnv() error {

	if err := checkEnv(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func SetEnv(e string) {
	env = e
}

// GetNamespace 获取当前服务所属命名空间
//
// Returns：
//   - string 命名空间
func GetNamespace() string {
	return namespace
}

// GetEnableListenTenant 获取是否开启监听商户命名空间配置
//
// Returns：
//   - bool true=是，false=不是
func GetEnableListenTenant() bool {
	return enableListenTenant
}

// HasFallbackTenant 当前 pod 是否配置了兜底租户（来自环境变量 TENANT_FALLBACK）
//
// 为 true 时该 pod 数据保持原始命名，数据层不加租户前缀（独享型租户如 NS 的专属部署用）。
//
// Returns：
//   - bool true=已配置兜底租户，false=未配置，走常规多租户前缀逻辑
func HasFallbackTenant() bool {
	return fallbackTenantName != ""
}

// SetFallbackTenantName 设置兜底租户名（主要用于测试注入）
func SetFallbackTenantName(tn string) {
	fallbackTenantName = tn
}

// 检查环境变量
func checkEnv() error {
	if _, err := GetDisableEonidHashTag(); err != nil {
		return err
	}
	if _, err := parseTenantNameCase(tenantNameCase); err != nil {
		return err
	}
	// 运行时环境变量不能为空
	if env == "" {
		return errors.New(fmt.Sprintf("缺少环境变量 - [%s]", cst.ENV_ENVIRON))
	}
	fmt.Printf("env: key=%v, value=%v\n", cst.ENV_ENVIRON, env)

	if env != cst.RUNTIME_ENV_LOCAL { //本地环境不需要校验etcd
		// 当前两个注册中心需要校验
		if etcd == "" {
			if nacos == "" {
				return errors.New(fmt.Sprintf("缺少环境变量 - [%s]", cst.ENV_NACOS_URLS))
			} else {
				if nacosNamespaceId == "" {
					return fmt.Errorf("缺少环境变量 - [%s]", cst.ENV_NACOS_NAMESPACE_ID)
				}
				fmt.Printf("env: key=%v, value=%v\nkey=%v, value=%v\n",
					cst.ENV_NACOS_URLS, nacos,
					cst.ENV_NACOS_NAMESPACE_ID, nacosNamespaceId)
			}
		} else {
			fmt.Printf("env: key=%v, value=%v\n", cst.ENV_ETCD_URLS, etcd)
		}
	}

	return nil
}

func init() {
	// 获取运行时环境
	env = os.Getenv(cst.ENV_ENVIRON)
	// 获取 etcd 地址
	etcd = os.Getenv(cst.ENV_ETCD_URLS)
	// 获取 nacos 地址
	nacos = os.Getenv(cst.ENV_NACOS_URLS)
	es := os.Getenv(cst.ENV_ENABLE_LISTEN_TENANT)
	if es == "false" {
		enableListenTenant = false
	} else {
		enableListenTenant = true
	}
	// 获取服务版本
	version = os.Getenv(cst.ENV_SERVER_VER)
	if len(version) == 0 {
		version = "latest"
	}
	namespace = os.Getenv(cst.ENV_NAMESPACE)
	// 获取兜底租户名
	fallbackTenantName = os.Getenv(cst.ENV_TENANT_FALLBACK)
	// 设置当前服务 id
	if id = os.Getenv(cst.ENV_ID); len(id) == 0 {
		id = uuid.NewString()
		os.Setenv(cst.ENV_ID, id)
	}
	// 获取namespace id
	nacosNamespaceId = os.Getenv(cst.ENV_NACOS_NAMESPACE_ID)
}

// GetOsEnv 获取系统环境变量
//
// 支持获取不到时候设置默认值
//
// Parameters:
//   - key string 环境变量名称
//   - defaultValue ...string 默认值
//
// Returns：
//   - string 环境变量值
func GetOsEnv(key string, defaultValue ...string) string {
	v := os.Getenv(key)
	if v == "" && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return v
}
