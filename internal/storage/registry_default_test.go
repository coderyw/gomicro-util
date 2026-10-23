package storage

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/coderyw/gomicro-util/model/storage"
)

// stubUploader 记录最后一次上传由哪个实例承接，用来验证默认实例的分派结果。
// 用桩而不是真 provider：默认实例的解析是纯 registry 逻辑，不该依赖网络。
type stubUploader struct {
	name   string
	called bool
}

func (s *stubUploader) Provider() Provider { return ProviderS3 }

func (s *stubUploader) Upload(_ context.Context, _ *storage.UploadRequest) (*storage.UploadResult, error) {
	s.called = true
	return &storage.UploadResult{Instance: s.name}, nil
}

func (s *stubUploader) Close() {}

// Instance 留空时应回落到默认实例
func TestRegistryUpload_FallsBackToDefaultInstance(t *testing.T) {
	r := NewRegistry()
	primary := &stubUploader{name: "oss_core"}
	other := &stubUploader{name: "oss_backup"}
	r.MustRegister("oss_core", primary)
	r.MustRegister("oss_backup", other)

	if err := r.SetDefault("oss_core"); err != nil {
		t.Fatalf("设默认实例失败: %v", err)
	}

	res, err := r.Upload(context.Background(), &storage.UploadRequest{Key: "demo/a.txt"})
	if err != nil {
		t.Fatalf("留空 Instance 应回落到默认实例: %v", err)
	}
	if res.Instance != "oss_core" {
		t.Errorf("应由默认实例承接, got %q", res.Instance)
	}
	if other.called {
		t.Error("非默认实例不应被调用")
	}
}

// 显式指定的实例名优先级高于默认实例
func TestRegistryUpload_ExplicitInstanceWinsOverDefault(t *testing.T) {
	r := NewRegistry()
	r.MustRegister("oss_core", &stubUploader{name: "oss_core"})
	r.MustRegister("oss_backup", &stubUploader{name: "oss_backup"})
	if err := r.SetDefault("oss_core"); err != nil {
		t.Fatalf("设默认实例失败: %v", err)
	}

	res, err := r.Upload(context.Background(), &storage.UploadRequest{
		Instance: "oss_backup",
		Key:      "demo/a.txt",
	})
	if err != nil {
		t.Fatalf("上传失败: %v", err)
	}
	if res.Instance != "oss_backup" {
		t.Errorf("应由显式指定的实例承接, got %q", res.Instance)
	}
}

// 没有默认实例时，留空 Instance 仍应是明确的参数错误，
// 且错误信息要带上可选实例名——否则排查得回 yaml 里逐段比对
func TestRegistryUpload_NoDefaultKeepsInvalidArgument(t *testing.T) {
	r := NewRegistry()
	r.MustRegister("oss_core", &stubUploader{name: "oss_core"})

	_, err := r.Upload(context.Background(), &storage.UploadRequest{Key: "demo/a.txt"})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("应返回 ErrInvalidArgument, got %v", err)
	}
	if !strings.Contains(err.Error(), "oss_core") {
		t.Errorf("错误信息应列出可选实例, got %v", err)
	}
}

// 默认实例必须已注册：指向不存在实例的默认值会让每次上传都 instance not found，
// 必须在装配阶段就拒绝
func TestRegistrySetDefault_UnregisteredInstance(t *testing.T) {
	r := NewRegistry()
	r.MustRegister("oss_core", &stubUploader{name: "oss_core"})

	if err := r.SetDefault("typo_core"); err == nil {
		t.Fatal("默认实例未注册时应报错")
	}
	if r.Default() != "" {
		t.Errorf("设置失败后不应留下半生效的默认值, got %q", r.Default())
	}
}

// 传空串表示不设默认，是合法输入
func TestRegistrySetDefault_Empty(t *testing.T) {
	r := NewRegistry()
	if err := r.SetDefault(""); err != nil {
		t.Fatalf("空串应表示不设默认: %v", err)
	}
	if r.Default() != "" {
		t.Errorf("默认实例应为空, got %q", r.Default())
	}
}
