package namespace

import (
	"context"

	mul_tenant "github.com/coderyw/gomicro-util/internal/mul-tenant"
	"github.com/coderyw/gomicro-util/model/tenant"
)

// GetTenantsInNamespace 获取当前服务命名空间下所有租户
//
// 读线程安全，写线程不安全。
//
// Returns：
//   - []context.Context 租户上下文列表（租户未加载时兜底返回单个 context.Background()，保证调用方至少执行一次）
func GetTenantsInNamespace() []context.Context {
	ts := mul_tenant.GetTenants()
	if len(ts) == 0 {
		return []context.Context{context.Background()}
	}
	arr := make([]context.Context, len(ts))
	for i, v := range ts {
		arr[i] = v.Ctx
	}
	return arr
}

// GetAllTenantsInfo 获取所有租户信息
//
// Returns：
//   - []tenant.Tenant 内部租户列表的浅拷贝，可安全持有与遍历；租户未加载时返回空切片，调用方需判空
//     （注意与 GetTenantsInNamespace 的兜底行为不同，后者为空时会返回单个 Background 上下文）
func GetAllTenantsInfo() []tenant.Tenant {
	return mul_tenant.GetTenants()
}

// OnTenantsChange 注册租户列表变更监听器
//
// 仅当运行期租户配置变更导致列表内容变化时触发（启动首次加载不触发；
// 需要初始状态的调用方注册后应自行读取一次 GetAllTenantsInfo 做对账）。
// 回调在 nacos 监听回调 goroutine 内同步依次执行，应快速返回、不可阻塞；
// 入参为变更后租户列表的浅拷贝，可安全持有。监听器注册后不可注销。
func OnTenantsChange(fn func([]tenant.Tenant)) {
	mul_tenant.RegisterChangeListener(fn)
}
