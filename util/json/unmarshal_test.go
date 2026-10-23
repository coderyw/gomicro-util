// Package json
// @Author: yinwei
// @File: test_unmarshal
// @Version: 1.0.0
// @Date: 2025/4/14 14:47

package json

import (
	"fmt"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
	"testing"
)

func init() {
	//json.RegisterExtension(&Obj{})
	json.RegisterExtension(&Json{})
	//json.RegisterExtension(&UserCacheGo{})
	//json.RegisterExtension(&PayAccount{})
}

type Obj struct {
	Json
	Id   int     `json:"id"`
	Name *string `json:"name"`
}

func TestUnmarshal(t *testing.T) {
	str := `{"id":"123","name":"ewere"}`
	o := new(Obj)
	err := json.Unmarshal([]byte(str), o)
	fmt.Println(err)
	str111 := "fwfewefwe"
	newO := Obj{
		Id:   3333,
		Name: &str111,
	}
	//fmt.Printf("%#v\n", o)
	fmt.Printf("%#v\n", newO)
	log.Info("Obj", zap.Any("got", o))
}

func TestUser(t *testing.T) {

	str := `{"id":282419,"user_id":282419,"nickname":"G282419","avatar":"1","fb_avatar":"","phone":"9100000009","gcash_phone":"","gcash_customer_id":"","withdraw_password":0,"login_password":0,"pay_account_id":0,"card_back":0,"avatar_frame":0,"app_package_name":"com.playmate.playzone","bind":[1],"type":"16384","invite_user_id":"","invite_code":"00282419","invite":null,"is_register":0,"enable":1,"recharge_amount":0,"withdraw_amount":0,"s_player":0,"c_player":0,"withdraw_model":2,"total_rounds":0,"first_rw_reward":false,"level":0,"level_max":0,"score":0,"created_at":1721289021,"status":0,"last_login_channel":"Maya","grayscale":0,"app_channel":"web","is_vip":"0","vip_start":"0","vip_end":"0","pay_account":{"email":"hcg@rummy.com","phone":"9100000009","name":"G282419"},"player_id":282419,"token":"282419:1744942612:0:76ec34f11f082b8a0b1a402d68bb633b:env-PRE"}`
	obj := new(UserCacheGo)
	err := json.Unmarshal([]byte(str), obj)
	if err != nil {
		t.Fatal(err)
	}
	log.Info("解析完成", zap.Any("obj", obj))
}
