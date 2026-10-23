package object_storage

import (
	"context"
	"io"
)

type ObjectStorage[T any] interface {
	Put(ctx context.Context, key string, r io.Reader, opts ...PutOption[T]) error
	//Get(ctx context.Context, key string) (io.ReadCloser, error)
	//Delete(ctx context.Context, key string) error
	//DeleteBatch(ctx context.Context, keys []string) error
	//Exists(ctx context.Context, key string) (bool, error)
	//
	//PresignPut(ctx context.Context, key string, expire time.Duration) (string, error)
	//PresignGet(ctx context.Context, key string, expire time.Duration) (string, error)
	//
	//MultipartUpload(ctx context.Context, key string, r io.Reader) error
}
