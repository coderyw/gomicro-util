// Package http
// @Author: yinwei
// @File: http_req
// @Version: 1.0.0
// @Date: 2024/12/6 09:13

package http_req

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/skywalking"
	"github.com/coderyw/gomicro-util/model/cst"
	"github.com/coderyw/gomicro-util/util"
	"go.uber.org/zap"
)

const defaultTimeout = time.Second * 20

var client = &http.Client{
	Timeout: defaultTimeout, //默认客户端超时时间20s
}

func getClient(timeout time.Duration) *http.Client {
	if timeout <= 0 || timeout == defaultTimeout {
		return client
	}
	return &http.Client{Timeout: timeout}

}

type Request[K any] struct {
	// 请求体数据 可为空
	ReqBody any
	//url 参数 可为空
	Query map[string]interface{}
	// 请求header 可为空
	Header http.Header
	// 超时时间 可为空 默认为 20s
	Timeout time.Duration
	// 数据类型
	ContentType string
	// 请求体数据重制
	ReqBodyReMake func(data []byte) (newBody []byte, header map[string]string, err error) `json:"-"`
	// 请求返回数据重制
	RespBodyMake (func([]byte) ([]byte, error)) `json:"-"`
}

// NewRequest 初始化一个 Http 请求
//
// Parameters:
//   - reqBody: interface{}
//
// Returns:
//   - *Request
func NewRequest[K any](reqBody any) *Request[K] {
	return &Request[K]{
		ReqBody:       reqBody,
		Header:        make(http.Header),
		ContentType:   "application/json;charset=UTF-8",
		ReqBodyReMake: nil,
	}
}

// Post 发起 post 请求
//
// Parameters:
//   - ctx: context.Context 上下文
//   - url: string 请求地址 域名加上路由
//   - ignoreEmptyResp: ...bool 是否忽略空的相应体;false=当响应是空的时候返回错误信息。true=当相应是空的时候继续序列化。默认为true
//
// Returns:
//   - T 响应体
//   - int http 状态码
//   - error 错误信息
func (r *Request[K]) Post(ctx context.Context, urlStr string, ignoreEmptyResp ...bool) (response K, httpCode int, err error) {
	skywalking.Start()
	start := time.Now()
	var (
		data []byte
	)
	// 打印日志 以及耗时
	defer func() {
		duration := time.Since(start)
		if err != nil {
			log.ErrorCtx(ctx, "[HTTP] Post", zap.Int("code", httpCode), zap.String("body", util.Bytes2Str(data)), zap.Duration("Duration", duration), zap.String(log.DUR_LOG_KEY, cst.GetHttpHandleLevel(duration)), zap.Error(err))
		} else {
			log.InfoCtx(ctx, "[HTTP] Post", zap.Int("code", httpCode), zap.Duration("Duration", duration), zap.String(log.DUR_LOG_KEY, cst.GetHttpHandleLevel(duration)), zap.Any("response", response))
		}
		skywalking.StopSpan()
	}()

	// 请求头请求提预处理
	if r.Header == nil {
		r.Header = make(http.Header)
	}

	if r.ReqBody != nil {
		// 请求体不为空时，处理对于类型
		switch r.ReqBody.(type) {
		case string: // 字符串类型要使用原始值，否则会多出一次字符串引号
			data = util.Str2Bytes(r.ReqBody.(string))
		default:
			data, err = json.Marshal(r.ReqBody)
			if err != nil {
				err = errors.Errorf("序列化http请求信息错误: %v", err.Error())
				return
			}
		}
	}

	// 请求体重制（加密等操作）
	// 将序列化后的请求体传入函数，返回新的请求体和请求头
	if r.ReqBodyReMake != nil {
		var h map[string]string
		// 新的请求体和新增的 header
		data, h, err = r.ReqBodyReMake(data)
		if err != nil {
			return
		}
		if h != nil {
			for k, v := range h {
				r.Header.Add(k, v)
			}
		}
	}
	// query 参数
	if r.Query != nil {
		var u *url.URL
		u, err = url.Parse(urlStr)
		if err != nil {
			return
		}
		params := url.Values{}
		// 拼接 query 参数
		for k, v := range r.Query {
			params.Add(k, fmt.Sprint(v))
		}
		u.RawQuery = params.Encode()
		urlStr = u.String()
	}
	// 初始化 http 请求
	req, err := http.NewRequest("POST", urlStr, bytes.NewReader(data))
	if err != nil {
		err = errors.Errorf("http请求创建错误: %v", err.Error())
		return
	}
	req.Header = r.Header
	if len(r.ContentType) != 0 { // 使用配置的 content-type
		req.Header.Add("Content-Type", r.ContentType)
	}
	if _, ok := req.Header["Content-Type"]; !ok { //如果没有配置，走默认配置
		req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	}
	resp, err := getClient(r.Timeout).Do(req) // 获取客户端并且发出请求
	if err != nil {
		err = errors.Errorf("http请求错误: %v", err.Error())
		return
	}
	defer resp.Body.Close()
	//读取响应体
	if data, err = io.ReadAll(resp.Body); err != nil {
		httpCode, err = resp.StatusCode, errors.Errorf("http读取数据错误: %v", err.Error())
		return
	}
	if len(ignoreEmptyResp) > 0 && !ignoreEmptyResp[0] && len(data) == 0 {
		// 如果响应体是空的，且没有配置忽略空的响应体，则会返回错误信息
		return response, resp.StatusCode, errors.Errorf("Got empty response body.")
	}
	//继续处理响应体

	// 响应体重制（解密等操作）
	if r.RespBodyMake != nil {
		data, err = r.RespBodyMake(data)
		if err != nil {
			return response, 500, err
		}
	}
	// 获取响应体类型
	response, isPtr := ref[K]()
	if isPtr { //指针类型
		err = json.Unmarshal(data, response)
	} else { //非指针类型
		err = json.Unmarshal(data, &response)
	}

	if err != nil {
		log.ErrorCtx(ctx, "[HTTP] 解析返回值错误", zap.String("data", string(data)))
		httpCode, err = resp.StatusCode, errors.Errorf("返序列化http返回消息错误: %v", err.Error())
	}

	return response, resp.StatusCode, err
}

// Get http get 请求
//
// Parameters:
//   - ctx: context.Context 上下文
//   - urlStr: string 请求地址，如果地址上携带了参数，并且 reqInfo也配置了Query参数，那么会被 Query 参数覆盖
//   - ignoreEmptyResp: ...bool 是否忽略空的返回体;false=当响应是空的时候返回错误信息。true=当相应是空的时候继续序列化。默认为true
//
// Returns:
//   - T 响应体
//   - int http 响应码
//   - error 错误信息
func (r *Request[K]) Get(ctx context.Context, urlStr string, ignoreEmptyResp ...bool) (response K, httpCode int, err error) {
	skywalking.Start()
	start := time.Now()
	var (
		data []byte
	)
	defer func() {
		duration := time.Since(start)
		if err != nil {
			log.ErrorCtx(ctx, "[HTTP] Get", zap.Int("code", httpCode), zap.String("body", util.Bytes2Str(data)), zap.Duration("Duration", duration), zap.String(log.DUR_LOG_KEY, cst.GetHttpHandleLevel(duration)), zap.Error(err))
		} else {
			log.InfoCtx(ctx, "[HTTP] Get", zap.Int("code", httpCode), zap.Duration("Duration", duration), zap.String(log.DUR_LOG_KEY, cst.GetHttpHandleLevel(duration)), zap.Any("response", response))
		}
		skywalking.StopSpan()
	}()
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	if r.ReqBody != nil {
		data, err = json.Marshal(r.ReqBody)
		if err != nil {
			err = errors.Errorf("序列化http请求信息错误: %v", err.Error())
			return
		}
	}

	if r.ReqBodyReMake != nil {
		var h map[string]string
		data, h, err = r.ReqBodyReMake(data)
		if err != nil {
			return
		}
		if h != nil {
			for k, v := range h {
				r.Header.Add(k, v)
			}
		}
	}
	// query 参数
	if r.Query != nil {
		var u *url.URL
		u, err = url.Parse(urlStr)
		if err != nil {
			return
		}
		params := url.Values{}
		for k, v := range r.Query {
			params.Add(k, fmt.Sprint(v))
		}
		u.RawQuery = params.Encode()
		urlStr = u.String()
	}

	req, err := http.NewRequest("GET", urlStr, bytes.NewReader(data))
	if err != nil {
		err = errors.Errorf("http请求创建错误: %v", err.Error())
		return
	}
	req.Header = r.Header
	if len(r.ContentType) != 0 {
		req.Header.Add("Content-Type", r.ContentType)
	}
	if _, ok := req.Header["Content-Type"]; !ok { //如果没有配置，走默认配置
		req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	}
	if r.Timeout > 0 {
		client.Timeout = r.Timeout
	}
	resp, err := getClient(r.Timeout).Do(req)
	if err != nil {
		err = errors.Errorf("http请求错误: %v", err.Error())
		return
	}
	defer resp.Body.Close()
	if data, err = io.ReadAll(resp.Body); err != nil {
		httpCode, err = resp.StatusCode, errors.Errorf("http读取数据错误: %v", err.Error())
		return
	}
	if len(ignoreEmptyResp) > 0 && !ignoreEmptyResp[0] && len(data) == 0 {
		return response, resp.StatusCode, errors.Errorf("Got empty response body.")
	}
	if r.RespBodyMake != nil {
		data, err = r.RespBodyMake(data)
		if err != nil {
			return response, 500, err
		}
	}
	response, isPtr := ref[K]()
	if isPtr {
		err = json.Unmarshal(data, response)
	} else {
		err = json.Unmarshal(data, &response)
	}
	if err != nil {
		err = errors.Errorf("返序列化http返回消息错误: %v", err.Error())
		return response, resp.StatusCode, err
	}
	return response, resp.StatusCode, err
}

func ref[T any]() (t T, isPtr bool) {
	// 获取类型的反射信息
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		isPtr = true
		typ = typ.Elem()
	}

	var nt reflect.Value
	if typ.Kind() == reflect.Ptr {
		nt = reflect.New(typ)
	} else if typ.Kind() == reflect.Struct {
		nt = reflect.New(typ).Elem()
	} else if typ.Kind() == reflect.Slice {
		return reflect.MakeSlice(typ, 0, 0).Interface().(T), isPtr
	}
	// 存入缓存
	if isPtr {
		return nt.Addr().Interface().(T), isPtr
	} else {
		return nt.Interface().(T), isPtr
	}
}
