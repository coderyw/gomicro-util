package config

import (
	"reflect"
	"testing"
)

type CustomerConfig struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
	Sex  string `json:"sex"`
}

var defaultVal = CustomerConfig{
	Name: "default",
	Age:  18,
	Sex:  "male",
}

func TestSetCustomerConfig(t *testing.T) {
	initCustomerConfig(defaultVal, func(value any) {
		val, ok := value.(CustomerConfig)
		if !ok {
			t.Error("value is not CustomerConfig")
		} else {
			t.Logf("%+v\n", val)
		}
	})
	t.Logf("%+v\n", GetCustomerConfig())
	err := updateFromJSON("{\"name\":\"test\",\"age\":20}")
	if err != nil {
		t.Error(err)
	}
	t.Logf("%+v\n", GetCustomerConfig())
}

func TestSetPtrCustomerConfig(t *testing.T) {
	if err := initCustomerConfig(&defaultVal, func(value any) {
		val, ok := value.(CustomerConfig)
		if !ok {
			t.Error("value is not CustomerConfig")
		} else {
			t.Logf("%+v\n", val)
		}
	}); err != nil {
		t.Error(err)
	}
	t.Logf("%+v\n", GetCustomerConfig())
	err := updateFromJSON("{\"name\":\"test\",\"age\":20}")
	if err != nil {
		t.Error(err)
	}
	t.Logf("%+v\n", GetCustomerConfig())
}

func TestGetCustomerConfig(t *testing.T) {
	tests := []struct {
		name    string
		Json    string
		Default any
		want    any
	}{
		{
			name: "DefaultIsPtr",
			Json: `{"name":"test","age":20}`,
			Default: &CustomerConfig{
				Name: "default",
				Age:  18,
				Sex:  "male",
			},
			want: CustomerConfig{
				Name: "test",
				Age:  20,
				Sex:  "male",
			},
		},
		{
			name: "DefaultIsNotPtr",
			Json: `{"name":"test","age":20}`,
			Default: CustomerConfig{
				Name: "default",
				Age:  18,
				Sex:  "male",
			},
			want: CustomerConfig{
				Name: "test",
				Age:  20,
				Sex:  "male",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := initCustomerConfig(tt.Default, func(value any) {
				val, ok := value.(CustomerConfig)
				if !ok {
					t.Error("value is not CustomerConfig")
				} else {
					t.Logf("%+v\n", val)
				}
			}); err != nil {
				t.Error(err)
			}
			err := updateFromJSON(tt.Json)
			if err != nil {
				t.Error(err)
			}
			if got := GetCustomerConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCustomerConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 未注册自定义配置的服务（多为无 other 段）收到热更推送时，updateFromJSON 收到 "null" 不得 panic
func TestUpdateFromJSON_未注册自定义配置时安全跳过(t *testing.T) {
	saved := cc
	cc = nil
	defer func() { cc = saved }()
	for _, in := range []string{"null", "", "{}"} {
		if err := updateFromJSON(in); err != nil {
			t.Fatalf("入参 %q 期望静默返回 nil，实际 %v", in, err)
		}
	}
}

// 已注册自定义配置时，Other 为空（marshal 得到 "null"）也不得 panic，且保留默认值
func TestUpdateFromJSON_Other为空保留默认值(t *testing.T) {
	saved := cc
	defer func() { cc = saved }()
	if err := initCustomerConfig(defaultVal, nil); err != nil {
		t.Fatal(err)
	}
	if err := updateFromJSON("null"); err != nil {
		t.Fatal(err)
	}
	got, ok := GetCustomerConfig().(CustomerConfig)
	if !ok || got.Name != defaultVal.Name || got.Age != defaultVal.Age {
		t.Fatalf("期望保留默认值 %+v，实际 %+v", defaultVal, got)
	}
}
