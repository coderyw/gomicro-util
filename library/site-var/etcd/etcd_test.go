package etcd

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
)

func TestEtcd(t *testing.T) {
	os.Setenv("ETCD_URLS", "106.53.98.119:2379")
	SyncETCDConfig()

	etcdInstance.KeepAlive("keepAlive", "value", 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

type Str1 struct {
	A int         `json:"a"`
	B int         `json:"b"`
	C interface{} `json:"c"`
}

type Ee struct {
	D int `json:"d"`
}

func TestParse(t *testing.T) {
	s := `{"a":1,"b":2,"c":{"d":12}}`
	s1 := new(Str1)
	e1 := new(Ee)
	s1.C = e1

	err := json.Unmarshal([]byte(s), s1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(e1)
}
