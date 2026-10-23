// Package metadata 元数据
//
// 数据通过 context 传递，贯穿整个链路，支持 gin.Context 和 go-micro.dev/v4/metadata.Metadata
//
// 支持参数：
//   - Userid: int64 用户 id
//   - Terminalname: string 渠道名称
//   - Reqip: string 请求 ip
//   - Terminal: int64 渠道
//   - Version: string H5版本号
//   - AppVersion: string App版本号
//   - Deviceid: string 设备号 id
//   - TurnstileStatus: string CF人机验证状态
//   - TurnstileError: string CF人机验证错误信息
//   - NameSpace: string 链路所属租户
//   - Custom_: string 自定义参数
//
// 自定义参数请通过 Custom_ 前缀进行设置值，否则是不允许传递的。同时请不要传递过多和多大的自定义数据，防止影响传输效率
//
// 文档：docs/components/http.md
package metadata
