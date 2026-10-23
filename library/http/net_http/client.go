// Package net_http
// @Author: yinwei
// @File: client
// @Version: 1.0.0
// @Date: 2024/12/6 09:09

package net_http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/coderyw/gomicro-util/library/http/http_req"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/skywalking"
	"github.com/coderyw/gomicro-util/model/cst"
	"github.com/coderyw/gomicro-util/util"
	"go.uber.org/zap"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const defaultTimeout = time.Second * 20

var defaultClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 200,
	},
	Timeout: defaultTimeout,
}

func getClient(timeout time.Duration) *http.Client {
	if timeout <= 0 || timeout == defaultTimeout {
		return defaultClient
	}
	return &http.Client{Timeout: timeout}
}

// Post 发起 post 请求
//
// Parameters:
//   - ctx: context.Context 上下文
//   - urlStr: string 请求地址 域名加上路由
//   - reqInfo: http_req.Request 请求信息
//   - ignoreEmptyResp: ...bool 是否忽略空的响应体；false=空响应返回错误，true=继续序列化，默认 true
//
// Returns:
//   - T 响应体
//   - int http 状态码
//   - error 错误信息
func Post[T any](ctx context.Context, urlStr string, reqInfo http_req.Request[T], ignoreEmptyResp ...bool) (T, int, error) {
	return doRequest[T](ctx, http.MethodPost, urlStr, reqInfo, ignoreEmptyResp...)
}

// Get 发起 get 请求
//
// Parameters:
//   - ctx: context.Context 上下文
//   - urlStr: string 请求地址，Query 参数会覆盖 URL 中已有的同名参数
//   - reqInfo: http_req.Request 请求信息
//   - ignoreEmptyResp: ...bool 是否忽略空的响应体；false=空响应返回错误，true=继续序列化，默认 true
//
// Returns:
//   - T 响应体
//   - int http 响应码
//   - error 错误信息
func Get[T any](ctx context.Context, urlStr string, reqInfo http_req.Request[T], ignoreEmptyResp ...bool) (T, int, error) {
	return doRequest[T](ctx, http.MethodGet, urlStr, reqInfo, ignoreEmptyResp...)
}

func doRequest[T any](ctx context.Context, method, urlStr string, reqInfo http_req.Request[T], ignoreEmptyResp ...bool) (response T, httpCode int, err error) {
	skywalking.Start()
	start := time.Now()
	var data []byte
	defer func() {
		duration := time.Since(start)
		if err != nil {
			log.ErrorCtx(ctx, "[HTTP] "+method, zap.Int("code", httpCode), zap.String("body", util.Bytes2Str(data)), zap.Duration("Duration", duration), zap.String(log.DUR_LOG_KEY, cst.GetHttpHandleLevel(duration)), zap.Error(err))
		} else {
			log.InfoCtx(ctx, "[HTTP] "+method, zap.Int("code", httpCode), zap.Duration("Duration", duration), zap.String(log.DUR_LOG_KEY, cst.GetHttpHandleLevel(duration)), zap.Any("response", response))
		}
		skywalking.StopSpan()
	}()

	if reqInfo.Header == nil {
		reqInfo.Header = make(http.Header)
	}

	if reqInfo.ReqBody != nil {
		if s, ok := reqInfo.ReqBody.(string); ok {
			data = []byte(s)
		} else {
			data, err = json.Marshal(reqInfo.ReqBody)
			if err != nil {
				err = errors.Errorf("序列化http请求信息错误: %v", err.Error())
				return
			}
		}
	}

	if reqInfo.ReqBodyReMake != nil {
		var h map[string]string
		data, h, err = reqInfo.ReqBodyReMake(data)
		if err != nil {
			return
		}
		for k, v := range h {
			reqInfo.Header.Add(k, v)
		}
	}

	if reqInfo.Query != nil {
		var u *url.URL
		u, err = url.Parse(urlStr)
		if err != nil {
			return
		}
		params := url.Values{}
		for k, v := range reqInfo.Query {
			params.Add(k, fmt.Sprint(v))
		}
		u.RawQuery = params.Encode()
		urlStr = u.String()
	}

	req, err := http.NewRequest(method, urlStr, bytes.NewReader(data))
	if err != nil {
		err = errors.Errorf("http请求创建错误: %v", err.Error())
		return
	}
	req.Header = reqInfo.Header
	if reqInfo.ContentType != "" {
		req.Header.Set("Content-Type", reqInfo.ContentType)
	} else if _, ok := req.Header["Content-Type"]; !ok {
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	}

	resp, err := getClient(reqInfo.Timeout).Do(req)
	if err != nil {
		err = errors.Errorf("http请求错误: %v", err.Error())
		return
	}
	defer resp.Body.Close()

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		httpCode, err = resp.StatusCode, errors.Errorf("http读取数据错误: %v", err.Error())
		return
	}
	httpCode = resp.StatusCode

	if len(ignoreEmptyResp) > 0 && !ignoreEmptyResp[0] && len(data) == 0 {
		return response, httpCode, errors.New("Got empty response body.")
	}

	if reqInfo.RespBodyMake != nil {
		data, err = reqInfo.RespBodyMake(data)
		if err != nil {
			return response, 500, err
		}
	}

	response, _ = ref[T]()
	if err = json.Unmarshal(data, response); err != nil {
		log.ErrorCtx(ctx, "[HTTP] 解析返回值错误", zap.String("data", string(data)))
		err = errors.Errorf("返序列化http返回消息错误: %v", err.Error())
	}
	return response, httpCode, err
}

func ref[T any]() (t T, isPtr bool) {
	typ := reflect.TypeOf(t)
	if typ == nil {
		return
	}
	if typ.Kind() == reflect.Ptr {
		isPtr = true
		typ = typ.Elem()
	}

	var nt reflect.Value
	switch typ.Kind() {
	case reflect.Ptr:
		nt = reflect.New(typ)
	case reflect.Struct:
		nt = reflect.New(typ).Elem()
	case reflect.Slice:
		return reflect.MakeSlice(typ, 0, 0).Interface().(T), isPtr
	default:
		return
	}

	if isPtr {
		return nt.Addr().Interface().(T), isPtr
	}
	return nt.Interface().(T), isPtr
}
