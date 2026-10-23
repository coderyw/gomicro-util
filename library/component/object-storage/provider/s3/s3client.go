package s3

import (
	"context"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	object_storage "github.com/coderyw/gomicro-util/library/component/object-storage"
)

type s3Storage struct {
	client    *s3.Client
	bucket    string
	presigner *s3.PresignClient
}

func newS3(cfg object_storage.Config) (*s3Storage, error) {
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg)

	return &s3Storage{
		client:    client,
		bucket:    cfg.Bucket,
		presigner: s3.NewPresignClient(client),
	}, nil
}

func (s *s3Storage) resolvePutOptions(opts []object_storage.PutOption[*s3.Options]) (*object_storage.PutOptions[*s3.Options], func(*s3.Options)) {
	opt := new(object_storage.PutOptions[*s3.Options])
	for _, v := range opts {
		v(opt)
	}
	var s3Option func(*s3.Options)
	if opt != nil {
		s3Option = func(options *s3.Options) {
			opt.ProviderOption(options)
		}
	} else {
		s3Option = func(options *s3.Options) {
			// do nothing
		}
	}
	return opt, s3Option
}

func (s *s3Storage) Put(ctx context.Context, key string, r io.Reader, opts ...object_storage.PutOption[*s3.Options]) error {
	// option 解析
	opt, s3Option := s.resolvePutOptions(opts)
	return object_storage.Retry(ctx, 3, func() error {
		_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      &s.bucket,
			Key:         &key,
			Body:        r,
			ContentType: &opt.ContentType,
		}, s3Option)
		return mapS3Error(err)
	})
}

func (s *s3Storage) Get(ctx context.Context, key string, opts ...object_storage.PutOption[*s3.Options]) (io.ReadCloser, error) {
	// option 解析（Get 不需要 PutOptions 内容，仅取 s3 层 option）
	_, s3Option := s.resolvePutOptions(opts)
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	}, s3Option)
	if err != nil {
		return nil, mapS3Error(err)
	}
	return out.Body, nil
}

func mapS3Error(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "NotFound") {
		return object_storage.ErrNotFound
	}
	if strings.Contains(err.Error(), "AccessDenied") {
		return object_storage.ErrDenied
	}
	return object_storage.ErrInternal
}
