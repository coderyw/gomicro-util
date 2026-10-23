package storage

import (
	"context"

	"github.com/coderyw/gomicro-util/model/storage"
)

type Provider string

const (
	ProviderS3        Provider = "s3"
	ProviderHuaweiObs Provider = "huawei_obs"
	// 后续可扩展:
	// ProviderOSS Provider = "oss"
	// ProviderCOS Provider = "cos"
)

type Uploader interface {
	Provider() Provider
	Upload(ctx context.Context, req *storage.UploadRequest) (*storage.UploadResult, error)
	Close()
}
