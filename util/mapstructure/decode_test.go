package mapstructure

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
)

type CommonActivityConfigCache struct {
	Id           int64           `json:"id"`
	ActivityName string          `json:"activity_name"`
	IsStart      int16           `json:"is_start"`
	UpdatedAt    string          `json:"updated_at"`
	D            decimal.Decimal `json:"d"`
}

func TestCommonActivityConfigCache(t *testing.T) {
	m := map[string]string{
		"id":            "23423234",
		"activity_name": "test",
		"is_start":      "1",
		"updated_at":    "2023-07-08 18:04:05",
		"d":             "1.2",
	}
	resp := new(CommonActivityConfigCache)
	err := Decode(m, resp)
	fmt.Println(err)
	fmt.Println(resp)
}

func TestMapStringDecode(t *testing.T) {
	m := map[string]string{
		"a":     "a",
		"b":     "1",
		"c":     "1.2",
		"cp":    "1.44",
		"d":     "1.2",
		"dp":    "1.3",
		"e":     "2023-07-08 18:04:05",
		"args1": `{"a":"args1a","b":1,"c":1.2,"cp":1.44,"d":"1.2","e":"2023-07-01 18:04:05"}`,
	}
	obj := new(Args)

	err := Decode(m, obj)
	if err != nil {
		t.Fatal(err)
	}
	log.Info("解析完成", zap.Any("obj", obj))
}

func TestMapInterfaceDecode(t *testing.T) {
	m := map[string]interface{}{
		"a":     "a",
		"b":     "1",
		"c":     1.2,
		"cp":    1.44,
		"d":     "1.2",
		"dp":    "1.3",
		"e":     "2023-07-08 18:04:05",
		"args1": `{"a":"args1a","b":1,"c":1.2,"cp":1.44,"d":"1.2","e":"2023-07-01 18:04:05"}`,
	}
	obj := new(Args)

	err := Decode(m, obj)
	if err != nil {
		t.Fatal(err)
	}
	log.Info("解析完成", zap.Any("obj", obj))
}

type Args struct {
	A  string           `json:"a"`
	B  int8             `json:"b"`
	C  float64          `json:"c"`
	CP *float64         `json:"cp"`
	D  decimal.Decimal  `json:"d"`
	DP *decimal.Decimal `json:"dp"`
	E  time.Time        `json:"e" time_format:"2006-01-02 15:04:06"`
	F  Args1            `json:"args1" mapstructure:"-"`
}

type Args1 struct {
	A  string          `json:"a"`
	B  int8            `json:"b"`
	C  float64         `json:"c"`
	CP *float64        `json:"cp"`
	D  decimal.Decimal `json:"d"`
	E  time.Time       `json:"e"`
}

func TestJsonDecode(t *testing.T) {
	data := `{"a":"1212","b":11,"c":33,"cp":333.3,"d":"12","dp":"12","e":"2025-04-22 09:51:51","args1":{"a":"aaaa","b":34,"c":23,"cp":333.3,"d":"12","e":"2025-04-22 09:51:51"}}`
	obj := new(Args)
	err := ParseJSONToStruct(data, obj)
	if err != nil {
		t.Fatal(err)
	}
	log.Info("解析完成", zap.Any("obj", obj))
}

func TestUser(t *testing.T) {
	str := `{"id":282419,"user_id":282419,"nickname":"G282419","avatar":"1","fb_avatar":"","phone":"9100000009","gcash_phone":"","gcash_customer_id":"","withdraw_password":0,"login_password":0,"pay_account_id":0,"card_back":0,"avatar_frame":0,"app_package_name":"com.playmate.playzone","bind":[1],"type":"16384","invite_user_id":"","invite_code":"00282419","invite":{},"is_register":0,"enable":1,"recharge_amount":0,"withdraw_amount":0,"s_player":0,"c_player":0,"withdraw_model":2,"total_rounds":0,"first_rw_reward":false,"level":0,"level_max":0,"score":0,"created_at":1721289021,"status":0,"last_login_channel":"Maya","grayscale":0,"app_channel":"web","is_vip":"0","vip_start":"0","vip_end":"0","pay_account":{"email":"hcg@rummy.com","phone":"9100000009","name":"G282419"},"player_id":282419,"token":"282419:1744942612:0:76ec34f11f082b8a0b1a402d68bb633b:env-PRE"}`
	obj := new(UserCacheGo)
	err := ParseJSONToStruct(str, obj)
	if err != nil {
		t.Fatal(err)
	}
	log.Info("解析完成", zap.Any("obj", obj))
}

func TestUnmarshlSlice(t *testing.T) {
	str := `{"slic":[1,2,3,9223372036854775807]}`
	m := make(map[string]interface{})
	fmt.Println(math.MaxInt64)
	json.Unmarshal([]byte(str), &m)
	fmt.Println(m)
}

func TestARg(t *testing.T) {
	f := float64(333.3)
	d := decimal.NewFromInt(12)
	a := Args{
		A:  "1212",
		B:  11,
		C:  33,
		CP: &f,
		D:  d,
		DP: &d,
		E:  time.Now(),
		F: Args1{
			A:  "aaaa",
			B:  34,
			C:  23,
			CP: &f,
			D:  d,
			E:  time.Now(),
		},
	}
	b, _ := json.Marshal(a)
	fmt.Println(string(b))
}
