/**
 * @Author: yinwei
 * @Description:
 * @File: config_cst
 * @Version: 1.0.0
 * @Date: 2023/10/24 16:26
 */
package cst

// etcd 对应key
const (
	ETCD_SERVER_CFG_KEY  = "server_cfg"
	ETCD_MYSQL_kEY       = "mysql"
	ETCD_REDIS_KEY       = "redis"
	ETCD_ALI_SMS_VER_KEY = "ali_sms_ver"
	ETCD_WECHAT_KEY      = "wechat"
	ETCD_ALI_OSS_KEY     = "ali_oss"
	ETCD_BEANSTALK_KEY   = "beanstalk"     // beanstalk配置key，延迟队列
	ETCD_RABBIT_MQ       = "rabbitmq"      //rabbitmq配置
	ETCD_CLIENT_NOTIFY   = "client_notify" // 用户推送配置
)
