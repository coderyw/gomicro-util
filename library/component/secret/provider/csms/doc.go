// Package csms 华为云 CSMS（DEW 凭据管理）Provider。
//
// 凭证走 SDK 默认链：环境变量 AK/SK → profile → ECS 元数据 → CCE Pod Identity
// （HC_CONTAINER_CREDENTIALS_FULL_URI / HC_CONTAINER_AUTHORIZATION_TOKEN_FILE，开启工作负载身份的 CCE 自动注入），
// 容器内不需要落任何 AK/SK。region 取环境变量 HUAWEICLOUD_SDK_REGION，缺省 ap-southeast-3。
package csms
