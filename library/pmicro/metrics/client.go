// Package metrics
// @Author: yinwei
// @File: client
// @Version: 1.0.0
// @Date: 2026/8/12 14:00

package metrics

import (
	"sort"
	"strings"
	"sync/atomic"

	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
)

// Tags 是一次打点携带的维度集合；key 是维度名（如 endpoint/ret），value 是该维度的取值。
// Tag value 只能来自可枚举白名单，不得把请求原文（userId、wagerId、原始 URL path）当 tag，
// 否则时间序列数量爆炸（高基数问题），拖垮后端存储和查询。
type Tags map[string]string

// Client 是业务代码打点的唯一入口，屏蔽指标声明、label 顺序、后端选择等所有实现细节。
// 指标注册在 prometheus 全局 registry：/metrics 拉取端点可见，
// 配置开启 Truewatch 上报后随框架指标一起推送，业务代码无需关心通道。
type Client interface {
	// Inc 计数类打点：请求量、失败次数、金额流水（分）等只增不减的值，value 必须 >= 0
	Inc(name string, tags Tags, value float64)
	// Latency 耗时/分布类打点：接口耗时、下游调用耗时等需要看分位数的值，单位秒
	Latency(name string, tags Tags, value float64)
}

// businessDisabled 业务打点总开关（对应配置 metrics.disable_business）。
// 关闭后 Inc/Latency 直接返回，作为高基数误用等场景的运维紧急开关。
var businessDisabled atomic.Bool

// NewClient 用当前服务的唯一标识初始化一个 Client；service 会作为公共 tag（service=<name>）
// 附加到这个 Client 打的每一条指标上。
// NewClient 本身不会失败：即使框架未初始化或上报未配置，打点也只是安全的内存操作。
func NewClient(service string) Client {
	return &client{service: service}
}

type client struct {
	service string
}

func (c *client) Inc(name string, tags Tags, value float64) {
	defer recoverAndLog("Inc", name)
	if businessDisabled.Load() {
		return
	}
	if value < 0 {
		log.Warn("metrics: Inc 收到负值，已丢弃", zap.String("name", name), zap.Float64("value", value))
		return
	}
	keys, values := c.sortedTagKeysAndValues(tags)
	vec := counterVecFor(sanitizeName(name), keys)
	if vec == nil {
		return
	}
	vec.WithLabelValues(values...).Add(value)
}

func (c *client) Latency(name string, tags Tags, value float64) {
	defer recoverAndLog("Latency", name)
	if businessDisabled.Load() {
		return
	}
	keys, values := c.sortedTagKeysAndValues(tags)
	vec := histogramVecFor(sanitizeName(name), keys)
	if vec == nil {
		return
	}
	vec.WithLabelValues(values...).Observe(value)
}

// sortedTagKeysAndValues 把业务传入的 tags 与 c.service 合并，按 key 排序后拆成两个平行数组，
// 供惰性注册（keys 决定 label 维度）和打点（values 按同一顺序传给 WithLabelValues）复用。
func (c *client) sortedTagKeysAndValues(tags Tags) (keys, values []string) {
	merged := make(Tags, len(tags)+1)
	for k, v := range tags {
		merged[sanitizeName(k)] = v
	}
	merged["service"] = c.service

	keys = make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	values = make([]string, len(keys))
	for i, k := range keys {
		values[i] = merged[k]
	}
	return keys, values
}

// sanitizeName 把指标名/label 名里 prometheus 不允许的字符（如连字符）替换成下划线
func sanitizeName(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == ':':
			return r
		default:
			return '_'
		}
	}, s)
}

// recoverAndLog 打点路径的最后一道弱依赖保险：任何内部 panic 只记日志，绝不进业务请求路径
func recoverAndLog(op, name string) {
	if r := recover(); r != nil {
		log.Error("metrics: 打点 panic 已恢复", zap.String("op", op), zap.String("name", name), zap.Any("panic", r))
	}
}
