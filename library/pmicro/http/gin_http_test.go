package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newCorsEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Cors([]string{"X-Device-Id"}))
	r.PATCH("/news/:id/status", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestCorsPreflight(t *testing.T) {
	r := newCorsEngine()

	t.Run("预检放行全部常用方法并带上业务自定义头", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/news/1/status", nil)
		req.Header.Set("Origin", "https://admin.example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodPatch)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
		methods := strings.Split(w.Header().Get("Access-Control-Allow-Methods"), ", ")
		for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions} {
			found := false
			for _, v := range methods {
				found = found || v == m
			}
			if !found {
				t.Fatalf("Allow-Methods %q missing %s", methods, m)
			}
		}
		if !strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), "X-Device-Id") {
			t.Fatalf("Allow-Headers missing configured header: %q", w.Header().Get("Access-Control-Allow-Headers"))
		}
	})

	t.Run("实际请求正常处理", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/news/1/status", nil)
		req.Header.Set("Origin", "https://admin.example.com")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK || w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Fatalf("status = %d, Allow-Origin = %q", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("没有 Origin 不写跨域头", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/news/1/status", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Header().Get("Access-Control-Allow-Methods") != "" {
			t.Fatalf("unexpected Allow-Methods %q", w.Header().Get("Access-Control-Allow-Methods"))
		}
	})
}
