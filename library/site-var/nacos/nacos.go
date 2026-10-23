// Package nacos
// @Author: yinwei
// @File: nacos
// @Version: 1.0.0
// @Date: 2024/6/27 09:56

package nacos

import (
	"context"
	"errors"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/coderyw/gomicro-util/library/component/secret"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/config"
	"go.uber.org/zap"
)

var client config_client.IConfigClient

func SyncNacos(cfg *config.RegistryCfg) error {
	if cfg == nil {
		return errors.New("NACOS_URLS 有误")
	}
	clientConfig := constant.ClientConfig{
		NamespaceId:          cfg.NameId,
		Password:             cfg.Pwd,
		Username:             cfg.Account,
		NotLoadCacheAtStart:  true,
		UpdateCacheWhenEmpty: true,
		LogLevel:             "error",
	}
	serverConfigs := make([]constant.ServerConfig, 0)
	contextPath := "/nacos"
	for _, addr := range cfg.Hosts {
		// check we have a port
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return err
		}

		p, err := strconv.ParseUint(port, 10, 64)
		if err != nil {
			return err
		}

		serverConfigs = append(serverConfigs, constant.ServerConfig{
			IpAddr:      host,
			Port:        p,
			ContextPath: contextPath,
		})
	}

	clientConfig.TimeoutMs = uint64(time.Millisecond * 3)
	var err error
	client, err = clients.CreateConfigClient(map[string]interface{}{
		constant.KEY_SERVER_CONFIGS: serverConfigs,
		constant.KEY_CLIENT_CONFIG:  clientConfig,
	})
	logger.SetLogger(log.GetNacosLogger())
	return err
}

func GetNacosCfgData(dataId, group string, data interface{}) error {
	log.Info("获取nacos配置", zap.String("dataId", dataId), zap.String("group", group))
	content, err := client.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
	if err != nil {
		return err
	}
	return decodeConfig(dataId, content, data)
}

// decodeConfig 配置文本经凭据引用解析后反序列化：走 yaml/json 解析器的节点树替换引用，
// 引用解析失败直接返回错误（启动阶段 fail-fast），反序列化错误已脱去凭据值
func decodeConfig(dataId, content string, data interface{}) error {
	var err error
	if strings.HasSuffix(dataId, ".yaml") {
		err = secret.UnmarshalYAML(context.Background(), content, data)
	} else {
		err = secret.UnmarshalJSON(context.Background(), content, data)
	}
	if err != nil {
		log.Error("[decodeConfig] 配置解析失败", zap.String("dataId", dataId), zap.Error(err))
	}
	return err
}

func ConfigListen[T any](dataId, group string, onChange func(obj T)) error {
	return client.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
		OnChange: func(namespace, group, dataId, data string) {
			log.Info("NacosConfigChange", zap.String("namespace", namespace),
				zap.String("group", group),
				zap.String("dataId", dataId))
			if onChange != nil {
				obj := ref[T](true)
				// 热更新：解析或反序列化失败都只告警并保留旧配置，不退出进程
				if err := decodeConfig(dataId, data, obj); err != nil {
					log.Error("[ConfigListen] 配置变更解析失败，保留旧配置", zap.String("dataId", dataId), zap.Error(err))
					return
				}
				onChange(obj)
			}
		},
	})
}

func ref[T any](ptr bool) (t T) {
	// 获取类型的反射信息
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	var nt reflect.Value
	if typ.Kind() == reflect.Ptr {
		nt = reflect.New(typ)
	} else if typ.Kind() == reflect.Struct {
		nt = reflect.New(typ).Elem()
	} else if typ.Kind() == reflect.Slice {
		return reflect.MakeSlice(typ, 0, 0).Interface().(T)
	}
	// 存入缓存
	if ptr {
		return nt.Addr().Interface().(T)
	} else {
		return nt.Interface().(T)
	}
}
