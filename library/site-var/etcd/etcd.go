package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/coderyw/gomicro-util/library/env"
	config2 "github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/cst"
	"github.com/coderyw/gomicro-util/util"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"

	"log"
)

const TIMEOUT = 5 * time.Second

var (
	once         = sync.Once{}
	etcdInstance *Etcd
	Release      bool
	NoValueError = fmt.Errorf("no value in etcd")
)

type EtcdConf struct {
	Urls     []string
	UserName string
	Password string
}

func GetDefaultEtcdService() *Etcd {
	if etcdInstance == nil {
		if err := SyncETCDConfig(); err != nil {
			log.Fatal(err.Error())
		}
	}
	return etcdInstance
}

func GetEtcdCfg(key string, data interface{}) (config2.EtcdConfig, error) {
	cfg := config2.DefaulConfig
	cfg.Data = data
	val, err := GetDefaultEtcdService().Get(key)
	if err != nil {
		return cfg, err
	}
	log.Printf("获取etcd配置 %+v\n", val)
	err = json.Unmarshal(util.Str2Bytes(val), &cfg)
	return cfg, err
}

// GetEtcdCfgData 根据传入的key和data解析出具体配置数据
// data必须传入指针类型，否则会报错
func GetEtcdCfgData(key string, data interface{}) error {
	_, err := GetEtcdCfg(key, data)
	if err != nil {
		return errors.WithStack(err)
	}
	return err
}

func GetCfgData(key, serverName string, data interface{}) error {
	key = fmt.Sprintf(key, env.GetEnv(), serverName)
	return GetEtcdCfgData(key, data)
}

type Etcd struct {
	cli *clientv3.Client
}

type EtcdItem struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}

func getRuntimeEnv() {
	s, _ := etcdInstance.Get("/develop/config/runtime/env")
	if s == "release" {
		Release = true
	} else {
		Release = false
	}
}

func SyncETCDConfig(urlStr ...[]string) error {
	var urls []string
	if len(urlStr) > 0 {
		urls = urlStr[0]
	} else {
		urls = strings.Split(os.Getenv(cst.ENV_ETCD_URLS), ",")
	}
	if len(urls) == 0 {
		return errors.New("ETCD_URLS 有误")
	}

	//log.Println("etcd地址" + os.Getenv(cst.ENV_ETCD_URLS))

	conf := &EtcdConf{
		Urls: urls,
	}

	cli, err := clientv3.New(clientv3.Config{
		DialTimeout: TIMEOUT,
		Endpoints:   conf.Urls,
		Username:    conf.UserName,
		Password:    conf.Password,
	})
	if err != nil {
		return errors.New(fmt.Sprintf("初始化ETCD有误 - [%s]", err.Error()))
	}
	log.Println("初始化ETCD成功")
	//fmt.Println(conf.Urls)
	etcdInstance = &Etcd{cli: cli}
	getRuntimeEnv() //获取运行时环境
	return nil
}

func (etcd *Etcd) Delete(keyPath string) error {
	kv := clientv3.KV(etcd.cli)
	ctx, _ := context.WithTimeout(context.Background(), TIMEOUT)
	res, err := kv.Delete(ctx, keyPath)
	log.Println("delete etcd value", zap.String("keyPath", keyPath), zap.Any("res", res), zap.Error(err))
	if err != nil {
		return err
	}
	return nil
}

func (etcd *Etcd) Put(keyPath, value string) error {
	kv := clientv3.KV(etcd.cli)
	ctx, _ := context.WithTimeout(context.Background(), TIMEOUT)
	res, err := kv.Put(ctx, keyPath, value)
	log.Println("put etcd value", zap.String("keyPath", keyPath), zap.Any("res", res), zap.Error(err))
	if err != nil {
		return err
	}
	return nil
}

func (etcd *Etcd) Get(keyPath string) (string, error) {
	kv := clientv3.KV(etcd.cli)
	ctx, _ := context.WithTimeout(context.Background(), TIMEOUT)
	res, err := kv.Get(ctx, keyPath)
	if err != nil {
		return "", err
	}

	for _, val := range res.Kvs {
		if string(val.Key[:]) == keyPath {
			return string(val.Value[:]), nil
		}
	}

	return "", NoValueError
}

func (etcd *Etcd) GetBytes(keyPath string) ([]byte, error) {
	kv := clientv3.KV(etcd.cli)
	ctx, _ := context.WithTimeout(context.Background(), TIMEOUT)
	res, err := kv.Get(ctx, keyPath)
	if err != nil {
		return nil, err
	}

	for _, val := range res.Kvs {
		if string(val.Key[:]) == keyPath {
			return val.Value, nil
		}
	}

	return nil, NoValueError
}

func (etcd *Etcd) GetList(keyPath string) ([]*EtcdItem, error) {

	kv := clientv3.KV(etcd.cli)
	ctx, _ := context.WithTimeout(context.Background(), TIMEOUT)
	res, err := kv.Get(ctx, keyPath, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var items []*EtcdItem

	for _, val := range res.Kvs {
		item := new(EtcdItem)
		item.Path = string(val.Key[:])
		item.Value = string(val.Value[:])

		items = append(items, item)
	}

	return items, nil
}

func (etcd *Etcd) GetMap(keyPath string) (map[string]string, error) {
	kv := clientv3.KV(etcd.cli)
	ctx, _ := context.WithTimeout(context.Background(), TIMEOUT)
	res, err := kv.Get(ctx, keyPath, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var items = make(map[string]string, len(res.Kvs))

	for _, val := range res.Kvs {
		items[util.Bytes2Str(val.Key[:])] = util.Bytes2Str(val.Value[:])
	}

	return items, nil
}

func (etcd *Etcd) KeepAlive(key, value string, timeOut int64) {
	//resp,err := etcd.cli.Grant(context.Background(), timeOut)
	//if err!= nil {
	//	log.Error("etcdkeepAlive错误", zap.Error(err))
	//	return
	//}

	go func() {
		for {
			resp, err := etcd.cli.Grant(context.Background(), timeOut)
			if err != nil {
				log.Println("etcdkeepAlive错误", zap.Error(err))
				return
			}
			log.Println("注册租约", zap.String("key", key), zap.String("value", value), zap.Int64("timeOut", timeOut))
			etcd.cli.Put(context.Background(), key, value, clientv3.WithLease(resp.ID))
			ch, err := etcd.cli.KeepAlive(context.Background(), resp.ID)
			if err != nil {
				log.Println("etcdkeepAlive错误", zap.Error(err))
				return
			}
			for {
				ka := <-ch
				if ka == nil {
					break
				}
				log.Println("etcd lease ttl", zap.Any("msg", ka))
			}
		}

	}()
}

// 实验
func (etcd *Etcd) keepAlive(id clientv3.LeaseID, close <-chan struct{}) error {
	ch, err := etcd.cli.KeepAlive(context.Background(), id)
	if err != nil {
		log.Println("etcdkeepAlive错误", zap.Error(err))
		return err
	}
	go func() {
		for {
			select {
			case <-close:
				log.Printf("etcd lease close id=%v\n", id)
				etcd.cli.Revoke(context.Background(), id)
				return
			case resp, ok := <-ch:
				if !ok {
					return
				}
				log.Printf("etcd lease keepalive resp %v\n", zap.Any("resp", resp))
			}
		}
	}()
	return nil
}

// 实验
func (etcd *Etcd) keepAliveOnce(ctx context.Context, id clientv3.LeaseID, t int64, cc <-chan struct{}) {
	c := time.NewTimer(time.Duration(t-2) * time.Second)
	for {
		select {
		case <-c.C:
			resp, err := etcd.cli.KeepAliveOnce(ctx, id)
			if err != nil {
				log.Println("keepAliveOnce error ", err)
			} else {
				log.Println("keepAliveOnce resp ", resp)
			}
		case <-cc:
			return
		}

	}
}

// 分布式锁
// key 分布式锁对应的key
// timeoutArr超市时间
//
//	第一个为持有锁的时间，用来防止长时间持有锁，默认为5s，单位为s
//	第二个为加锁等待时间，当锁已经被其他持有，则会等待这个时间，如果等待时间内没有获取到锁，则会取消等待锁，单位时间为s，
//		如果传0或者不传，则会在锁获取不到时候立即返回
func (etcd *Etcd) Lock(key string, timeoutArr ...int) (func(), error) {
	timeout := 5
	wait := 0
	if len(timeoutArr) > 0 {
		timeout = timeoutArr[0]
	}
	if len(timeoutArr) > 1 {
		wait = timeoutArr[1]
	}
	f, _, err := etcd.lock(key, timeout, wait)
	return f, err
}

// 一直持有锁，如果没有获取到锁，在等待tryTimeout后返回错误
// 返回cancel函数，执行函数取消对锁持有
func (etcd *Etcd) LockAlways(key string, tryTimeout int) (func(), error) {
	f, id, err := etcd.lock(key, 5, tryTimeout) //加锁
	if err != nil {
		return nil, err
	}
	ch, err := etcd.cli.KeepAlive(context.Background(), id) //对这个锁对应的id keepalive
	if err != nil {
		f()
		return nil, err
	}
	go func() {
		for {
			select {
			case _, ok := <-ch:
				if !ok {
					log.Println("keepalive resp chan closed")
					return
				}
				//log.Printf("etcd lease keepalive resp %v\n", zap.Any("resp", resp))
			}
		}
	}()
	return f, nil
}

func (etcd *Etcd) lock(key string, timeout, wait int) (func(), clientv3.LeaseID, error) {
	session, e := concurrency.NewSession(etcd.cli, concurrency.WithTTL(timeout))
	////# 生成一个超时的上下文
	//timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	////# 获取租约
	//response, e := etcd.cli.Grant(timeoutCtx, int64(timeout))
	//if e != nil {
	//	return nil, e
	//}
	////# 通过租约创建session
	//session, e := concurrency.NewSession(etcd.cli, concurrency.WithLease(response.ID))
	//if e != nil {
	//	return nil, e
	//}
	//# 通过session和锁前缀
	mutex := concurrency.NewMutex(session, fmt.Sprintf("/lock/%v", key))
	var waitCancel context.CancelFunc

	if wait == 0 {
		e = mutex.TryLock(context.Background())
	} else {
		var waitCtx context.Context
		waitCtx, waitCancel = context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
		e = mutex.Lock(waitCtx)
	}

	if e != nil {
		return nil, 0, e
	}
	return func() {
		mutex.Unlock(context.Background()) //释放锁
		if waitCancel != nil {
			waitCancel() //释放
		}
		session.Close() //关闭session
	}, session.Lease(), nil
}

// 监控 grpc host修改
func (etcd *Etcd) WatchGrpcHostModify() clientv3.WatchChan {
	grpcConfPath := fmt.Sprintf("/micro/registry/")
	return etcd.cli.Watch(context.Background(), grpcConfPath, clientv3.WithPrefix())
}

func (etcd *Etcd) GetClient() *clientv3.Client {
	return etcd.cli
}

func l() {
	se, _ := concurrency.NewSession(etcdInstance.cli)
	fmt.Println(se)
	//concurrency.NewMutex(etcdInstance.cli, "/se")
}
