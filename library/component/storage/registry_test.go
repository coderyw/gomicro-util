package storage

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/storage"
)

func Init(t *testing.T) {
	_, err := InitObjectStorage(&config.ObjectStorage{
		AwsS3Config: map[string]*config.AwsS3Config{
			"s3_core": &config.AwsS3Config{
				Region:                    "ap-southeast-1",
				Endpoint:                  "http://s3.ap-southeast-1.amazonaws.com",
				UseDefaultCredentialChain: false, // 推荐：EKS IRSA / ECS Task Role / EC2 IAM Role
				AccessKeyID:               "xxxx",
				SecretAccessKey:           "E+oC7i+W/8wwLtzHaSorAoqREBN5/xxx",
				MaxRetries:                3,
				Buckets: []config.AwsS3BucketConfig{
					{
						IsDefault:     true,
						Alias:         "dev-ok-res",
						Name:          "dev-ok-res",
						PublicBaseURL: "http://dev-ok-res.s3-website-ap-xxx-1.amazonaws.com",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestUpload(t *testing.T) {
	Init(t)
	ctx := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok")
	content := []byte("hello storage\n")
	res, err := Upload(ctx, &storage.UploadRequest{
		Instance:       "s3_core",
		Key:            "demo/hello21.txt",
		Body:           bytes.NewReader(content),
		ContentLength:  int64(len(content)),
		ContentType:    "text/plain; charset=utf-8",
		CacheControl:   "public, max-age=300",
		AllowOverwrite: true,
		Timeout:        20 * time.Second,
		Metadata:       map[string]string{"source": "demo", "env": "prod"},
		Tags:           map[string]string{"biz": "common", "module": "upload"},
		//ContentDisposition: `inline; filename="hello5.txt"`,
	})
	if err != nil {
		t.Fatalf("upload failed: %+v", err)
	}

	t.Logf("upload success: instance=%s bucket=%s key=%s version=%s url=%s request_id=%s\n",
		res.Instance, res.Bucket, res.Key, res.VersionID, res.ObjectURL, res.RequestID)
}

func TestUploadFile(t *testing.T) {
	// 打开本地文件
	file, err := os.Open("./upload_test.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	// 确保从文件头开始读取
	if _, err = file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	Init(t)
	ctx := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok")
	res, err := Upload(ctx, &storage.UploadRequest{
		Instance:       "s3_core",
		Key:            "demo/upload_test1.txt",
		Body:           file,
		ContentType:    "text/plain; charset=utf-8",
		AllowOverwrite: true,
		Timeout:        20 * time.Second,
	})
	if err != nil {
		t.Fatalf("upload failed: %+v", err)
	}

	t.Logf("upload success: instance=%s bucket=%s key=%s version=%s url=%s request_id=%s\n",
		res.Instance, res.Bucket, res.Key, res.VersionID, res.ObjectURL, res.RequestID)
}
