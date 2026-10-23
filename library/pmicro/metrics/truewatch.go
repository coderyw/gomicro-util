// Package metrics
// @Author: yinwei
// @File: truewatch
// @Version: 1.0.0
// @Date: 2026/8/12 10:00

package metrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	dto "github.com/prometheus/client_model/go"
	srv_config "github.com/coderyw/gomicro-util/model/config"
)

// truewatchReporter 对接 Truewatch DataWay HTTP API，把指标快照以行协议 POST 到 /v1/write/metric。
type truewatchReporter struct {
	endpoint    string
	token       string
	client      *http.Client
	tags        map[string]string
	measurement string
}

// newTruewatchReporter 校验配置并构造上报器；配置不全返回 error，
// 由调用方（Init）降级为不上报，而不是让服务启动失败。
func newTruewatchReporter(cfg srv_config.TruewatchCfg, measurement string, commonTags map[string]string) (*truewatchReporter, error) {
	if cfg.DatawayUrl == "" {
		return nil, fmt.Errorf("truewatch: dataway_url 为空")
	}
	if cfg.WorkspaceToken == "" {
		return nil, fmt.Errorf("truewatch: workspace_token 为空")
	}
	if measurement == "" {
		return nil, fmt.Errorf("truewatch: measurement(serviceName) 为空")
	}
	return &truewatchReporter{
		endpoint:    strings.TrimRight(cfg.DatawayUrl, "/") + "/v1/write/metric",
		token:       cfg.WorkspaceToken,
		client:      &http.Client{Timeout: reportTimeout},
		tags:        commonTags,
		measurement: measurement,
	}, nil
}

func (r *truewatchReporter) report(ctx context.Context, families []*dto.MetricFamily) error {
	payload := toLineProtocol(r.measurement, r.tags, families, time.Now().UnixNano())
	if payload == "" {
		return nil
	}

	target, err := url.Parse(r.endpoint)
	if err != nil {
		return fmt.Errorf("解析 dataway url: %w", err)
	}
	query := target.Query()
	query.Set("token", r.token)
	target.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("构造请求: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送指标: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("dataway 返回 http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
