package middleware

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseLogHeaders(t *testing.T) {
	tests := []struct {
		name         string
		cfg          string
		wantPrintAll bool
		wantAllow    []string
	}{
		{
			name: "不配置则不打印",
			cfg:  "",
		},
		{
			name: "纯空白等同不配置",
			cfg:  "   ",
		},
		{
			name:         "星号打印全部",
			cfg:          "*",
			wantPrintAll: true,
		},
		{
			name:      "多个 header 按逗号拆分",
			cfg:       "X-Request-Id,User-Agent",
			wantAllow: []string{"X-Request-Id", "User-Agent"},
		},
		{
			name:      "全小写配置被规范化",
			cfg:       "x-request-id",
			wantAllow: []string{"X-Request-Id"},
		},
		{
			name:      "全大写配置被规范化",
			cfg:       "X-REQUEST-ID",
			wantAllow: []string{"X-Request-Id"},
		},
		{
			name:      "忽略首尾空白与空段",
			cfg:       "  x-request-id ,, user-agent  ,",
			wantAllow: []string{"X-Request-Id", "User-Agent"},
		},
		{
			// 已知行为：星号与具体 header 名混配时，星号不再走 printAll 分支，
			// 而是被当成字面 header 名（规范化后仍是 "*"）。http.Header.Values("*")
			// 永远返回空，因此该项静默失效——只有 X-Foo 会被打印。
			name:      "星号与具体名混配时星号静默失效",
			cfg:       "*,X-Foo",
			wantAllow: []string{"*", "X-Foo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printAll, allow := parseLogHeaders(tt.cfg)
			if printAll != tt.wantPrintAll {
				t.Errorf("printAll = %v, want %v", printAll, tt.wantPrintAll)
			}
			if !reflect.DeepEqual(allow, tt.wantAllow) {
				t.Errorf("allowHeaders = %v, want %v", allow, tt.wantAllow)
			}
		})
	}
}

// newCtxWithHeaders 构造一个带指定请求头的 gin.Context
func newCtxWithHeaders(h map[string]string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	for k, v := range h {
		c.Request.Header.Set(k, v)
	}
	return c
}

func TestCollectLogHeaders(t *testing.T) {
	t.Run("printAll 返回全部 header 且不做任何脱敏", func(t *testing.T) {
		c := newCtxWithHeaders(map[string]string{
			"Authorization": "Bearer secret-token",
			"User-Agent":    "curl/8.4.0",
		})
		got := collectLogHeaders(c, true, nil)

		// 当前行为：原文返回，敏感 header 不脱敏。
		// 该断言是对现状的记录，不是对期望的背书——见报告 4.3 节的 P1。
		if want := []string{"Bearer secret-token"}; !reflect.DeepEqual(got["Authorization"], want) {
			t.Errorf("Authorization = %v, want %v", got["Authorization"], want)
		}
		if len(got) != len(c.Request.Header) {
			t.Errorf("len = %d, want %d (应为完整 header 集)", len(got), len(c.Request.Header))
		}
	})

	t.Run("白名单只收集命中的 header", func(t *testing.T) {
		c := newCtxWithHeaders(map[string]string{
			"X-Request-Id":  "req-1",
			"Authorization": "Bearer secret-token",
		})
		got := collectLogHeaders(c, false, []string{"X-Request-Id"})

		if len(got) != 1 {
			t.Fatalf("len = %d, want 1，实际内容 %v", len(got), got)
		}
		if want := []string{"req-1"}; !reflect.DeepEqual(got["X-Request-Id"], want) {
			t.Errorf("X-Request-Id = %v, want %v", got["X-Request-Id"], want)
		}
		if _, ok := got["Authorization"]; ok {
			t.Error("未配置的 Authorization 不应被收集")
		}
	})

	t.Run("白名单未命中的 header 不产生空条目", func(t *testing.T) {
		c := newCtxWithHeaders(map[string]string{"X-Request-Id": "req-1"})
		got := collectLogHeaders(c, false, []string{"X-Absent"})
		if len(got) != 0 {
			t.Errorf("len = %d, want 0，实际内容 %v", len(got), got)
		}
	})

	t.Run("空白名单返回 nil", func(t *testing.T) {
		c := newCtxWithHeaders(map[string]string{"X-Request-Id": "req-1"})
		if got := collectLogHeaders(c, false, nil); got != nil {
			t.Errorf("got = %v, want nil", got)
		}
	})

	t.Run("多值 header 全部保留", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Add("X-Multi", "a")
		c.Request.Header.Add("X-Multi", "b")

		got := collectLogHeaders(c, false, []string{"X-Multi"})
		if want := []string{"a", "b"}; !reflect.DeepEqual(got["X-Multi"], want) {
			t.Errorf("X-Multi = %v, want %v", got["X-Multi"], want)
		}
	})

	t.Run("端到端：配置经 parse 后大小写不敏感命中", func(t *testing.T) {
		c := newCtxWithHeaders(map[string]string{"X-Request-Id": "req-1"})
		// 配置写全小写，请求头是规范形式，应能命中
		printAll, allow := parseLogHeaders("x-request-id")
		got := collectLogHeaders(c, printAll, allow)
		if want := []string{"req-1"}; !reflect.DeepEqual(got["X-Request-Id"], want) {
			t.Errorf("X-Request-Id = %v, want %v", got["X-Request-Id"], want)
		}
	})

	t.Run("端到端：星号混配时星号项静默失效", func(t *testing.T) {
		c := newCtxWithHeaders(map[string]string{
			"X-Foo":      "foo",
			"User-Agent": "curl/8.4.0",
		})
		printAll, allow := parseLogHeaders("*,X-Foo")
		got := collectLogHeaders(c, printAll, allow)

		if printAll {
			t.Fatal("混配时不应进入 printAll 分支")
		}
		// 只有 X-Foo 命中；"*" 作为字面 header 名永远查不到，User-Agent 也不会被打印
		if len(got) != 1 {
			t.Errorf("len = %d, want 1，实际内容 %v", len(got), got)
		}
		if _, ok := got["User-Agent"]; ok {
			t.Error("星号未生效，User-Agent 不应出现")
		}
	})
}
