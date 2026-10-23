package namespace

import (
	"context"
	"fmt"
	"strings"

	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/metadata"
)

// 命名格式类型
type NamespaceFmt string

const (
	NAMESPACE_TYPE_MYSQL  NamespaceFmt = "ns_type_mysql"  // 命名格式类型 mysql
	NAMESPACE_TYPE_REDIS  NamespaceFmt = "ns_type_redis"  // 命名格式类型 redis
	NAMESPACE_TYPE_KAFKA  NamespaceFmt = "ns_type_kafka"  // 命名格式类型 kafka
	NAMESPACE_TYPE_COMMON NamespaceFmt = "ns_type_common" // 命名格式类型 common 通用类型
	NAMESPACE_TYPE_OSS    NamespaceFmt = "ns_type_oss"    // 命名格式类型 oss
	splitMysql                         = "_"              //mysql 使用_区分
	splitKafka                         = "_"              //kafka 使用_区分
	splitRedis                         = ":"              //redis 使用:区分
	splitCommon                        = ":"              //common 使用:区分
	splitOss                           = "/"              // oss 使用/区分
	prefix                             = "tenant"         // 租户前缀
)

var (
	mysqlFmt  = fmt.Sprintf("%%v%v%%v", splitMysql)
	redisFmt  = fmt.Sprintf("%v%%v%v%%v", prefix, splitRedis)
	kafkaFmt  = fmt.Sprintf("%v%%v%v%%v", prefix, splitKafka)
	commonFmt = fmt.Sprintf("%v%%v%v%%v", prefix, splitCommon)
	ossFmt    = fmt.Sprintf("%v%%v%v%%v", prefix, splitOss)

	splitMap = map[NamespaceFmt]string{
		NAMESPACE_TYPE_MYSQL:  splitMysql,
		NAMESPACE_TYPE_REDIS:  splitRedis,
		NAMESPACE_TYPE_KAFKA:  splitKafka,
		NAMESPACE_TYPE_COMMON: splitCommon,
		NAMESPACE_TYPE_OSS:    splitOss,
	}

	fmtMap = map[NamespaceFmt]string{
		NAMESPACE_TYPE_MYSQL:  mysqlFmt,
		NAMESPACE_TYPE_REDIS:  redisFmt,
		NAMESPACE_TYPE_KAFKA:  kafkaFmt,
		NAMESPACE_TYPE_COMMON: commonFmt,
		NAMESPACE_TYPE_OSS:    ossFmt,
	}
)

// FmtByType 根据 t 类型，去格式化 fmtKey
//
// Parameters:
//   - t NamespaceFmt
//   - nsKey string
//   - fmtKey string
//
// Returns：
//   - string
func FmtByType(t NamespaceFmt, nsKey string, fmtKey string) string {
	// 配置了兜底租户（环境变量 TENANT_FALLBACK 已设）的 pod，数据保持原始命名，一律不加租户前缀
	if env.HasFallbackTenant() {
		return fmtKey
	}
	if nsKey == "" {
		return fmtKey
	}
	if fmtStr, ok := fmtMap[t]; ok {
		return fmt.Sprintf(fmtStr, NormalizeTenantName(nsKey), fmtKey)
	}
	return fmtKey
}

func FmtByTypeOnCtx(ctx context.Context, t NamespaceFmt, fmtKey string, defaultNs ...string) string {
	nsKey := metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE)
	if nsKey == "" {
		if len(defaultNs) == 0 {
			return fmtKey
		}
		nsKey = defaultNs[0]
	}
	return FmtByType(t, nsKey, fmtKey)
}

// GetNameSpaceFromKey 从 resetKey 获取 namespace 名称
//
// 比如库名 ok_test.ca_user, 将会返回 ok  test.ca
//
// 注意如果是去从数据库的库名取商户可能会出现 Bug，因为数据库的库名没有商户标识，无法确认拆分出的是否是商户标识
//
// Parameters:
//   - t NamespaceFmt
//   - resetKey string
//
// Returns：
//   - string 重置后的 namespace key
func GetNameSpaceFromKey(t NamespaceFmt, resetKey string) string {
	splitStr, ok := splitMap[t]
	if ok {
		switch t {
		case NAMESPACE_TYPE_MYSQL:
			arr := strings.SplitN(resetKey, splitStr, 2)
			if len(arr) < 2 {
				return ""
			}
			return arr[0]
		default:
			if strings.HasPrefix(resetKey, prefix) {
				arr := strings.SplitN(resetKey, splitStr, 2)
				if len(arr) < 2 {
					return ""
				}
				return strings.TrimPrefix(arr[0], prefix)
			}
		}
	}

	return ""
}

// GetOriginKey 获取除去了商户信息的原始 key
//
// Parameters:
//   - t NamespaceFmt
//   - resetKey string
//
// Returns：
//   - string
func GetOriginKey(t NamespaceFmt, resetKey string) string {
	_, ori := parseOriginKey(t, resetKey)
	return ori
}

func parseOriginKey(t NamespaceFmt, resetKey string) (string, string) {
	splitStr, ok := splitMap[t]
	if ok {
		switch t {
		case NAMESPACE_TYPE_MYSQL:
			arr := strings.SplitN(resetKey, splitStr, 2)
			if len(arr) < 2 {
				return "", resetKey
			}
			return strings.TrimPrefix(arr[0], prefix), arr[1]
		default:
			if strings.HasPrefix(resetKey, prefix) {
				arr := strings.SplitN(resetKey, splitStr, 2)
				if len(arr) < 2 {
					return "", resetKey
				}
				return strings.TrimPrefix(arr[0], prefix), arr[1]
			}
		}
	}

	return "", resetKey

	//switch t {
	//case NAMESPACE_TYPE_MYSQL:
	//	arr := strings.SplitN(resetKey, splitMysql, 2)
	//	if len(arr) < 2 {
	//		return "", resetKey
	//	}
	//	return strings.TrimPrefix(arr[0], prefix), arr[1]
	//case NAMESPACE_TYPE_KAFKA:
	//	if strings.HasPrefix(resetKey, prefix) {
	//		arr := strings.SplitN(resetKey, splitMysql, 2)
	//		if len(arr) < 2 {
	//			return "", resetKey
	//		}
	//		return strings.TrimPrefix(arr[0], prefix), arr[1]
	//	}
	//case NAMESPACE_TYPE_REDIS:
	//	if strings.HasPrefix(resetKey, prefix) {
	//		arr := strings.SplitN(resetKey, splitRedis, 2)
	//		if len(arr) < 2 {
	//			return "", resetKey
	//		}
	//		return strings.TrimPrefix(arr[0], prefix), arr[1]
	//	}
	//case NAMESPACE_TYPE_COMMON:
	//	if strings.HasPrefix(resetKey, prefix) {
	//		arr := strings.SplitN(resetKey, splitCommon, 2)
	//		if len(arr) < 2 {
	//			return "", resetKey
	//		}
	//		return strings.TrimPrefix(arr[0], prefix), arr[1]
	//	}
	//}

}

// GetCtxAndOriginKey 从商户 key 中获取商户上下文和原始 key
//
// 如：tenantok:32423这个rediskey，会返回带有 ok 的上下文，和 32423 的原始 key
//
// Parameters:
//   - ctx context.Context
//   - t NamespaceFmt
//   - resetKey string
//
// Returns：
//   - context.Context 商户上下文
//   - string 原始 key
func GetCtxAndOriginKey(ctx context.Context, t NamespaceFmt, resetKey string) (context.Context, string) {
	if ctx == nil {
		ctx = context.Background()
	}
	ns, ori := parseOriginKey(t, resetKey)
	if ns == "" {
		return ctx, ori
	}
	return metadata.SetCtxData(ctx, metadata.CONTEXT_NAME_SPACE, ns), ori
}

// GetNameSpaceCtxFromKey 从 resetKey 获取 携带或者不携带 namespace 的 context
//
// Parameters:
//   - ctx context.Context 父 context，可空
//   - t NamespaceFmt 命名空间类型
//   - resetKey string
//
// Returns：
//   - context.Context
func GetNameSpaceCtxFromKey(ctx context.Context, t NamespaceFmt, resetKey string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ns, _ := parseOriginKey(t, resetKey)
	if ns == "" {
		return ctx
	}
	return metadata.SetCtxData(ctx, metadata.CONTEXT_NAME_SPACE, ns)
}
