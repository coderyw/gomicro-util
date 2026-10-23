// Package net_http
// @Author: yinwei
// @File: client_test.go
// @Version: 1.0.0
// @Date: 2025/3/13 15:49

package net_http

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/coderyw/gomicro-util/library/http/http_req"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/trace"
	"go.uber.org/zap"
)

func Init() {
	ge := gin.Default()
	ge.GET("/get", func(c *gin.Context) {
		log.Info("got a get request", zap.Any("param1", c.Query("param1")), zap.Any("param2", c.Query("param2")))

		c.JSON(200, RequestGetResp{
			Name: "GetName",
			Age:  234,
		})
	})
	ge.POST("/post", func(c *gin.Context) {
		req := RequestPostReq{}
		err := c.Bind(&req)
		if err != nil {
			log.ErrorCtx(c, "bind error", zap.Error(err))
			c.JSON(200, RequestPostResp{
				Code: 500,
				Data: "error",
			})
			return
		}
		log.Info("收到 post 请求", zap.Any("data", req))
		c.JSON(200, RequestPostResp{
			Code: 0,
			Data: "success",
		})
	})
	go ge.Run(":8081")
}

type RequestPostReq struct {
	Reason string `json:"reason"`
	Amount int    `json:"amount"`
}

func (r RequestPostReq) MarshalBinary() (data []byte, err error) {
	return json.Marshal(r)
}

type RequestPostResp struct {
	Code int    `json:"code"`
	Data string `json:"data"`
}

func (r *RequestPostResp) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}

type RequestGetResp struct {
	Name string `json:"name"`
	Age  int64  `json:"age"`
}

func (r *RequestGetResp) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}

func TestGet(t *testing.T) {
	Init()
	type args struct {
		ctx             context.Context
		urlStr          string
		reqInfo         http_req.Request[*RequestGetResp]
		ignoreEmptyResp []bool
	}
	type testCase[T any] struct {
		name    string
		args    args
		want    T
		want1   int
		wantErr bool
	}
	tests := []testCase[*RequestGetResp]{
		{name: "succ", args: args{
			ctx:    trace.GetTraceCtx(nil),
			urlStr: "http://127.0.0.1:8081/get",
			reqInfo: http_req.Request[*RequestGetResp]{
				Query: map[string]interface{}{
					"param1": "参数 1",
					"param2": "参数 2",
				},
			},
			ignoreEmptyResp: []bool{true},
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := Get(tt.args.ctx, tt.args.urlStr, tt.args.reqInfo, tt.args.ignoreEmptyResp...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			t.Logf("Get() got = %v", got)

			t.Logf("Get() got1 = %v", got1)
		})
	}
}

func TestPost(t *testing.T) {
	Init()
	type args struct {
		ctx             context.Context
		urlStr          string
		reqInfo         http_req.Request[*RequestPostResp]
		ignoreEmptyResp []bool
	}
	type testCase[T any] struct {
		name    string
		args    args
		wantErr bool
	}
	tests := []testCase[*RequestPostResp]{
		{name: "succ", args: args{
			ctx:    trace.GetTraceCtx(nil),
			urlStr: "http://127.0.0.1:8081/post",
			reqInfo: http_req.Request[*RequestPostResp]{
				ReqBody: &RequestPostReq{
					Reason: "reason",
					Amount: 100,
				},
			},
			ignoreEmptyResp: []bool{true},
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := Post[*RequestPostResp](tt.args.ctx, tt.args.urlStr, tt.args.reqInfo, tt.args.ignoreEmptyResp...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Post() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			t.Logf("Post() got = %v", got)

			t.Logf("Post() got1 = %v", got1)
		})
	}
}
