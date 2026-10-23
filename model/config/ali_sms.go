/**
 * @Author: yinwei
 * @Description: 阿里短信配置
 * @File: ali_sms
 * @Version: 1.0.0
 * @Date: 2023/10/25 16:00
 */
package config

// {"config_name":"阿里短信配置","desc":"","server":["ysw-user-go"],"data":{"access_key":"","access_secret":"", "sign_name":"linkysw","template_code":""}}
type AliSmsCfg struct {
	AccessKey    string `json:"access_key"`
	AccessSecret string `json:"access_secret"`
	SignName     string `json:"sign_name"`
	TemplateCode string `json:"template_code"`
}
