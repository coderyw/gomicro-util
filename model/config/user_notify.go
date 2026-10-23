// Package etcd_config
// @Author: yinwei
// @File: wxpush
// @Version: 1.0.0
// @Date: 2023/11/9 15:30

package config

type Pages struct {
	// 语言
	Lang string `json:"lang,omitempty"`
	// 开发环境
	MiniprogramState string `json:"miniprogramState,omitempty"`
	// 路径
	Page string `json:"page,omitempty"`
}

type WxNotifyConfig struct {
	// 地址
	Host string `json:"host,omitempty"`
	// 跳转页面 key为类型
	Pages map[string]Pages `json:"pages,omitempty"`
	// 模板id
	TemplateId string `json:"template_id,omitempty"`
}

// unipush配置
type UniNotifyCfg struct {
	Host  string `json:"host"`
	AppId string `json:"app_id"`
	Path  string `json:"path"`
}

type UserNotifyConfig struct {
	//微信推送配置
	WxNotifyCfg *WxNotifyConfig `json:"wx_notify_cfg"`
	// unipush配置
	UniNotifyCfg *UniNotifyCfg `json:"uni_notify_cfg"`
}
