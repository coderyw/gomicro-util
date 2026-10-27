// Package redis
// @Author: yinwei
// @File: cache_test
// @Version: 1.0.0
// @Date: 2024/7/8 18:04

package redis

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	facade2 "github.com/coderyw/gomicro-util/library/db/cache/redis/facade"
	"github.com/coderyw/gomicro-util/library/db/cache/redis/test_model"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/trace"
	"github.com/coderyw/gomicro-util/model/facade"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 全局缓存
var typeCache sync.Map
var value reflect.Value

// 泛型函数，通过反射初始化变量并缓存
func InitWithReflection[T any]() T {
	var t T
	// 获取类型的反射信息
	typ := reflect.TypeOf(t)
	var isPtr bool
	if typ.Kind() == reflect.Ptr {
		isPtr = true
		typ = typ.Elem()
	}

	var nt reflect.Value
	if typ.Kind() == reflect.Ptr {
		nt = reflect.New(typ)
	} else if typ.Kind() == reflect.Struct {
		nt = reflect.New(typ).Elem()
	} else if typ.Kind() == reflect.Slice {
		return reflect.MakeSlice(typ, 0, 0).Interface().(T)
	}
	// 存入缓存
	if isPtr {
		return nt.Addr().Interface().(T)
	} else {
		return nt.Interface().(T)
	}
}

func TestInitWithReflection(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		a := InitWithReflection[int]()
		fmt.Printf("%v\n", a)
		b := 12
		fmt.Printf("%v\n", b)
	})
	t.Run("string", func(t *testing.T) {
		a := InitWithReflection[string]()
		fmt.Printf("%v\n", a)
	})
	t.Run("struct", func(t *testing.T) {
		a := InitWithReflection[Ab]()
		fmt.Printf("%v\n", a)
	})
	t.Run("*struct", func(t *testing.T) {
		a := InitWithReflection[*Ab]()
		b := InitWithReflection[*Ab]()
		fmt.Printf("%p\n", a)
		fmt.Printf("%p\n", b)
	})
	t.Run("slice", func(t *testing.T) {
		a := InitWithReflection[[]int]()
		t.Log(a)
		t.Logf("%p\n", a)
	})
}

type Ab struct {
	A int
}

func Te[T any]() (t T) {

	ty := reflect.TypeOf(t)
	//fmt.Println(ty)

	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}

	a := reflect.New(ty).Interface()
	//fmt.Println(a)

	return a.(T)
}

func TestGetHMCacheFieldInteger(t *testing.T) {
	Init(t)
	type args struct {
		ctx   context.Context
		key   string
		field string
		load  func() (map[string]interface{}, error)
		exp   time.Duration
	}
	tests := []struct {
		name       string
		args       args
		wantTarget int64
		wantErr    bool
	}{
		{name: "加载Key不存在的数据", args: args{
			ctx:   context.Background(),
			key:   "test_hash",
			field: "a",
			load: func() (map[string]interface{}, error) {
				return map[string]interface{}{
					"a": int64(123),
					"b": "dd",
				}, nil
			},
			exp: -1,
		}, wantTarget: 123, wantErr: false},
		{name: "加载field不存在的数据", args: args{
			ctx:   context.Background(),
			key:   "test_hash",
			field: "a1",
			load: func() (map[string]interface{}, error) {
				return map[string]interface{}{
					"a1": int64(123),
					"b":  "dd",
				}, nil
			},
			exp: -1,
		}, wantTarget: 123, wantErr: false},
		{name: "load错误", args: args{
			ctx:   context.Background(),
			key:   "test_hash1",
			field: "a1",
			load: func() (map[string]interface{}, error) {
				return map[string]interface{}{
					"a1": int64(123),
					"b":  "dd",
				}, fmt.Errorf("record not found")
			},
			exp: -1,
		}, wantTarget: 0, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTarget, err := GetHMCacheFieldInteger[int64](tt.args.ctx, GetRedis(tt.args.ctx).(*Redis), tt.args.key, tt.args.field, tt.args.load, tt.args.exp)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetHMCacheFieldInteger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotTarget != tt.wantTarget {
				t.Errorf("GetHMCacheFieldInteger() gotTarget = %v, want %v", gotTarget, tt.wantTarget)
			}
		})
	}
}

func TestGetObjHMCache(t *testing.T) {
	Init(t)
	type args[T facade.HMField] struct {
		ctx    context.Context
		key    string
		load   func() (map[string]interface{}, error)
		exp    time.Duration
		target T
		fields []string
	}
	type testCase[T facade.HMField] struct {
		name    string
		args    args[T]
		wantErr bool
	}
	tests := []testCase[*test_model.Hmtest]{
		{name: "加载部分key", args: args[*test_model.Hmtest]{
			ctx: context.Background(),
			key: "hmtest",
			load: func() (map[string]interface{}, error) {
				return map[string]interface{}{
					"v1": 3333,
					"v2": "fwefw",
				}, nil
			},
			exp:    -1,
			target: &test_model.Hmtest{},
		}, wantErr: false},
		{name: "加载不存在的key", args: args[*test_model.Hmtest]{
			ctx: context.Background(),
			key: "hmtest11",
			load: func() (map[string]interface{}, error) {
				return map[string]interface{}{
					"v1": 3333,
					"v2": "fwefw",
				}, nil
			},
			exp:    -1,
			target: &test_model.Hmtest{},
		}, wantErr: false},
		{name: "加载不存在的field", args: args[*test_model.Hmtest]{
			ctx: context.Background(),
			key: "hmtest11",
			load: func() (map[string]interface{}, error) {
				return map[string]interface{}{
					"v1": 3333,
					"v2": "fwefw",
					"v3": "123.4",
				}, nil
			},
			exp:    -1,
			target: &test_model.Hmtest{},
			fields: []string{"v3"},
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := GetObjHMCache[*test_model.Hmtest](tt.args.ctx, GetRedis(tt.args.ctx), tt.args.key, tt.args.load, tt.args.exp, tt.args.fields...); (err != nil) != tt.wantErr {
				t.Errorf("GetObjHMCache() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				t.Log(got)
			}

		})
	}
}

type Kv struct {
	A string
	B decimal.Decimal
}

func (k *Kv) Set(key string, b decimal.Decimal) {
	k.A = key
	k.B = b
}
func (k *Kv) Get() (string, decimal.Decimal) {
	return k.A, k.B
}

func TestGetZSetList(t *testing.T) {
	Init(t)
	type args[T facade2.ZSetAction] struct {
		ctx  context.Context
		rd   facade2.RedisInterface
		key  string
		load func(ctx context.Context) ([]T, error)
		exp  time.Duration
		vs   bool
	}
	type testCase[T facade2.ZSetAction] struct {
		name    string
		args    args[T]
		wantErr bool
	}

	tests := []testCase[*Kv]{
		{name: "升序", args: struct {
			ctx  context.Context
			rd   facade2.RedisInterface
			key  string
			load func(ctx context.Context) ([]*Kv, error)
			exp  time.Duration
			vs   bool
		}{
			ctx: context.Background(),
			rd:  GetRedis(context.Background()),
			key: "test-getzsetlist",
			load: func(ctx context.Context) ([]*Kv, error) {
				return []*Kv{
					{
						A: "a",
						B: decimal.NewFromFloat(1.1),
					},
					{
						A: "b",
						B: decimal.NewFromInt(12),
					},
				}, nil
			},
			exp: 60 * time.Second,
		}, wantErr: false},
		{name: "降序", args: struct {
			ctx  context.Context
			rd   facade2.RedisInterface
			key  string
			load func(ctx context.Context) ([]*Kv, error)
			exp  time.Duration
			vs   bool
		}{
			ctx: context.Background(),
			rd:  GetRedis(context.Background()),
			key: "test-getzsetlist",
			load: func(ctx context.Context) ([]*Kv, error) {
				return []*Kv{
					{
						A: "a",
						B: decimal.NewFromFloat(1.1),
					},
					{
						A: "b",
						B: decimal.NewFromInt(12),
					},
				}, nil
			},
			vs:  true,
			exp: 60 * time.Second,
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetZSetList(tt.args.ctx, tt.args.rd, tt.args.key, tt.args.load, tt.args.exp, tt.args.vs)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetZSetList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			log.InfoCtx(tt.args.ctx, "GetZSetList", zap.Any("got", got))

		})
	}
}

func TestGetFromCacheOrDb(t *testing.T) {
	Init(t)

	type StructA struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	type args[T any] struct {
		ctx           context.Context
		rd            facade2.RedisInterface
		key           string
		loadFromCache LoadFunc[T]
		loadFromDb    LoadFunc[T]
		saveCache     SaveFunc[T]
	}
	type testCase[T any] struct {
		name       string
		args       args[T]
		wantTarget T
		wantErr    bool
	}
	tests := []testCase[int64]{
		{name: "缓存不存在", args: args[int64]{
			ctx: trace.GetTraceCtx(context.Background()),
			rd:  GetRedis(context.Background()),
			key: "test:int",
			loadFromCache: func(ctx context.Context) (int64, error) {
				return GetRedis(context.Background()).GetValueByInt64(context.Background(), "", "test:int")
			},
			loadFromDb: func(ctx context.Context) (int64, error) {
				return 12, nil
			},
			saveCache: func(ctx context.Context, i int64) error {
				return GetRedis(context.Background()).SetValue(context.Background(), "", "test:int", i, 12*time.Second)
			},
		}, wantTarget: 12, wantErr: false},
		{name: "缓存存在", args: args[int64]{
			ctx: trace.GetTraceCtx(context.Background()),
			rd:  GetRedis(context.Background()),
			key: "test:int:forever",
			loadFromCache: func(ctx context.Context) (int64, error) {
				return GetRedis(context.Background()).GetValueByInt64(context.Background(), "", "test:int:forever")
			},
			loadFromDb: func(ctx context.Context) (int64, error) {
				return 223, nil
			},
			saveCache: func(ctx context.Context, i int64) error {
				return GetRedis(context.Background()).SetValue(context.Background(), "", "test:int:forever", i, 0)
			},
		}, wantTarget: 223, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTarget, err := GetFromCacheOrDb(tt.args.ctx, tt.args.rd, tt.args.key, tt.args.loadFromCache, tt.args.loadFromDb, tt.args.saveCache)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFromCacheOrDb() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotTarget, tt.wantTarget) {
				t.Errorf("GetFromCacheOrDb() gotTarget = %v, want %v", gotTarget, tt.wantTarget)
			}
		})
	}

	testsStruct := []testCase[StructA]{
		{name: "struct缓存不存在", args: args[StructA]{
			ctx: trace.GetTraceCtx(context.Background()),
			rd:  GetRedis(context.Background()),
			key: "test:struct",
			loadFromCache: func(ctx context.Context) (StructA, error) {
				val, err := GetRedis(context.Background()).GetValue(context.Background(), "", "test:struct")
				if err != nil {
					return StructA{}, err
				}
				obj := &StructA{}
				return *obj, json.Unmarshal([]byte(val), obj)
			},
			loadFromDb: func(ctx context.Context) (StructA, error) {
				return StructA{
					Name: "name",
					Age:  3333,
				}, nil
			},
			saveCache: func(ctx context.Context, i StructA) error {
				data, err := json.Marshal(i)
				if err != nil {
					return err
				}
				return GetRedis(context.Background()).SetValue(context.Background(), "", "test:struct", string(data), 12*time.Second)
			},
		}, wantTarget: StructA{
			Name: "name",
			Age:  3333,
		}, wantErr: false},
		{name: "struct缓存不存在,db也不存在,返回recordNotFound", args: args[StructA]{
			ctx: trace.GetTraceCtx(context.Background()),
			rd:  GetRedis(context.Background()),
			key: "test:struct:不存在",
			loadFromCache: func(ctx context.Context) (StructA, error) {
				val, err := GetRedis(context.Background()).GetValue(context.Background(), "", "test:struct:不存在")
				if err != nil {
					return StructA{}, err
				}
				obj := &StructA{}
				return *obj, json.Unmarshal([]byte(val), obj)
			},
			loadFromDb: func(ctx context.Context) (StructA, error) {
				return StructA{
					Name: "",
					Age:  0,
				}, gorm.ErrRecordNotFound
			},
			saveCache: func(ctx context.Context, i StructA) error {
				data, err := json.Marshal(i)
				if err != nil {
					return err
				}
				return GetRedis(ctx).SetValue(context.Background(), "", "test:struct:不存在", string(data), 12*time.Second)
			},
		}, wantTarget: StructA{
			Name: "",
			Age:  0,
		}, wantErr: true},
		{name: "struct缓存不存在,db也不存在,返回空结构体", args: args[StructA]{
			ctx: trace.GetTraceCtx(context.Background()),
			rd:  GetRedis(context.Background()),
			key: "test:struct:不存在",
			loadFromCache: func(ctx context.Context) (StructA, error) {
				val, err := GetRedis(context.Background()).GetValue(context.Background(), "", "test:struct:不存在")
				if err != nil {
					return StructA{}, err
				}
				obj := &StructA{}
				return *obj, json.Unmarshal([]byte(val), obj)
			},
			loadFromDb: func(ctx context.Context) (StructA, error) {
				return StructA{
					Name: "",
					Age:  0,
				}, nil
			},
			saveCache: func(ctx context.Context, i StructA) error {
				data, err := json.Marshal(i)
				if err != nil {
					return err
				}
				return GetRedis(context.Background()).SetValue(context.Background(), "", "test:struct:不存在", string(data), 12*time.Second)
			},
		}, wantTarget: StructA{
			Name: "",
			Age:  0,
		}, wantErr: false},
	}
	for _, tt := range testsStruct {
		t.Run(tt.name, func(t *testing.T) {
			gotTarget, err := GetFromCacheOrDb(tt.args.ctx, tt.args.rd, tt.args.key, tt.args.loadFromCache, tt.args.loadFromDb, tt.args.saveCache)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFromCacheOrDb() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotTarget, tt.wantTarget) {
				t.Errorf("GetFromCacheOrDb() gotTarget = %v, want %v", gotTarget, tt.wantTarget)
			}
		})
	}

	testsPtrStruct := []testCase[*StructA]{
		{name: "*struct缓存不存在", args: args[*StructA]{
			ctx: trace.GetTraceCtx(context.Background()),
			rd:  GetRedis(context.Background()),
			key: "test:struct:ptr",
			loadFromCache: func(ctx context.Context) (*StructA, error) {
				val, err := GetRedis(ctx).GetValue(context.Background(), "", "test:struct:ptr")
				if err != nil {
					return nil, err
				}
				obj := &StructA{}
				return obj, json.Unmarshal([]byte(val), obj)
			},
			loadFromDb: func(ctx context.Context) (*StructA, error) {
				return &StructA{
					Name: "name",
					Age:  3333,
				}, nil
			},
			saveCache: func(ctx context.Context, i *StructA) error {
				data, err := json.Marshal(i)
				if err != nil {
					return err
				}
				return GetRedis(ctx).SetValue(context.Background(), "", "test:struct:ptr", string(data), 12*time.Second)
			},
		}, wantTarget: &StructA{
			Name: "name",
			Age:  3333,
		}, wantErr: false},
		{name: "*struct缓存不存在,db也不存在,返回recordNotFound", args: args[*StructA]{
			ctx: trace.GetTraceCtx(context.Background()),
			rd:  GetRedis(context.Background()),
			key: "test:struct:ptr:不存在",
			loadFromCache: func(ctx context.Context) (*StructA, error) {
				val, err := GetRedis(ctx).GetValue(context.Background(), "", "test:struct:ptr:不存在")
				if err != nil {
					return nil, err
				}
				obj := &StructA{}
				return obj, json.Unmarshal([]byte(val), obj)
			},
			loadFromDb: func(ctx context.Context) (*StructA, error) {
				return nil, gorm.ErrRecordNotFound
			},
			saveCache: func(ctx context.Context, i *StructA) error {
				data, err := json.Marshal(i)
				if err != nil {
					return err
				}
				return GetRedis(ctx).SetValue(context.Background(), "", "test:struct:ptr:不存在", string(data), 12*time.Second)
			},
		}, wantTarget: nil, wantErr: true},
		{name: "*struct缓存不存在,db也不存在,返回空结构体", args: args[*StructA]{
			ctx: trace.GetTraceCtx(context.Background()),
			rd:  GetRedis(context.Background()),
			key: "test:struct:ptr:不存在",
			loadFromCache: func(ctx context.Context) (*StructA, error) {
				val, err := GetRedis(ctx).GetValue(context.Background(), "", "test:struct:ptr:不存在")
				if err != nil {
					return nil, err
				}
				obj := &StructA{}
				return obj, json.Unmarshal([]byte(val), obj)
			},
			loadFromDb: func(ctx context.Context) (*StructA, error) {
				return nil, nil
			},
			saveCache: func(ctx context.Context, i *StructA) error {
				data, err := json.Marshal(i)
				if err != nil {
					return err
				}
				return GetRedis(ctx).SetValue(context.Background(), "", "test:struct:ptr:不存在", string(data), 12*time.Second)
			},
		}, wantTarget: nil, wantErr: false},
	}
	for _, tt := range testsPtrStruct {
		t.Run(tt.name, func(t *testing.T) {
			gotTarget, err := GetFromCacheOrDb(tt.args.ctx, tt.args.rd, tt.args.key, tt.args.loadFromCache, tt.args.loadFromDb, tt.args.saveCache)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFromCacheOrDb() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotTarget, tt.wantTarget) {
				t.Errorf("GetFromCacheOrDb() gotTarget = %v, want %v", gotTarget, tt.wantTarget)
			}
		})
	}
}
