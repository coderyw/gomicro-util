// Package etcd_config
// @Author: yinwei
// @File: user_notify_test.go
// @Version: 1.0.0
// @Date: 2023/11/9 18:43

package config

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestUserNotifyConfig_UnmarshalJSON(t *testing.T) {
	data := `{"wx_notify_cfg":{"host":"https://api.weixin.qq.com/cgi-bin/message/subscribe/send","pages":{"1":{"lang":"zh_CN","miniprogramState":"developer","page":"/pages/tabbar/home"}},"template_id":"NXnCrG-AiwqDJ-A770qMP923nOuXBKsTWwNSc-aRHEE"},"uni_notify_cfg":{"app_id":"__UNI__FA3BDF6","host":"https://fc-mp-3b1ca11d-0a0d-487b-bbf3-d3237c2e480c.next.bspapp.com","path":"push"}}`
	obj := new(UserNotifyConfig)
	err := json.Unmarshal([]byte(data), obj)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", obj.UniNotifyCfg)
	fmt.Printf("%+v\n", obj.WxNotifyCfg)
}

func Test1(t *testing.T) {
	data := `{"server_name":"ysw-gateway-go","environ":"dev","http":{"host":":8099"},"etcd":{"etcd_hosts":["127.0.0.1:2379"]},"rpc_server":{"listen_on":"0.0.0.0:9899"},"log":{"development":true,"disableStacktrace":false}}`
	cfg := new(ServerConfig)
	err := json.Unmarshal([]byte(data), cfg)
	fmt.Println(*cfg)
	fmt.Println(err)
}
