// Package md5
// @Author: yinwei
// @File: md5
// @Version: 1.0.0
// @Date: 2024/5/27 17:48

package md5

import (
	"crypto/md5"
	"encoding/hex"
	"io"
)

// 获取byte[]的信息摘要
func BytesMd5(data []byte) string {
	hash := md5.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

// 从reader中读取数据，然后返回信息摘要
func ReaderMd5s(r io.Reader) (string, error) {
	hash := md5.New()
	_, err := io.Copy(hash, r)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
