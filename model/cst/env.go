package cst

const (
	ENV_ENVIRON            = "ENVIRON"
	ENV_NACOS_NAMESPACE_ID = "NACOS_NAMESPACE_ID"
	ENV_ETCD_URLS          = "ETCD_URLS"
	ENV_NACOS_URLS         = "NACOS_URLS"
	NACOS_USERNAME         = "NACOS_USERNAME"
	NACOS_PASSWORD         = "NACOS_PASSWORD"
	NACOS_NAMEID           = "NACOS_NAME_ID"
	ENV_POD_NAME           = "POD_NAME"
	// 服务所属集群
	ENV_NAMESPACE = "NAMESPACE"
	// 是否开启监听商户命名空间配置
	ENV_ENABLE_LISTEN_TENANT = "ENABLE_LISTEN_TENANT"
	// 雪花 ID Redis hash tag：0（默认）开启，1 使用旧版 eonId 前缀。
	ENV_DISABLE_EONID_HASHTAG = "DISABLE_EONID_HASHTAG"
	// 兜底租户名。设置后该 pod 数据层不加租户前缀，保持原始命名（独享型租户如 NS 的专属部署用）
	ENV_TENANT_FALLBACK = "TENANT_FALLBACK"
	// 中间件资源名中的租户段格式：lower（默认）、upper、original。
	ENV_TENANT_NAME_CASE = "TENANT_NAME_CASE"
	// 云厂商标识（huawei/aws/azure/google/tencent/aliyun）。显式配置优先于自动探测，云迁移或特殊部署时用于人工覆盖
	ENV_CLOUD_TYPE = "CLOUD_TYPE"

	// 服务id
	ENV_ID = "ID"
	// 环境变量 服务名
	//ENV_SERVER_NAME = "SERVER_NAME"
	// 环境变量 服务版本
	ENV_SERVER_VER = "SERVER_VERSION"
	// 凭据引用解析的重试整体 deadline（秒），0 为不重试；未设置用组件默认值
	ENV_SECRET_RETRY_DEADLINE_SECONDS = "SECRET_RETRY_DEADLINE_SECONDS"
	// 启动时等待 istio sidecar 就绪的上限（秒），0 为不等待；未设置用框架默认值
	ENV_STARTUP_PROXY_WAIT_SECONDS = "STARTUP_PROXY_WAIT_SECONDS"
	// Nacos 账密是否允许明文（默认允许）；为 false 时账密非引用形态直接启动失败
	ENV_NACOS_AUTH_ALLOW_PLAINTEXT = "NACOS_AUTH_ALLOW_PLAINTEXT"
)

const (
	RUNTIME_ENV_PROD      = "server-prod"      //生产环境
	RUNTIME_ENV_PRE       = "server-pre"       //预发环境
	RUNTIME_ENV_ROADSHOW  = "server-roadshow"  //路演环境
	RUNTIME_ENV_TEST      = "server-test"      //测试环境
	RUNTIME_ENV_DEV       = "server-dev"       //开发环境
	RUNTIME_ENV_LOCAL     = "server-local"     //本地环境
	RUNTIME_ENV_TEST_AUTO = "server-test-auto" //测试环境自动化
	RUNTIME_ENV_PRE_AUTO  = "server-pre-auto"  //预发环境自动化
)
