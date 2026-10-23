package env

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/coderyw/gomicro-util/model/cst"
	"github.com/coderyw/gomicro-util/model/tenant"
)

// useDMIDir 把 DMI 探测路径指向临时目录并写入指定文件，用例结束后恢复
func useDMIDir(t *testing.T, files map[string]string) {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("写入 DMI 测试文件失败: %v", err)
		}
	}
	old := dmiSysPath
	dmiSysPath = dir
	t.Cleanup(func() { dmiSysPath = old })
}

// stubIMDS 替换 IMDS 探测列表，用例结束后恢复；传 nil 可禁用网络探测
func stubIMDS(t *testing.T, list []imdsProbe) {
	t.Helper()
	old := imdsProbeList
	imdsProbeList = list
	t.Cleanup(func() { imdsProbeList = old })
}

// 环境变量显式配置优先，且大小写不敏感
func TestDetectCloudType_EnvFirst(t *testing.T) {
	t.Setenv(cst.ENV_CLOUD_TYPE, "AWS")
	// DMI 指向华为，验证环境变量优先级更高
	useDMIDir(t, map[string]string{"sys_vendor": "Huawei Cloud\n"})
	stubIMDS(t, nil)

	if got := detectCloudType(); got != tenant.CLOUD_TYPE_AWS {
		t.Errorf("detectCloudType() = %v, want %v", got, tenant.CLOUD_TYPE_AWS)
	}
}

// 环境变量配置了无法识别的值：显式配置仍然生效，返回 OTHER 而不再探测
func TestDetectCloudType_EnvUnknown(t *testing.T) {
	t.Setenv(cst.ENV_CLOUD_TYPE, "some-private-cloud")
	useDMIDir(t, map[string]string{"sys_vendor": "Huawei Cloud\n"})
	stubIMDS(t, nil)

	if got := detectCloudType(); got != tenant.CLOUD_TYPE_OTHER {
		t.Errorf("detectCloudType() = %v, want %v", got, tenant.CLOUD_TYPE_OTHER)
	}
}

// 未配置环境变量时走 DMI 探测
func TestDetectCloudType_DMI(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  tenant.CloudType
	}{
		{name: "huawei", files: map[string]string{"sys_vendor": "Huawei Cloud\n"}, want: tenant.CLOUD_TYPE_HW},
		{name: "huawei-chassis", files: map[string]string{"chassis_asset_tag": "HUAWEICLOUD\n"}, want: tenant.CLOUD_TYPE_HW},
		{name: "aws-nitro", files: map[string]string{"sys_vendor": "Amazon EC2\n"}, want: tenant.CLOUD_TYPE_AWS},
		{name: "aws-xen-uuid", files: map[string]string{"sys_vendor": "Xen\n", "product_uuid": "EC2A1B2C-3D4E-5F60-7182-93A4B5C6D7E8\n"}, want: tenant.CLOUD_TYPE_AWS},
		{name: "azure", files: map[string]string{"sys_vendor": "Microsoft Corporation\n", "product_name": "Virtual Machine\n"}, want: tenant.CLOUD_TYPE_AZURE},
		{name: "google", files: map[string]string{"product_name": "Google Compute Engine\n"}, want: tenant.CLOUD_TYPE_GOOGLE},
		{name: "tencent", files: map[string]string{"sys_vendor": "Tencent Cloud\n"}, want: tenant.CLOUD_TYPE_TENCENT},
		{name: "aliyun", files: map[string]string{"sys_vendor": "Alibaba Cloud\n"}, want: tenant.CLOUD_TYPE_ALIYUN},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(cst.ENV_CLOUD_TYPE, "")
			useDMIDir(t, c.files)
			stubIMDS(t, nil)
			if got := detectCloudType(); got != c.want {
				t.Errorf("detectCloudType() = %v, want %v", got, c.want)
			}
		})
	}
}

// DMI 未识别时走 IMDS 探测：按序命中返回对应厂商
func TestDetectCloudTypeByIMDS_Hit(t *testing.T) {
	// 前两个探测失败（404），第三个命中
	miss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(miss.Close)
	hit := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(hit.Close)

	stubIMDS(t, []imdsProbe{
		{tenant.CLOUD_TYPE_AWS, http.MethodPut, miss.URL, nil},
		{tenant.CLOUD_TYPE_AZURE, http.MethodGet, miss.URL, nil},
		{tenant.CLOUD_TYPE_HW, http.MethodGet, hit.URL, nil},
	})

	got, ok := detectCloudTypeByIMDS()
	if !ok || got != tenant.CLOUD_TYPE_HW {
		t.Errorf("detectCloudTypeByIMDS() = (%v, %v), want (%v, true)", got, ok, tenant.CLOUD_TYPE_HW)
	}
}

// IMDS 全部失败（封禁/非云环境）：返回 false 交由上层兜底
func TestDetectCloudTypeByIMDS_AllFail(t *testing.T) {
	miss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(miss.Close)

	stubIMDS(t, []imdsProbe{
		{tenant.CLOUD_TYPE_AWS, http.MethodPut, miss.URL, nil},
		{tenant.CLOUD_TYPE_HW, http.MethodGet, "http://127.0.0.1:1/unreachable", nil},
	})

	if got, ok := detectCloudTypeByIMDS(); ok {
		t.Errorf("detectCloudTypeByIMDS() = (%v, true), want ok=false", got)
	}
}

// 环境变量、DMI、IMDS 均未识别：兜底返回 OTHER
func TestDetectCloudType_Fallback(t *testing.T) {
	t.Setenv(cst.ENV_CLOUD_TYPE, "")
	// 目录存在但厂商信息无法识别
	useDMIDir(t, map[string]string{"sys_vendor": "QEMU\n"})
	stubIMDS(t, nil)

	if got := detectCloudType(); got != tenant.CLOUD_TYPE_OTHER {
		t.Errorf("detectCloudType() = %v, want %v", got, tenant.CLOUD_TYPE_OTHER)
	}
}

// DMI 路径不可读（受限运行时）：不报错，兜底返回 OTHER
func TestDetectCloudType_DMIUnreadable(t *testing.T) {
	t.Setenv(cst.ENV_CLOUD_TYPE, "")
	old := dmiSysPath
	dmiSysPath = filepath.Join(t.TempDir(), "not-exist")
	t.Cleanup(func() { dmiSysPath = old })
	stubIMDS(t, nil)

	if got := detectCloudType(); got != tenant.CLOUD_TYPE_OTHER {
		t.Errorf("detectCloudType() = %v, want %v", got, tenant.CLOUD_TYPE_OTHER)
	}
}

// GetCloudType 结果缓存：首次探测后环境变化不再影响返回值
func TestGetCloudType_Cached(t *testing.T) {
	t.Setenv(cst.ENV_CLOUD_TYPE, "aws")
	first := GetCloudType()
	t.Setenv(cst.ENV_CLOUD_TYPE, "huawei")
	if got := GetCloudType(); got != first {
		t.Errorf("GetCloudType() 缓存失效: first=%v, second=%v", first, got)
	}
}
