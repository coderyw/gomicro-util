package env

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/coderyw/gomicro-util/model/cst"
	"github.com/coderyw/gomicro-util/model/tenant"
)

var (
	cloudType     tenant.CloudType
	cloudTypeOnce sync.Once

	// DMI 信息挂载路径，测试时可注入
	dmiSysPath = "/sys/class/dmi/id"
)

// GetCloudType 获取当前 pod 所在云厂商
//
// 探测顺序（结果在首次调用后缓存）：
//  1. 环境变量 CLOUD_TYPE，显式配置优先，云迁移或特殊部署时用于人工覆盖
//  2. DMI/SMBIOS 硬件厂商信息（/sys/class/dmi/id，容器内可读、无需网络与权限）
//  3. IMDS 元数据服务探测（部分老实例 DMI 无厂商标识时兜底；本地环境跳过；
//     集群若封禁 pod 访问 IMDS 则各探测快速失败）
//  4. 均未识别时返回 CLOUD_TYPE_OTHER
//
// Returns：
//   - tenant.CloudType 云厂商类型
func GetCloudType() tenant.CloudType {
	cloudTypeOnce.Do(func() {
		cloudType = detectCloudType()
	})
	return cloudType
}

func detectCloudType() tenant.CloudType {
	// 1. 显式配置优先
	if v := os.Getenv(cst.ENV_CLOUD_TYPE); v != "" {
		ct := tenant.ParseCloudType(v)
		fmt.Printf("cloud type: key=%v, value=%v, parsed=%v\n", cst.ENV_CLOUD_TYPE, v, ct)
		return ct
	}
	// 2. DMI 硬件信息探测
	if ct, ok := detectCloudTypeByDMI(); ok {
		fmt.Printf("cloud type: detected by dmi, value=%v\n", ct)
		return ct
	}
	// 3. IMDS 元数据服务探测（本地环境无 IMDS，跳过以免探测超时拖慢启动）
	if !IsLocal() {
		if ct, ok := detectCloudTypeByIMDS(); ok {
			fmt.Printf("cloud type: detected by imds, value=%v\n", ct)
			return ct
		}
	}
	// 4. 兜底
	fmt.Printf("cloud type: 未配置 %v 且 DMI/IMDS 探测未识别，返回 %v\n", cst.ENV_CLOUD_TYPE, tenant.CLOUD_TYPE_OTHER)
	return tenant.CLOUD_TYPE_OTHER
}

// detectCloudTypeByDMI 通过 DMI/SMBIOS 硬件厂商信息识别云厂商
//
// 虚拟机厂商信息经 sysfs 暴露，容器内通常可直接读取（cloud-init 的 ds-identify 同类做法）。
// 逐个文件读取失败时忽略（受限运行时可能读不到），全部未命中返回 false 交由上层兜底。
func detectCloudTypeByDMI() (tenant.CloudType, bool) {
	// 各厂商标识可能出现在不同字段（如华为在 chassis_asset_tag，AWS Nitro 在 sys_vendor）
	files := []string{"sys_vendor", "product_name", "board_vendor", "chassis_vendor", "chassis_asset_tag", "bios_vendor", "bios_version"}
	var sb strings.Builder
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(dmiSysPath, f))
		if err != nil {
			continue
		}
		sb.WriteString(strings.ToLower(string(b)))
		sb.WriteByte('\n')
	}
	s := sb.String()
	switch {
	case strings.Contains(s, "huawei"):
		return tenant.CLOUD_TYPE_HW, true
	case strings.Contains(s, "amazon"):
		return tenant.CLOUD_TYPE_AWS, true
	case strings.Contains(s, "microsoft"):
		return tenant.CLOUD_TYPE_AZURE, true
	case strings.Contains(s, "google"):
		return tenant.CLOUD_TYPE_GOOGLE, true
	case strings.Contains(s, "tencent"):
		return tenant.CLOUD_TYPE_TENCENT, true
	case strings.Contains(s, "alibaba"), strings.Contains(s, "aliyun"):
		return tenant.CLOUD_TYPE_ALIYUN, true
	}
	// Xen 时代的老 EC2 实例：uuid 以 ec2 开头（product_uuid 可能仅 root 可读，忽略错误）
	for _, f := range []string{"product_uuid", "board_asset_tag"} {
		b, err := os.ReadFile(filepath.Join(dmiSysPath, f))
		if err != nil {
			continue
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(b))), "ec2") {
			return tenant.CLOUD_TYPE_AWS, true
		}
	}
	return tenant.CLOUD_TYPE_OTHER, false
}

// imdsProbe 单个云厂商的元数据服务探测项
type imdsProbe struct {
	cloud  tenant.CloudType
	method string
	url    string
	header map[string]string
}

// imdsProbeList IMDS 探测列表，按序探测、命中即返回；测试时可注入
//
// 顺序说明：AWS 用 IMDSv2 token 接口（PUT）识别，放最前；Azure/GCP 靠专属 header 网关控制，误报率低；
// 阿里云/腾讯云走各自独立 IP；OpenStack 元数据路径为多家云共用，AWS 已被 token 探测先行识别，
// 剩余场景（华为老实例 DMI 无厂商标识）将其判为华为，放最后。
var imdsProbeList = []imdsProbe{
	{tenant.CLOUD_TYPE_AWS, http.MethodPut, "http://169.254.169.254/latest/api/token",
		map[string]string{"X-aws-ec2-metadata-token-ttl-seconds": "60"}},
	{tenant.CLOUD_TYPE_AZURE, http.MethodGet, "http://169.254.169.254/metadata/instance?api-version=2021-02-01",
		map[string]string{"Metadata": "true"}},
	{tenant.CLOUD_TYPE_GOOGLE, http.MethodGet, "http://169.254.169.254/computeMetadata/v1/instance/id",
		map[string]string{"Metadata-Flavor": "Google"}},
	{tenant.CLOUD_TYPE_ALIYUN, http.MethodGet, "http://100.100.100.200/latest/meta-data/instance-id", nil},
	{tenant.CLOUD_TYPE_TENCENT, http.MethodGet, "http://169.254.0.23/latest/meta-data/instance-id", nil},
	{tenant.CLOUD_TYPE_HW, http.MethodGet, "http://169.254.169.254/openstack/latest/meta_data.json", nil},
}

// detectCloudTypeByIMDS 通过各云厂商元数据服务识别云厂商
//
// 仅在启动探测时执行一次；单个探测 300ms 超时，pod 被封禁访问 IMDS 时快速失败走兜底。
func detectCloudTypeByIMDS() (tenant.CloudType, bool) {
	client := &http.Client{Timeout: 300 * time.Millisecond}
	for _, p := range imdsProbeList {
		req, err := http.NewRequest(p.method, p.url, nil)
		if err != nil {
			continue
		}
		for k, v := range p.header {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return p.cloud, true
		}
	}
	return tenant.CLOUD_TYPE_OTHER, false
}
