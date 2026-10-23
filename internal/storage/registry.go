package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/coderyw/gomicro-util/model/storage"
)

type Registry struct {
	mu        sync.RWMutex
	uploaders map[string]Uploader // key = instance
	// defaultInstance 为 UploadRequest.Instance 留空时使用的实例名，未配时为空串。
	//
	// 与 uploaders 共用一把锁：装配阶段（SetDefault）之后本字段其实只读，
	// 但 Registry 是可导出类型，无法保证调用方一定在发布前写完，
	// 加锁的代价可以忽略，换掉一个 -race 才能发现的隐患是划算的。
	defaultInstance string
}

func NewRegistry() *Registry {
	return &Registry{
		uploaders: make(map[string]Uploader),
	}
}

func (r *Registry) Register(instance string, uploader Uploader) error {
	if instance == "" {
		return fmt.Errorf("instance is empty")
	}
	if uploader == nil {
		return fmt.Errorf("uploader is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.uploaders[instance]; ok {
		return fmt.Errorf("instance already registered: %s", instance)
	}
	r.uploaders[instance] = uploader
	return nil
}

func (r *Registry) MustRegister(instance string, uploader Uploader) {
	if err := r.Register(instance, uploader); err != nil {
		panic(err)
	}
}

func (r *Registry) Get(instance string) (Uploader, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.uploaders[instance]
	return u, ok
}

// SetDefault 设置默认实例，instance 必须已经注册过；传空串表示不设默认。
func (r *Registry) SetDefault(instance string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if instance != "" {
		if _, ok := r.uploaders[instance]; !ok {
			return fmt.Errorf("default instance not registered: %s", instance)
		}
	}
	r.defaultInstance = instance
	return nil
}

// Default 返回默认实例名，未设置时返回空串。
func (r *Registry) Default() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaultInstance
}

// Instances 返回所有已注册的实例名，顺序不保证稳定。
func (r *Registry) Instances() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.uploaders))
	for name := range r.uploaders {
		names = append(names, name)
	}
	return names
}

func (r *Registry) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, uploader := range r.uploaders {
		uploader.Close()
	}
}
func (r *Registry) Upload(ctx context.Context, req *storage.UploadRequest) (*storage.UploadResult, error) {
	if req == nil {
		return nil, ErrInvalidArgument
	}
	// 实例名留空时回落到默认实例（storage.object_storage.default_instance）。
	// 不改写 req：调用方可能复用同一个请求对象，就地写回会让日志里的 Instance 与传入值不符。
	instance := req.Instance
	if instance == "" {
		instance = r.Default()
	}
	if instance == "" {
		return nil, fmt.Errorf("%w: instance is empty and storage.object_storage.default_instance is not set, available: %v",
			ErrInvalidArgument, r.Instances())
	}
	uploader, ok := r.Get(instance)
	if !ok {
		return nil, &OpError{
			Op:       "upload",
			Instance: instance,
			Bucket:   req.Bucket,
			Key:      req.Key,
			Err:      ErrInstanceNotFound,
		}
	}
	return uploader.Upload(ctx, req)
}
