package metadata

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/coderyw/gomicro-util/util/mapstructure"
	"go-micro.dev/v4/metadata"
	"go.uber.org/zap"
)

const _contextMetadataStr = "_context_metadata"

const (

	// CONTEXT_USER_ID context 包含的 key 用户 id
	CONTEXT_USER_ID string = "Userid"
	// CONTEXT_TERMINAL_NAME context 包含的 key 渠道名称
	CONTEXT_TERMINAL_NAME string = "Terminalname"
	// CONTEXT_REQ_IP 请求 ip
	CONTEXT_REQ_IP string = "Reqip"
	// CONTEXT_TERMINAL 渠道
	CONTEXT_TERMINAL string = "Terminal"
	// CONTEXT_VERSION H5版本号
	CONTEXT_VERSION string = "Version"
	// CONTEXT_TURNSTILE_STATUS CF人机验证状态 ca-open-api路由开启中间件才有
	CONTEXT_TURNSTILE_STATUS string = "TurnstileStatus"
	// CONTEXT_TURNSTILE_ERROR CF人机验证错误信息
	CONTEXT_TURNSTILE_ERROR string = "TurnstileError"
	// CONTEXT_APP_VERSION App 版本号 只有 app 端有
	CONTEXT_APP_VERSION string = "AppVersion"
	// CONTEXT_DEVICE_ID 设备号 id
	CONTEXT_DEVICE_ID string = "Deviceid"
	// 链路所属命名空间
	CONTEXT_NAME_SPACE string = "NameSpace"
	// CONTEXT_CUSTOM_PREFIX 自定义的元数据 key 前缀
	CONTEXT_CUSTOM_PREFIX string = "Custom_"
)

// Metadata 元数据，上下文中保存，贯穿整个链路
type Metadata struct {
	UserId       int64  `json:"Userid" metadata:"Userid"`
	ReqIp        string `json:"Reqip" metadata:"Reqip"`
	TerminalName string `json:"Terminalname" metadata:"Terminalname"`
	Terminal     int64  `json:"Terminal" metadata:"Terminal"`
	//Phone           string `json:"Phone" metadata:"Phone"`
	Version         string `json:"Version" metadata:"Version"`
	AppVersion      string `json:"AppVersion" metadata:"AppVersion"`
	TurnstileStatus string `json:"TurnstileStatus" metadata:"TurnstileStatus"`
	TurnstileError  string `json:"TurnstileError" metadata:"TurnstileError"`
	DeviceId        string `json:"Deviceid" metadata:"Deviceid"`
	NameSpace       string `json:"NameSpace" metadata:"NameSpace"`
	// 其他元数据 q
	Other map[string]string `json:"other" metadata:"Other"`
}

// ToMicroMetadata 转换到 micro 的元数据
//
// Parameters:
//
// Returns：
//   - metadata.Metadata
func (md *Metadata) ToMicroMetadata() metadata.Metadata {
	gm := metadata.Metadata{
		CONTEXT_USER_ID:          strconv.FormatInt(md.UserId, 10),
		CONTEXT_TERMINAL_NAME:    md.TerminalName,
		CONTEXT_REQ_IP:           md.ReqIp,
		CONTEXT_TERMINAL:         strconv.FormatInt(md.Terminal, 10),
		CONTEXT_VERSION:          md.Version,
		CONTEXT_APP_VERSION:      md.AppVersion,
		CONTEXT_TURNSTILE_STATUS: md.TurnstileStatus,
		CONTEXT_TURNSTILE_ERROR:  md.TurnstileError,
		CONTEXT_DEVICE_ID:        md.DeviceId,
		CONTEXT_NAME_SPACE:       md.NameSpace,
	}
	if md.Other != nil {
		for k, v := range md.Other {
			gm[k] = v
		}
	}
	return gm
}

// FromMicroMetadata 从 micro 的元数据转换
//
// Parameters:
//   - md2 metadata.Metadata
//
// Returns：
//   - error
func (md *Metadata) FromMicroMetadata(md2 metadata.Metadata) error {
	if err := mapstructure.Decode(md2, md); err != nil {
		return err
	}
	// 处理自定义的元数据
	for k, v := range md2 {
		if strings.HasPrefix(k, string(CONTEXT_CUSTOM_PREFIX)) {
			if md.Other == nil {
				md.Other = make(map[string]string)
			}
			md.Other[k] = v
		}
	}
	return nil
}

// 元数据所有映射
var kv = map[string]struct{}{
	CONTEXT_USER_ID:          {},
	CONTEXT_TERMINAL_NAME:    {},
	CONTEXT_TERMINAL:         {},
	CONTEXT_REQ_IP:           {},
	CONTEXT_VERSION:          {},
	CONTEXT_APP_VERSION:      {},
	CONTEXT_TURNSTILE_STATUS: {},
	CONTEXT_TURNSTILE_ERROR:  {},
	CONTEXT_DEVICE_ID:        {},
	CONTEXT_NAME_SPACE:       {},
}

// GetLogMd 获取日志打印元数据
//
// Parameters:
//   - ctx: context.Context
//
// Returns:
//   - []zap.Field
func GetLogMd(ctx context.Context) []zap.Field {
	val := ctx.Value(_contextMetadataStr)
	if val != nil {
		val := val.(*Metadata)
		zp := make([]zap.Field, 0, 3)
		if val.UserId != 0 {
			zp = append(zp, zap.Int64(string(CONTEXT_USER_ID), val.UserId))
		}
		if len(val.ReqIp) != 0 {
			zp = append(zp, zap.String(string(CONTEXT_REQ_IP), val.ReqIp))
		}
		if len(val.TerminalName) != 0 {
			zp = append(zp, zap.String(string(CONTEXT_TERMINAL_NAME), val.TerminalName))
		}
		if len(val.NameSpace) != 0 {
			zp = append(zp, zap.String(string(CONTEXT_NAME_SPACE), val.NameSpace))
		}
		return zp
	}
	return nil
}

// SetCtxData 设置元数据到 context 中
//
// 支持类型为 string 的元数据
//   - CONTEXT_USER_ID 用户 id
//   - CONTEXT_TERMINAL_NAME 渠道名称
//   - CONTEXT_REQ_IP 请求 ip
//   - CONTEXT_TERMINAL 渠道
//   - CONTEXT_VERSION H5版本号
//   - CONTEXT_APP_VERSION App 版本号
//   - CONTEXT_TURNSTILE_STATUS CF人机验证状态
//   - CONTEXT_TURNSTILE_ERROR CF人机验证错误信息
//   - CONTEXT_DEVICE_ID 设备号 id
//   - CONTEXT_CUSTOM_PREFIX 为前缀的自定义元数据（即自定义元数据的 key 需要以 'Custom_' 开头）
//
// 自定义属性请勿滥用，否则会影响性能（不要传入过多自定义属性。以及不要传入过大的值）
//
// 设置了元数据后，会返回新的 context，可以通过 Get / MustGet 方法去获取对应的元数据值。
//
// Parameters:
//   - ctx: context.Context 上下文
//   - key: string 元数据 key。获取类型为 string 或则自定义 key。自定义 key 需以 Custom_ 开头
//   - value: interface{} 元数据值（）
//
// Returns:
//   - context.Context 返回新的 context。后续需要通过新的 context 去传递
func SetCtxData(ctx context.Context, key string, value interface{}) context.Context {
	// gin 使用特殊的方式获取上下文数据
	if gCtx, ok := ctx.(*gin.Context); ok {
		ctx = gCtx.Request.Context()
		// 写入数据到 Request 的 context 中
		ctx, newCtx := write(ctx, key, value)
		if newCtx {
			// 如果有新数据，则将数据写入 Request 的 context 中
			gCtx.Request = gCtx.Request.WithContext(ctx)
		}
		return ctx
	}
	if ctx == nil {
		ctx = context.TODO()
	}
	ctx, _ = write(ctx, key, value)
	return ctx
}

func write(ctx context.Context, key string, value interface{}) (context.Context, bool) {
	val := ctx.Value(_contextMetadataStr)
	new := false
	if val == nil {
		val = &Metadata{}
		new = true
	}
	data := val.(*Metadata)
	set(data, string(key), value)
	if new {
		// 由于存入的是 map，是引用类型，所以如果已经有值，则没必要再进行生成新的 ctx
		return context.WithValue(ctx, _contextMetadataStr, data), true
	}
	return ctx, false
}

// MustGet 从上下文中必须获取 Metadata数据
//
// 如果获取不到，则会 panic
//
// 支持的 key 如下
//   - CONTEXT_USER_ID 用户 id
//   - CONTEXT_TERMINAL_NAME 渠道名称
//   - CONTEXT_REQ_IP 请求 ip
//   - CONTEXT_TERMINAL 渠道
//   - CONTEXT_VERSION H5版本号
//   - CONTEXT_APP_VERSION App 版本号
//   - CONTEXT_TURNSTILE_STATUS CF人机验证状态
//   - CONTEXT_TURNSTILE_ERROR CF人机验证错误信息
//   - CONTEXT_DEVICE_ID 设备号 id
//   - CONTEXT_CUSTOM_PREFIX 为前缀的自定义元数据
//
// 如果使用自定义的元数据请与生成端协商好类型，同时注意“零值”问题，比如“””这个字符串，如果去转为数字类型，则会报错的。
//
// Parameters:
//   - T: any 支持任意类型泛型
//   - ctx: context.Context 上下文
//   - key: string 获取的 Metadata 数据的 key
//
// Returns:
//   - T 返回元数据值，具体类型参考 Metadata 属性类型（自定义元数据，默认全部返回 string 类型）
func MustGet[T any](ctx context.Context, key string) T {
	var got interface{}
	if val := ctx.Value(_contextMetadataStr); val != nil {
		got = get(val.(*Metadata), string(key))
	}
	if got == nil {
		panic(fmt.Sprintf("ctx got nothing with key: %v", key))
	}
	val, ok := got.(T)
	if ok {
		return val
	}
	return val
}

// Get 从上下文中获取 Metadata数据
//
// 支持的 key 如下
//   - CONTEXT_USER_ID 用户 id
//   - CONTEXT_TERMINAL_NAME 渠道名称
//   - CONTEXT_REQ_IP 请求 ip
//   - CONTEXT_TERMINAL 渠道
//   - CONTEXT_VERSION H5版本号
//   - CONTEXT_APP_VERSION App 版本号
//   - CONTEXT_TURNSTILE_STATUS CF人机验证状态
//   - CONTEXT_TURNSTILE_ERROR CF人机验证错误信息
//   - CONTEXT_DEVICE_ID 设备号 id
//   - CONTEXT_CUSTOM_PREFIX 为前缀的自定义元数据
//
// 如果使用自定义的元数据请与生成端协商好类型，同时注意“零值”问题，比如“””这个字符串，如果去转为数字类型，则会报错的。
//
// Parameters:
//   - T: any 支持任意类型泛型
//   - ctx: context.Context 上下文
//   - key: string 获取的 Metadata 数据的 key
//
// Returns:
//   - T 返回元数据值，具体类型参考 Metadata 属性类型（自定义元数据，默认全部返回 string 类型）
func Get[T any](ctx context.Context, key string) T {
	var got T
	if val := ctx.Value(_contextMetadataStr); val != nil {
		got1 := get(val.(*Metadata), string(key))
		if got1 != nil {
			got = got1.(T)
		}
	}
	return got
}

// MustGetMd 获取全局 Metadata
//
// Parameters:
//   - ctx: context.Context
//
// Returns:
//   - *Metadata
func MustGetMd(ctx context.Context) *Metadata {
	if val := ctx.Value(_contextMetadataStr); val != nil {
		return val.(*Metadata)
	}
	panic(fmt.Sprintf("ctx got nothing"))
}

func GetMd(ctx context.Context) *Metadata {
	if val := ctx.Value(_contextMetadataStr); val != nil {
		return val.(*Metadata)
	}
	return nil
}

// GetMicroMetadata 从上下文获取 micro 的元数据
//
// Parameters:
//   - ctx: context.Context
//
// Returns:
//   - metadata.Metadata go micro 类型元数据
func GetMicroMetadata(ctx context.Context) metadata.Metadata {
	val := ctx.Value(_contextMetadataStr)
	if val == nil {
		return nil
	}
	return val.(*Metadata).ToMicroMetadata()
}

// SetFromMicroMetadata 从micro 的元数据设置到上下文
//
// Parameters:
//   - ctx: context.Context
//   - mmd: metadata.Metadata
//
// Returns:
//   - context.Context
//   - error
func SetFromMicroMetadata(ctx context.Context, mmd metadata.Metadata) (context.Context, error) {
	md := new(Metadata)
	err := md.FromMicroMetadata(mmd)
	if err != nil {
		// 修复 返回出去是空的 context，防止 panic
		return ctx, err
	}
	return context.WithValue(ctx, _contextMetadataStr, md), nil
}

func toString(i interface{}) string {
	if i == nil {
		return ""
	}
	if val, ok := i.(string); ok {
		return val
	}
	return fmt.Sprint(i)
}

// 设置元数据
func set(md *Metadata, key string, value interface{}) {
	if _, ok := kv[key]; ok {
		switch key {
		case CONTEXT_USER_ID:
			md.UserId = value.(int64)
		case CONTEXT_REQ_IP:
			md.ReqIp = value.(string)
		case CONTEXT_TERMINAL_NAME:
			md.TerminalName = value.(string)
		case CONTEXT_TERMINAL:
			md.Terminal = value.(int64)
		case CONTEXT_VERSION:
			md.Version = value.(string)
		case CONTEXT_APP_VERSION:
			md.AppVersion = value.(string)
		case CONTEXT_TURNSTILE_STATUS:
			md.TurnstileStatus = value.(string)
		case CONTEXT_TURNSTILE_ERROR:
			md.TurnstileError = value.(string)
		case CONTEXT_DEVICE_ID:
			md.DeviceId = value.(string)
		case CONTEXT_NAME_SPACE:
			md.NameSpace = value.(string)
		}
	} else if strings.HasPrefix(string(key), string(CONTEXT_CUSTOM_PREFIX)) {
		// 如果类型是自定义类型，则保存为 string
		if md.Other == nil {
			md.Other = make(map[string]string)
		}
		md.Other[string(key)] = toString(value)
	}
}

func get(md *Metadata, key string) interface{} {
	if _, ok := kv[key]; ok {
		switch key {
		case CONTEXT_USER_ID:
			return md.UserId
		case CONTEXT_REQ_IP:
			return md.ReqIp
		case CONTEXT_TERMINAL_NAME:
			return md.TerminalName
		case CONTEXT_TERMINAL:
			return md.Terminal
		case CONTEXT_VERSION:
			return md.Version
		case CONTEXT_APP_VERSION:
			return md.AppVersion
		case CONTEXT_TURNSTILE_STATUS:
			return md.TurnstileStatus
		case CONTEXT_TURNSTILE_ERROR:
			return md.TurnstileError
		case CONTEXT_DEVICE_ID:
			return md.DeviceId
		case CONTEXT_NAME_SPACE:
			return md.NameSpace
		}
	} else if strings.HasPrefix(string(key), string(CONTEXT_CUSTOM_PREFIX)) {
		return md.Other[string(key)]
	}
	return nil
}
