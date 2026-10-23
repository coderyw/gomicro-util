package auth

import (
	"context"
	"fmt"

	error_code "github.com/coderyw/gomicro-util/model/error-code"
	"github.com/coderyw/gomicro-util/library/component/jwt"
	"github.com/coderyw/gomicro-util/library/log"
	config2 "github.com/coderyw/gomicro-util/library/pmicro/config"
	"go-micro.dev/v4/metadata"
	"go-micro.dev/v4/server"
	"go.uber.org/zap"
)

func NewHandlerWrapper(cfg *config2.ServerCfg) server.HandlerWrapper {
	return func(fn server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			asCfg := cfg.MatchAuthServer(fmt.Sprintf("%v.%v", req.Service(), req.Method()))
			if asCfg == nil {
				return fn(ctx, req, rsp)
			}
			// 元数据获取数据
			_, cas, _ := getFromMetadata(ctx)
			// 获取认证数据
			switch asCfg.AuthType {
			case config2.RPC_AUTH_TYPE_BASIC:
				if cas == "" && !asCfg.ForceAuth {
					return fn(ctx, req, rsp)
				}
				if na, ok := asCfg.AuthMap[cas]; !ok {
					return error_code.GRPC_AUTHENTICATION_FAILURE.New()
				} else {
					log.InfoCtx(ctx, "AuthPass", zap.String("auth", na))
				}
			case config2.RPC_AUTH_TYPE_JWT:
				if !jwtCheck(ctx, cas, asCfg.AuthSecret) {
					return error_code.GRPC_AUTHENTICATION_FAILURE.New()
				}
			default:
				log.WarnCtx(ctx, "auth 类型暂未实现", zap.String("authType", asCfg.AuthType))
			}
			return fn(ctx, req, rsp)
		}
	}
}

// 校验 jwt
func jwtCheck(ctx context.Context, token string, secrets []string) bool {
	info := &jwt.AuthGrpcClaims{}
	// 循环对每个 secret 进行校验
	for _, secret := range secrets {
		err := jwt.ParseGrpcJwtToken(token, []byte(secret), info, false)
		if err != nil {
			log.WarnCtx(ctx, "jwt 解析失败", zap.Error(err))
			continue
		} else {
			log.InfoCtx(ctx, "jwt 解析成功", zap.Any("claims", info))
			return true
		}
	}
	return false
}

func getFromMetadata(ctx context.Context) (string, string, string) {
	at, _ := metadata.Get(ctx, config2.RPC_METADATA_X_AUTH_TYPE)
	as, _ := metadata.Get(ctx, config2.RPC_METADATA_X_AUTH_SECRET)
	av, _ := metadata.Get(ctx, config2.RPC_METADATA_X_AUTH_VERSION)
	return at, as, av
}
