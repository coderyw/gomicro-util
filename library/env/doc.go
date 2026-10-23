// Package env 读取并校验服务运行所需的环境变量。
//
// 环境变量在包 init 时读取并缓存，修改后需重启进程；SyncEnv 只做校验，不会重新读取。
// pmicro 启动时会自动调用 SyncEnv。
//
// 常用方法：
//   - GetEnv / IsLocal / IsDev / IsTest / IsPre / IsRelease / IsFormal：运行环境（ENVIRON）
//   - GetId / GetVersion：实例 ID（ID）与服务版本（SERVER_VERSION）
//   - GetNamespace / HasFallbackTenant / GetTenantNameCase：多租户部署模式
//   - GetDisableEonidHashTag：雪花 ID Redis 前缀（DISABLE_EONID_HASHTAG）
//   - GetCloudType：云厂商（CLOUD_TYPE 或自动探测）
//   - GetOsEnv：读取任意环境变量，支持默认值
//
// 注意：IsTest、IsPre 不包含 server-test-auto、server-pre-auto；IsFormal 不包含 server-roadshow。
//
// 文档：docs/environment.md
package env
