// Package test
// @Author: yinwei
// @File: reflect_test
// @Version: 1.0.0
// @Date: 2023/11/15 10:24

package test

import (
	"fmt"
	"github.com/coderyw/protobuf/keyreflect"
	"reflect"
	"testing"
	"time"
)

type A struct {
	D string `json:"d"`
}

func (a *A) Get() int {
	return 1
}
func (a *A) IsNil() bool {
	return a == nil
}

// 数组反射 获取数组元素类型
func TestReflectArr(t *testing.T) {
	var a []*A

	ta := reflect.TypeOf(a)
	tae := ta.Elem()
	fmt.Println(tae.Name())

	fmt.Println(tae.Kind())
	taee := tae.Elem()
	fmt.Println(taee.Name())
	fmt.Println(taee.Kind())

}

func TestType(t *testing.T) {
	t.Run("", func(t *testing.T) {
		REf(nil)
	})
	t.Run("", func(t *testing.T) {
		REf(new(A))
		//REf(*A)
	})
}

func REf(t interface{}) {
	tt := reflect.TypeOf(t)
	if tt.Kind() == reflect.Ptr {
		tt = tt.Elem()
	}
	fmt.Println(tt.String())
}

func TestTimeUnixMicro(t *testing.T) {
	nm := time.Now().UnixNano()

	fmt.Println(time.UnixMicro(nm / 1000))
}

type AI interface {
	Get() int
	IsNil() bool
}

func TestNIl(t *testing.T) {

	var a *A
	var ai AI
	ai = a

	var inter interface{}
	inter = ai
	if ai == nil {
		fmt.Println("ai is nil")
	} else {
		fmt.Println("ai.IsNil ", ai.IsNil())
	}
	if inter == nil {
		fmt.Println("inter is nil")
	}
}

type A1 struct {
	A2
}

type A2 struct {
	N int
}

func (a *A2) UseTable() {
	a.N = 1
}

func (a A1) Table() *A1 {
	a.A2.UseTable()
	return a.updateTableName()
}

func (a *A1) updateTableName() *A1 {

	return a
}

func TestA1(t *testing.T) {
	a := A1{A2{
		N: 12,
	}}
	a1 := a.Table()
	fmt.Println(a)
	fmt.Println(a1)
}

func TestInterface(t *testing.T) {
	a := A{}
	fmt.Printf("%p\n", &a)
	ref := reflect.TypeOf(a)
	abref := reflect.New(ref)
	ab := abref.Interface()
	c := ab.(*A)
	c.D = "dwe"

	fmt.Println(abref.Interface())
	fmt.Println(ab)

	fmt.Printf("%p\n", ab)

	f, err := keyreflect.New("user.GGameNameMapInfo")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%T", f)

}

// 定义一个接口
type Initializable interface {
	Init()
}

// 定义一个结构体
type MyStruct struct {
	Name string
}

// 实现接口
func (m *MyStruct) Init() {
	m.Name = "Initialized"
}

// 泛型函数，使用约束
func NewInstance[T Initializable]() T {
	var instance = new(T)
	(*instance).Init() // 调用接口方法
	return *instance
}

func TestNewInstance(t *testing.T) {
	// 使用泛型函数创建实例
	instance := NewInstance[*MyStruct]()
	fmt.Println(instance.Name) // 输出：Initialized
}
