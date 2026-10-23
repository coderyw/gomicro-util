// Package gzip
// @Author: yinwei
// @File: gzip
// @Version: 1.0.0
// @Date: 2023/11/13 22:42

package gzip

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
)

// 使用gzip压缩结构，返回bytes数据
func EncodeObj2BytesByGzip(data interface{}) ([]byte, error) {
	bs, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize item: %w", err)
	}
	//使用gzip压缩
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	_, err = zw.Write(bs)
	if err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
func EncodeBytes2BytesByGzip(data []byte) ([]byte, error) {
	//使用gzip压缩
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	_, err := zw.Write(data)
	if err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeBytes2BytesByGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err = io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return data, err
}

func DecodeBytes2ObjByGzip(data []byte, obj interface{}) error {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer reader.Close()
	data, err = io.ReadAll(reader)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, obj); err != nil {
		return fmt.Errorf("failed to unmarshal span: %w", err)
	}
	return nil
}
