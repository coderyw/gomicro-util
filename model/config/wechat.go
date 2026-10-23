/**
 * @Author: yinwei
 * @Description:
 * @File: wechat
 * @Version: 1.0.0
 * @Date: 2023/10/26 17:06
 */
package config

type WechatConfig struct {
	RequestUrl          string `json:"request_url"`
	AccessTokenUrl      string `json:"access_token_url"`
	PhoneNumReqUrl      string `json:"phone_num_req_url"`
	UserInfoUrl         string `json:"user_info_url"`
	H5CallBackUrl       string `json:"h5_call_back_url"`
	Oauth2Url           string `json:"oauth_2_url"`
	OauthAccessTokenUrl string `json:"oauth_access_token_url"`
	TicketUrl           string `json:"ticket_url"`
	// 配置废弃
	Apps  []WechatAppCfg `json:"apps"`
	H5s   []WechatAppCfg `json:"h5s"`
	App2s []WechatAppCfg `json:"app2s"`
}

type WechatAppCfg struct {
	AppId      string `json:"app_id"`
	Secret     string `json:"secret"`
	NotifyUrl  string `json:"notify_url"`
	ClientType int32  `json:"client_type"`
}

/**

{"config_name":"微信配置","desc":"","server":["ysw-user-go"],"data":{"request_url":"https://api.weixin.qq.com/sns/jscode2session","access_token_url":"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&amp;appid=%s&amp;secret=%s","phone_num_req_url":"https://api.weixin.qq.com/wxa/business/getuserphonenumber","user_info_url":"https://api.weixin.qq.com/sns/userinfo","h5_call_back_url":"https://server.linkysw.cloud/api/ysw/user/auth/callback","oauth_2_url":"https://open.weixin.qq.com/connect/oauth2/authorize","oauth_access_token_url":"https://api.weixin.qq.com/sns/oauth2/access_token","ticket_url":"https://api.weixin.qq.com/cgi-bin/ticket/getticket","apps":[{"app_id":"","secret":"","notify_url":"https://server.linkysw.cloud/api/ysw/wx/pay/notify0"},{"app_id":"","secret":"","notify_url":"https://server.linkysw.cloud/api/ysw/wx/pay/notify1"}],"h5s":[{"app_id":"","secret":"","notify_url":"https://server.linkysw.cloud/api/ysw/wx/pay/notify2"},{"app_id":"","secret":"","notify_url":"https://server.linkysw.cloud/api/ysw/wx/pay/notify3"},{"app_id":"","secret":"","notify_url":"https://server.linkysw.cloud/api/ysw/wx/pay/notify4"}],"app2s":[{"app_id":"","secret":"","notify_url":"https://server.linkysw.cloud/api/ysw/wx/pay/notify0","client_type":0},{"app_id":"","secret":"","notify_url":"https://server.linkysw.cloud/api/ysw/wx/pay/notify1","client_type":0},{"app_id":"","secret":"","notify_url":"https://server.linkysw.cloud/api/ysw/wx/pay/notify4","client_type":1}]}}


*/
