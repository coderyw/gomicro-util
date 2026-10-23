package tenant

import (
	"context"
	"strings"
)

type Tenant struct {
	TenantId   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	Namespace  string `json:"namespace"`
	Type       string `json:"type"`
	Status     string `json:"status"`

	Ctx context.Context `json:"-"`
}

// 云厂商类型
type CloudType int

// 云厂商
//
// CLOUD_TYPE_OTHER 占据零值位：未显式赋值/漏配的 CloudType 表示"未识别"，而不会静默落到某个具体厂商
const (
	CLOUD_TYPE_OTHER   CloudType = iota // 云厂商 other/未识别（零值）
	CLOUD_TYPE_HW                       // 云厂商 华为
	CLOUD_TYPE_AWS                      // 云厂商 aws
	CLOUD_TYPE_AZURE                    // 云厂商 azure
	CLOUD_TYPE_GOOGLE                   // 云厂商 google
	CLOUD_TYPE_TENCENT                  // 云厂商 tencent
	CLOUD_TYPE_ALIYUN                   // 云厂商 aliyun
)

// String 返回云厂商名称，未识别的值返回 other
func (c CloudType) String() string {
	switch c {
	case CLOUD_TYPE_HW:
		return "huawei"
	case CLOUD_TYPE_AWS:
		return "aws"
	case CLOUD_TYPE_AZURE:
		return "azure"
	case CLOUD_TYPE_GOOGLE:
		return "google"
	case CLOUD_TYPE_TENCENT:
		return "tencent"
	case CLOUD_TYPE_ALIYUN:
		return "aliyun"
	default:
		return "other"
	}
}

// ParseCloudType 按名称解析云厂商类型（大小写不敏感），与 String 互逆
//
// Parameters:
//   - name string 云厂商名称，如 huawei、aws
//
// Returns：
//   - CloudType 未识别的名称返回 CLOUD_TYPE_OTHER
func ParseCloudType(name string) CloudType {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "huawei", "hw":
		return CLOUD_TYPE_HW
	case "aws":
		return CLOUD_TYPE_AWS
	case "azure":
		return CLOUD_TYPE_AZURE
	case "google", "gcp":
		return CLOUD_TYPE_GOOGLE
	case "tencent":
		return CLOUD_TYPE_TENCENT
	case "aliyun":
		return CLOUD_TYPE_ALIYUN
	default:
		return CLOUD_TYPE_OTHER
	}
}
