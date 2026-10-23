package test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	storage2 "github.com/coderyw/gomicro-util/internal/storage"
	s3provider "github.com/coderyw/gomicro-util/internal/storage/s3"
	"github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/storage"
)

func TestRegistry(t *testing.T) {
	registry := storage2.NewRegistry()

	// 实例1：生产 AWS 新加坡
	prodS3, err := s3provider.New("s3_core", config.AwsS3Config{
		Region:                    "ap-southeast-1",
		Endpoint:                  "https://s3.ap-southeast-1.amazonaws.com",
		UseDefaultCredentialChain: false, // 推荐：EKS IRSA / ECS Task Role / EC2 IAM Role
		AccessKeyID:               "",
		SecretAccessKey:           "",
		MaxRetries:                3,
		Buckets: []config.AwsS3BucketConfig{
			{
				IsDefault:     true,
				Alias:         "dev-ok-res",
				Name:          "dev-ok-res",
				PublicBaseURL: "https://dev-ok-res.s3.ap-southeast-1.amazonaws.com",
			},
		},
	})
	if err != nil {
		log.Fatalf("init prod s3 failed: %v", err)
	}

	registry.MustRegister("s3_core", prodS3)

	content := []byte("hello storage\n")

	res, err := registry.Upload(context.Background(), &storage.UploadRequest{
		Instance:       "s3_core",
		Key:            "demo/hello6.txt",
		Body:           bytes.NewReader(content),
		ContentLength:  int64(len(content)),
		ContentType:    "text/plain; charset=utf-8",
		CacheControl:   "public, max-age=300",
		AllowOverwrite: true,
		Timeout:        20 * time.Second,
		Metadata:       map[string]string{"source": "demo", "env": "prod"},
		Tags:           map[string]string{"biz": "common", "module": "upload"},
		// ContentDisposition: `inline; filename="hello5.txt"`,
	})
	if err != nil {
		log.Fatalf("upload failed: %+v", err)
	}

	fmt.Printf("upload success: instance=%s bucket=%s key=%s version=%s url=%s request_id=%s\n",
		res.Instance, res.Bucket, res.Key, res.VersionID, res.ObjectURL, res.RequestID)
}
