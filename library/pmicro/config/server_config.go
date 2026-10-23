package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"
	mul_tenant "github.com/coderyw/gomicro-util/internal/mul-tenant"
	"github.com/coderyw/gomicro-util/library/component/secret"
	_ "github.com/coderyw/gomicro-util/library/component/secret/provider/awssm" // 注册 awssm:// 引用解析
	_ "github.com/coderyw/gomicro-util/library/component/secret/provider/csms"  // 注册 csms:// 引用解析
	env2 "github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/pmicro/option"
	"github.com/coderyw/gomicro-util/library/site-var/etcd"
	"github.com/coderyw/gomicro-util/library/site-var/nacos"
	srv_config "github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/cst"
	"github.com/coderyw/gomicro-util/model/facade"
	"go.uber.org/zap"
)

const (
	RPC_AUTH_TYPE_BASIC = "basic"
	RPC_AUTH_TYPE_JWT   = "jwt"

	// 认证元数据类型
	RPC_METADATA_X_AUTH_TYPE = "x-auth-type"
	// 认证元数据密钥
	RPC_METADATA_X_AUTH_SECRET       = "x-auth-secret"
	RPC_METADATA_X_AUTH_SECRET_MICRO = "X-Auth-Secret"
	// 认证元数据版本
	RPC_METADATA_X_AUTH_VERSION = "x-auth-version"
)

type ServerCfg struct {
	cfg                *srv_config.ServerConfig
	lock               sync.RWMutex
	authServer         map[string]*srv_config.AuthServer
	authClient         map[string]*srv_config.AuthClient
	defaultCustomerCfg any
}

// 服务的配置处理
func InitCfg(opt *option.Options) (*ServerCfg, error) {
	if opt.ServerName == "" {
		return nil, errors.New("serverName为空")
	}
	var (
		err  error
		scfg = new(ServerCfg)
	)
	scfg.lock.Lock()
	defer scfg.lock.Unlock()
	scfg.defaultCustomerCfg = opt.CustomDefaultCfg
	if scfg.defaultCustomerCfg != nil {
		if err := initCustomerConfig(scfg.defaultCustomerCfg, opt.CustomCfgChangeFunc); err != nil {
			return nil, err
		}
	}
	// 环境变量检测
	err = env2.SyncEnv()
	if err != nil {
		return nil, err
	}

	var cfgType string
	url := os.Getenv(cst.ENV_ETCD_URLS) //环境变量中的etcd地址
	if len(url) != 0 {
		cfgType = "etcd"
	}
	if url = os.Getenv(cst.ENV_NACOS_URLS); len(url) != 0 {
		cfgType = "nacos"
	}

	switch env2.GetEnv() {
	case cst.RUNTIME_ENV_LOCAL:

	case cst.RUNTIME_ENV_PROD,
		cst.RUNTIME_ENV_PRE,
		cst.RUNTIME_ENV_ROADSHOW,
		cst.RUNTIME_ENV_TEST,
		cst.RUNTIME_ENV_DEV,
		cst.RUNTIME_ENV_PRE_AUTO,
		cst.RUNTIME_ENV_TEST_AUTO:
		// 环境非本地时候，环境变量中的etcd地址必须有值
		if len(url) == 0 {
			return nil, errors.Errorf("环境变量缺失运行环境ENVIRON")
		}
	default:
		return nil, errors.Errorf("环境变量ENVIRON类型错误 %v", env2.GetEnv())
	}
	if len(cfgType) != 0 {
		err = scfg.initFromCfgCenter(opt, cfgType, url) // 从配置中心加载服务配置
		if err != nil {
			log.Error("配置缺失", zap.Error(err))
			return nil, err
		}
	} else {
		//配置文件中加载
		if err = scfg.initFromCfgFile(opt); err != nil { //从配置文件加载服务配置
			return nil, err
		}
	}

	if scfg.cfg.Http.JwtExpTime == 0 {
		scfg.cfg.Http.JwtExpTime = 3600 * 14
	}

	//fmt.Printf("加载配置成功\n%v", cfg)
	log.Info("加载配置成功", zap.Any("config", scfg.cfg))

	// registry的环境
	scfg.cfg.ENVIRON = os.Getenv(cst.ENV_ENVIRON)
	scfg.cfg.Registry.NameId = env2.GetNacosNamespaceId()
	srv_config.LarkConfig = scfg.cfg.LarkCfg

	// cfg校验
	if scfg.cfg.Other != nil {
		if obj, ok := scfg.cfg.Other.(facade.ConfigValidator); ok {
			if err := obj.CfgCheck(); err != nil {
				return nil, err
			}
		}
	}
	scfg.cfg.ServerName = opt.ServerName
	scfg.initAuth()
	return scfg, nil
}

func (s *ServerCfg) GetOtherCfg() interface{} {
	return s.cfg.Other
}

func (s *ServerCfg) GetCfg() srv_config.ServerConfig {
	return *s.cfg
}

// MatchAuthServer 匹配服务端认证信息
//
// Parameters:
//   - endpoint string 服务端端点 serverName.serviceName.method
//
// Returns：
//   - *srv_config.AuthServer
func (s *ServerCfg) MatchAuthServer(endpoint string) *srv_config.AuthServer {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var bestPrefix string
	var cg *srv_config.AuthServer
	for k, p := range s.authServer {
		// Full match (highest priority)
		if !strings.HasSuffix(k, "*") && k == endpoint {
			return p
		}

		// Prefix match
		if strings.HasSuffix(k, "*") {
			prefix := strings.TrimSuffix(k, "*")
			if strings.HasPrefix(endpoint, prefix) {
				// choose the longest prefix
				if len(prefix) > len(bestPrefix) {
					bestPrefix = prefix
					cg = p
				}
			}
		}
	}
	return cg
}

// MatchAuthClient 匹配客户端认证信息
//
// Parameters:
//   - endpoint string 服务端端点 serverName.serviceName.method
//
// Returns：
//   - *srv_config.AuthClient
func (s *ServerCfg) MatchAuthClient(endpoint string) *srv_config.AuthClient {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var bestPrefix string
	var cg *srv_config.AuthClient
	for k, p := range s.authClient {
		// Full match (highest priority)
		if !strings.HasSuffix(k, "*") && k == endpoint {
			return p
		}

		// Prefix match
		if strings.HasSuffix(k, "*") {
			prefix := strings.TrimSuffix(k, "*")
			if strings.HasPrefix(endpoint, prefix) {
				// choose the longest prefix
				if len(prefix) > len(bestPrefix) {
					bestPrefix = prefix
					cg = p
				}
			}
		}
	}
	return cg
}

// 从配置中心加载配置
func (s *ServerCfg) initFromCfgCenter(opt *option.Options, cfgType string, url string) (err error) {

	if cfgType == "etcd" {
		err = s.initFromEtcd(url, opt)
	} else {
		err = s.initFromNacos(url, opt)
	}
	if err != nil {
		return
	}
	return
}

// initFromNacos
//
//	@Description: 从nacos加载配置
//	@Author yw 2024-06-27 10:47:12
//	@return *srv_config.ServerConfig
//	@return error
func (s *ServerCfg) initFromNacos(urls string, opt *option.Options) error {

	// 获取 nacos 的账密：env 可为凭据引用 / 明文 / 空，见 loadNacosCredential
	cred, err := loadNacosCredential(context.Background())
	if err != nil {
		log.Error("[Init] 获取 Nacos 账密失败", zap.Error(err))
		return err
	}
	nameId := env2.GetNacosNamespaceId()
	if nameId == "" {
		return errors.Errorf("缺少环境变量 - [%s]，使用 Nacos 时必须显式指定 NamespaceId", cst.ENV_NACOS_NAMESPACE_ID)
	}
	//初始化配置
	err = nacos.SyncNacos(&srv_config.RegistryCfg{
		Hosts:   strings.Split(urls, ","),
		Account: cred.account,
		Pwd:     cred.password,
		NameId:  nameId,
	})
	if err != nil {
		return err
	}
	// 初始化多命名空间租户
	if err := mul_tenant.StartListen(srv_config.RegistryCfg{
		Hosts:   strings.Split(urls, ","),
		Account: cred.account,
		Pwd:     cred.password,
	}); err != nil {
		return err
	}
	s.cfg = new(srv_config.ServerConfig)
	//获取配置
	s.cfg.Other = make(map[string]interface{})
	if len(s.cfg.Registry.NameId) == 0 {
		s.cfg.Registry.NameId = os.Getenv(cst.ENV_ENVIRON)
	}
	err = nacos.GetNacosCfgData(fmt.Sprintf("%v.yaml", opt.ServerName), "DEFAULT_GROUP", s.cfg)
	if err != nil {
		log.Error("从nacos读取配置错误", zap.Error(err))
		return err
	}
	if s.defaultCustomerCfg != nil {
		// 解析自定义配置
		if err = s.nacosOtherConfigUnmarshal(s.cfg); err != nil {
			return err
		}
	}
	log.Info("从nacos读取配置成功")
	// 监听 nacos配置变动
	if err := s.listenNacosConfig(opt.ServerName); err != nil {
		log.Error("NacsoListenError", zap.Error(err))
	}
	return nil
}

// 解析 nacos 的 other 自定义配置
func (s *ServerCfg) nacosOtherConfigUnmarshal(cfg *srv_config.ServerConfig) (err error) {

	data, _ := json.Marshal(cfg.Other)
	return updateFromJSON(string(data))
}

// nacos 配置监听
func (s *ServerCfg) listenNacosConfig(serverName string) error {
	return nacos.ConfigListen[*srv_config.ServerConfig](fmt.Sprintf("%v.yaml", serverName), "DEFAULT_GROUP", func(data *srv_config.ServerConfig) {

		//	执行加载配置 操作
		s.reloadCfg(data)
	})
}

// 重新加载配置
func (s *ServerCfg) reloadCfg(newCfg *srv_config.ServerConfig) {
	s.lock.Lock()
	defer s.lock.Unlock()
	// fixme 暂时只更新 rpc 相关的 auth 配置。其他配置如果想实时更新，后续修改代码中修改中添加
	s.cfg.RpcServer.Server = newCfg.RpcServer.Server
	s.cfg.RpcServer.Client = newCfg.RpcServer.Client
	//重新初始化 auth
	s.initAuth()

	// 解析自定义配置：未注册自定义配置的服务（多为无 other 段）跳过，与启动路径保持一致
	if s.defaultCustomerCfg == nil {
		return
	}
	if err := s.nacosOtherConfigUnmarshal(newCfg); err != nil {
		log.Error("NacosListenParseOtherCfgError", zap.Error(err))
	}
}

// initFromEtcd
//
//	@Description: 从etcd加载配置
//	@Author yw 2024-06-27 10:46:42
//	@return *srv_config.ServerConfig
//	@return error
func (s *ServerCfg) initFromEtcd(url string, opt *option.Options) error {
	err := etcd.SyncETCDConfig(strings.Split(url, ","))
	if err != nil {
		return err
	}
	key := etcd.GetServerConfigKey(opt.ServerName)
	log.Info("获取etcd配置", zap.String("key", key))
	s.cfg = new(srv_config.ServerConfig)
	s.cfg.Other = make(map[string]interface{})
	//解析配置
	if err := etcd.GetEtcdCfgData(key, s.cfg); err != nil {
		log.Warn("读取etcd服务配置错误", zap.Error(err))
		return err
	}

	if s.defaultCustomerCfg != nil {
		data, _ := json.Marshal(s.cfg.Other)
		if err := updateFromJSON(string(data)); err != nil {
			return err
		}
	}

	log.Info("从etcd读取配置成功")
	return nil
}

// 从配置文件加载配置
func (s *ServerCfg) initFromCfgFile(opt *option.Options) error {
	s.cfg = new(srv_config.ServerConfig)
	s.cfg.Other = make(map[string]interface{})
	raw, err := os.ReadFile(opt.ConfigFile)
	if err != nil {
		log.Error("从文件读取配置错误", zap.Error(err))
		return err
	}
	// 本地配置同样过凭据引用解析：明文值原样透传，写了引用也能工作；反序列化错误已脱去凭据值
	if err = secret.UnmarshalYAML(context.Background(), string(raw), s.cfg); err != nil {
		log.Error("[initFromCfgFile] 从文件读取配置错误", zap.Error(err))
		return err
	}
	//
	if s.defaultCustomerCfg != nil {
		data, _ := json.Marshal(s.cfg.Other)
		if err := updateFromJSON(string(data)); err != nil {
			return err
		}
	}
	log.Info("从配置文件读取配置成功")
	return nil
}

// 对认证信息进行初始化
func (s *ServerCfg) initAuth() {
	s.authServer = make(map[string]*srv_config.AuthServer)
	// 初始化服务端
	for _, v := range s.cfg.RpcServer.Server.Auths {
		for _, k := range v.AuthEndpoint {
			mp := make(map[string]string)
			for _, as := range v.AuthSecret {

				mp[as] = maskString(as)
			}
			v.AuthMap = mp
			s.authServer[k] = &v
		}
	}
	// 初始化客户端
	s.authClient = make(map[string]*srv_config.AuthClient)
	for _, v := range s.cfg.RpcServer.Client.Auths {
		for _, k := range v.AuthEndpoint {
			s.authClient[k] = &v
		}
	}
}

func maskString(s string) string {
	r := []rune(s)
	n := len(r)

	if n <= 8 {
		// 不足 8 位时，只保留首尾各 1 位，其余用 *
		if n <= 2 {
			return s // 太短就不隐藏
		}
		return string(r[0]) + repeat('*', n-2) + string(r[n-1])
	}

	// 长度 > 8 正常隐藏
	prefix := r[:4]
	suffix := r[n-4:]
	maskLen := n - 8

	return string(prefix) + repeat('*', maskLen) + string(suffix)
}

func repeat(s rune, count int) string {
	b := make([]rune, count)
	for i := range b {
		b[i] = s
	}
	return string(b)
}
