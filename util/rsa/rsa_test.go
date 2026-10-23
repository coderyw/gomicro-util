/**
 * @Author: yinwei
 * @Description:
 * @File: rsa_test
 * @Version: 1.0.0
 * @Date: 2023/10/30 15:53
 */
package rsa

import (
	"bytes"
	"testing"
)

// TestGenRsaKey 校验 GenRsaKey 生成的密钥对可用于加解密往返。
func TestGenRsaKey(t *testing.T) {
	pri, pub, err := GenRsaKey()
	if err != nil {
		t.Fatalf("GenRsaKey 失败: %v", err)
	}
	if len(pri) == 0 || len(pub) == 0 {
		t.Fatal("GenRsaKey 返回空密钥")
	}

	data := []byte("path/api/ysw/user/getUserCard1698652453")

	// 公钥加密 → 私钥解密（字节接口）
	cipherBytes, err := EncryptBytesByPub(string(pub), data)
	if err != nil {
		t.Fatalf("EncryptBytesByPub 失败: %v", err)
	}
	plainBytes, err := DecryptBytesByPri(string(pri), cipherBytes)
	if err != nil {
		t.Fatalf("DecryptBytesByPri 失败: %v", err)
	}
	if !bytes.Equal(plainBytes, data) {
		t.Fatalf("字节接口往返不一致: got %q want %q", plainBytes, data)
	}
}

// TestEncryptDecryptStr 校验 base64 字符串加解密接口往返一致。
func TestEncryptDecryptStr(t *testing.T) {
	pri, pub, err := GenRsaKey()
	if err != nil {
		t.Fatalf("GenRsaKey 失败: %v", err)
	}

	data := []byte("api/ysw/user/info1705479734")

	cipherStr, err := EncryptStrByPub(string(pub), data)
	if err != nil {
		t.Fatalf("EncryptStrByPub 失败: %v", err)
	}
	plain, err := DecryptStrByPri(string(pri), cipherStr)
	if err != nil {
		t.Fatalf("DecryptStrByPri 失败: %v", err)
	}
	if !bytes.Equal(plain, data) {
		t.Fatalf("字符串接口往返不一致: got %q want %q", plain, data)
	}
}

// TestDecryptBadKey 校验非法密钥返回错误而非 panic。
func TestDecryptBadKey(t *testing.T) {
	if _, err := EncryptBytesByPub("not-a-key", []byte("x")); err == nil {
		t.Fatal("非法公钥应返回错误")
	}
	if _, err := DecryptBytesByPri("not-a-key", []byte("x")); err == nil {
		t.Fatal("非法私钥应返回错误")
	}
}
