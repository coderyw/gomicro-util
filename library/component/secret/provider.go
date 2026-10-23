package secret

import (
	"context"
	"fmt"
	"sync"
)

// Provider 按 scheme 提供凭据原文（整份 SecretString），键提取由组件统一完成
type Provider interface {
	// Scheme 引用前缀，如 csms / awssm
	Scheme() string
	// Fetch 拉取凭据原文；凭据不存在/无权限等确定性失败请用 NonRetryable 包装
	Fetch(ctx context.Context, name string) (string, error)
}

var (
	providerMu sync.RWMutex
	providers  = map[string]Provider{}
	// schemeList 已注册 scheme 的快照，Register 时重建；解析每个标量都要遍历，避免每次分配
	schemeList []string
)

// Register 注册 Provider；同 scheme 重复注册以后者为准
func Register(p Provider) {
	if p == nil || p.Scheme() == "" {
		panic("secret: register nil provider or empty scheme")
	}
	providerMu.Lock()
	defer providerMu.Unlock()
	providers[p.Scheme()] = p
	rebuildSchemeList()
}

// rebuildSchemeList 重建快照（调用方持写锁）；整片替换而非原地修改，已取走的旧快照不受影响
func rebuildSchemeList() {
	list := make([]string, 0, len(providers))
	for s := range providers {
		list = append(list, s)
	}
	schemeList = list
}

func getProvider(scheme string) (Provider, error) {
	providerMu.RLock()
	defer providerMu.RUnlock()
	p, ok := providers[scheme]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownScheme, scheme)
	}
	return p, nil
}

// registeredSchemes 返回只读快照，调用方不得修改
func registeredSchemes() []string {
	providerMu.RLock()
	defer providerMu.RUnlock()
	return schemeList
}

// Schemes 已注册的 scheme 列表（诊断用，返回副本）
func Schemes() []string {
	return append([]string(nil), registeredSchemes()...)
}
