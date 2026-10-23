package aes

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"testing"
)

// TestEncryptDecryptWithIv 校验带 IV 的 CBC 加解密往返一致。
func TestEncryptDecryptWithIv(t *testing.T) {
	key := []byte("0123456789abcdef") // 16 字节
	iv := []byte("abcdef0123456789")
	plain := []byte("hello ca-common-util 加密测试")

	ciphertext, err := EncryptWithIv(key, iv, plain)
	if err != nil {
		t.Fatalf("EncryptWithIv 失败: %v", err)
	}
	got, err := DecryptWithIv(key, iv, ciphertext)
	if err != nil {
		t.Fatalf("DecryptWithIv 失败: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("往返不一致: got %q want %q", got, plain)
	}
}

// TestDecryptRejectsMalformedLength 校验非法长度密文返回错误而非 panic。
func TestDecryptRejectsMalformedLength(t *testing.T) {
	key := []byte("0123456789abcdef")
	iv := []byte("abcdef0123456789")

	cases := map[string]string{
		"空密文":     base64.StdEncoding.EncodeToString(nil),
		"长度非块整数倍": base64.StdEncoding.EncodeToString([]byte("123")),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecryptWithIv(key, iv, data); err == nil {
				t.Fatal("期望返回错误，实际成功")
			}
		})
	}
}

// TestDecryptRejectsBadIv 校验 IV 长度非法时返回错误而非 panic。
func TestDecryptRejectsBadIv(t *testing.T) {
	key := []byte("0123456789abcdef")
	ciphertext := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0}, aes.BlockSize))
	if _, err := DecryptWithIv(key, []byte("short-iv"), ciphertext); err == nil {
		t.Fatal("非法 IV 长度应返回错误")
	}
}

// TestPkcs7Unpadding 白盒校验去填充的合法与非法分支（确定性）。
func TestPkcs7Unpadding(t *testing.T) {
	const bs = aes.BlockSize

	// 合法：一个满块，填充值为 4
	valid := append([]byte("abcdefghijkl"), 4, 4, 4, 4)
	if got, err := pkcs7Unpadding(valid, bs); err != nil || string(got) != "abcdefghijkl" {
		t.Fatalf("合法填充解析失败: got %q err %v", got, err)
	}

	bad := map[string][]byte{
		"空数据":     {},
		"长度非块整数倍": []byte("123"),
		"填充为0":    append(bytes.Repeat([]byte("a"), bs-1), 0),
		"填充超出块大小": append(bytes.Repeat([]byte("a"), bs-1), byte(bs+1)),
		"填充字节不一致": append(bytes.Repeat([]byte("a"), bs-3), 1, 3, 3), // 尾字节声明填充 3，但前面不是 3
	}
	for name, data := range bad {
		t.Run(name, func(t *testing.T) {
			if _, err := pkcs7Unpadding(data, bs); err == nil {
				t.Fatal("期望返回错误，实际成功")
			}
		})
	}
}

// TestAESEncryptDecryptCBC 校验 key 派生 IV 的 CBC 接口往返一致。
func TestAESEncryptDecryptCBC(t *testing.T) {
	key := "0123456789abcdef"
	plain := []byte("payload")

	enc, err := AESEncryptCBC(key, plain)
	if err != nil {
		t.Fatalf("AESEncryptCBC 失败: %v", err)
	}
	got, err := AESDecryptCBC(key, enc)
	if err != nil {
		t.Fatalf("AESDecryptCBC 失败: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("往返不一致: got %q want %q", got, plain)
	}
}
