// Package response
// @Author: yinwei
// @File: resp_handler_wrapper
// @Version: 1.0.0
// @Date: 2024/6/8 13:35

package response

import (
	context "context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	error_code "github.com/coderyw/gomicro-util/model/error-code"
	"github.com/coderyw/gomicro-util/library/db/cache/redis"
	"github.com/coderyw/gomicro-util/library/log"
	grpc2 "github.com/coderyw/gomicro-util/library/pmicro/plugin/client_grpc"
	"github.com/coderyw/gomicro-util/model/cst"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	metadata2 "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const (
	MSG_CODE     = "msg-code"
	EXTRA_MSG    = "extra-msg"
	MSG_CODE_MSG = "msg-code-msg"

	EXTRA_MSG_CONTENT = "extra msg error"
)

// 主要用于grpc的返回值，对返回值进行处理
// 如果grpc返回报错，err。则会对err进行解析，解析出其code，然后将code设置在metadata中，返回个caller
// 如果grpc返回正常，则返回正常，则不会设置code

// metadata传递方式与gomicro不一样，这里使用grpc的SendHeader方法。接受端caller使用一个md进行接受，并通过calloption的ctx传递到下层caller

// 解析返回数据，对返回错误进行处理
func RespHandlerWrapper() server.HandlerWrapper {
	return func(handlerFunc server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			log.InfoCtx(ctx, "GRPC Handle start", zap.String("method", req.Method()), zap.Any("req", req.Body()))
			start := time.Now()
			defer func() {
				du := time.Since(start)
				log.InfoCtx(ctx, "GRPC Handle End", zap.String("method", req.Method()), zap.Duration("duration", du), zap.String(log.DUR_LOG_KEY, cst.GetHttpHandleLevel(du)), zap.Any("req", req.Body()))
			}()
			// 业务
			err := handlerFunc(ctx, req, rsp)
			if err != nil { //如果处理报错

				// 准备元数据返回
				var md metadata2.MD
				// 解析错误
				code, ok, err := error_code.MustUnwrap(err)
				if !ok { //err 非 errorCode 错误
					// 判断err是否可以打印堆栈
					if _, ok := err.(interface{ Format(st fmt.State, verb rune) }); ok {
						log.ErrorCtx(ctx, "Handler错误", zap.String("service", req.Service()), zap.String("method", req.Method()), zap.Any("req", req.Body()), zap.Error(err))
					} else {
						log.ErrorCtx(ctx, "Handler错误", zap.String("service", req.Service()), zap.String("method", req.Method()), zap.Error(err), zap.Any("req", req.Body()))
					}
					// 兼容gorm record not found情况
					if errors.Is(err, gorm.ErrRecordNotFound) {
						code = error_code.RECORD_NOT_FOUND
					} else if errors.Is(err, redis.RedisNil) { // 兼容redis nil情况
						code = error_code.RECORD_NOT_FOUND
					} else { //默认使用服务错误
						code = error_code.INTERNAL_SERVER_ERROR
					}
				} else {
					log.WarnCtx(ctx, "Handler错误", zap.String("service", req.Service()), zap.String("method", req.Method()), zap.Any("code", code), zap.Any("Status", err))
				}
				var msg string
				if err != nil {
					msg = err.Error()
				}

				md = metadata2.Pairs(
					MSG_CODE, strconv.Itoa(int(code.Code())),
					EXTRA_MSG, msg,
					MSG_CODE_MSG, code.GetMessage(),
				)
				// 在上下文中保存元数据
				err = grpc.SetTrailer(ctx, md)
				if err != nil {
					log.ErrorCtx(ctx, "set trailer error", zap.Error(err))
				}
				// 回复grpc错误信息
				return status.Error(codes.Internal, msg)
			}
			return nil
		}
	}
}

// NewCallWrapper 在call后执行，处理错误问题
func NewCallWrapper() client.CallWrapper {
	return func(callFunc client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {

			log.InfoCtx(ctx, makeGrpcLogKey(req.Service(), req.Method()), zap.Any("req", req.Body()))
			start := time.Now()
			var err error
			// 接受消息返回header
			respHeader := metadata2.New(make(map[string]string))
			if opts.Context == nil {
				opts.Context = context.Background()
			}
			var internalError bool
			defer func() {
				du := time.Since(start)
				key := makeGrpcLogKey(req.Service(), req.Method(), "end")
				if err == nil {
					fields := []zap.Field{zap.Duration("duration", du),
						zap.String(log.DUR_LOG_KEY, cst.GetHttpHandleLevel(du)), zap.Any("Grpc Md", respHeader)}
					if log.ShouldLogResponse(ctx) {
						fields = append(fields, zap.Any("rsp", rsp))
					} else {
						fields = append(fields, zap.String("rsp", "ignore response body by server"))
					}
					log.InfoCtx(ctx, key, fields...)
				} else if internalError {
					log.ErrorCtx(ctx, key,
						zap.Duration("duration", du),
						zap.String(log.DUR_LOG_KEY, cst.GetHttpHandleLevel(du)), zap.Any("Grpc Md", respHeader),
						zap.Any("rsp", rsp), zap.Error(err))
				} else {
					log.WarnCtx(ctx, key,
						zap.Duration("duration", du),
						zap.String(log.DUR_LOG_KEY, cst.GetHttpHandleLevel(du)), zap.Any("Grpc Md", respHeader),
						zap.Any("rsp", rsp), zap.Any("Status", err))
				}
			}()

			// 将接受元数据的变量传入上下文中，并带入请求
			opts.Context = grpc2.WithCtx(opts.Context, grpc.Trailer(&respHeader))
			// 发起请求
			err = callFunc(ctx, node, req, rsp, opts)
			if err != nil { // 请求报错
				// 解析元数据，获取错误信息
				if codeM, ok := respHeader[MSG_CODE]; ok && len(codeM) > 0 {
					code, err1 := strconv.Atoi(codeM[0])
					if err1 != nil {
						return error_code.INTERNAL_SERVER_ERROR.New(err1)
					}
					// 获取extra msg
					var msg, codeMsg string
					if ex, ok := respHeader[EXTRA_MSG]; ok && len(ex) > 0 {
						msg = ex[0]
					}
					if ex, ok := respHeader[MSG_CODE_MSG]; ok && len(ex) > 0 {
						codeMsg = ex[0]
					} else {
						codeMsg = error_code.INTERNAL_SERVER_ERROR.GetMessage()
					}
					err = error_code.GrpcCode2Error(uint32(code), codeMsg, msg)
				} else {
					internalError = true
					err = error_code.INTERNAL_SERVER_ERROR.New(err)
				}
			}
			return err
		}
	}
}

// 用于对象池
var bupool = sync.Pool{New: func() interface{} {
	return new(strings.Builder)
}}

func makeGrpcLogKey(s, m string, after ...string) string {
	builder := bupool.Get().(*strings.Builder)
	builder.WriteString("[GRPC] ")
	builder.WriteString(" ")
	builder.WriteString("Call")
	builder.WriteString(" ")
	builder.WriteString(s)
	builder.WriteString(".")
	builder.WriteString(m)
	if len(after) > 0 {
		for i, v := range after {
			if i == 0 {
				builder.WriteString(" ")
			}
			builder.WriteString(v)
			if i < len(after)-1 {
				builder.WriteString(" ")
			}
		}
	}

	str := builder.String()
	builder.Reset()
	bupool.Put(builder)
	return str
}

func statusError(err error) string {
	s, ok := status.FromError(err)
	if ok {
		return s.Message()
	}
	return err.Error()
}
