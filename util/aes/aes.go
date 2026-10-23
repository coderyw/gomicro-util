/**
* @Author: yinwei
* @Description: AES加密解密工具包
* @File: aes
* @Version: 1.0.0
* @Date: 2023/10/27 13:51
 */
package aes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"

	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
)

// EncryptWithIv AES加密，带有偏移量，使用标准base64编码
func EncryptWithIv(key, iv, plaintext []byte) (string, error) {
	// 创建AES加密块
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	// 使用PKCS7填充明文
	paddedPlaintext := pkcs7Padding(plaintext, block.BlockSize())
	// 创建密文存储空间
	ciphertext := make([]byte, len(paddedPlaintext))
	// 创建CBC加密模式
	mode := cipher.NewCBCEncrypter(block, iv)
	// 执行加密
	mode.CryptBlocks(ciphertext, paddedPlaintext)
	// 返回base64编码的密文
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptWithIv AES解密，带有偏移量
func DecryptWithIv(key, iv []byte, data string) ([]byte, error) {
	// 创建AES解密块
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	// 解码base64密文
	decodedCiphertext, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	return decryptCBC(block, iv, decodedCiphertext)
}

// decryptCBC 在校验 IV 与密文长度后执行 CBC 解密并去除 PKCS7 填充。
// 校验前置可避免非法输入触发 cipher.CryptBlocks 的 panic（对外部可控密文尤为重要）。
func decryptCBC(block cipher.Block, iv, ciphertext []byte) ([]byte, error) {
	blockSize := block.BlockSize()
	// IV 长度必须等于块大小，否则 NewCBCDecrypter 会 panic
	if len(iv) != blockSize {
		return nil, errors.New("aes: IV 长度必须等于块大小")
	}
	// 密文长度必须为块大小的非零整数倍，否则 CryptBlocks 会 panic
	if len(ciphertext) == 0 || len(ciphertext)%blockSize != 0 {
		return nil, errors.New("aes: 密文长度非法")
	}
	// 创建解密数据存储空间
	decryptedData := make([]byte, len(ciphertext))
	// 创建CBC解密模式并执行解密
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(decryptedData, ciphertext)
	// 去除PKCS7填充并返回
	return pkcs7Unpadding(decryptedData, blockSize)
}

// 使用PKCS7填充方式对数据进行填充
func pkcs7Padding(data []byte, blockSize int) []byte {
	// 计算需要填充的字节数
	padding := blockSize - (len(data) % blockSize)
	// 创建填充字节
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	// 返回填充后的数据
	return append(data, padText...)
}

// pkcs7Unpadding 对使用 PKCS7 填充方式的数据进行去填充，并完整校验填充合法性。
// 校验可防止空数据越界 panic、非法填充长度导致的负索引 pank，以及密文被篡改。
func pkcs7Unpadding(data []byte, blockSize int) ([]byte, error) {
	length := len(data)
	if length == 0 || length%blockSize != 0 {
		return nil, errors.New("aes: 待去填充数据长度非法")
	}
	// 获取填充长度，必须落在 [1, blockSize] 区间
	unpadding := int(data[length-1])
	if unpadding == 0 || unpadding > blockSize || unpadding > length {
		return nil, errors.New("aes: 填充字节非法")
	}
	// 校验所有填充字节值一致，抵御密文篡改
	for _, b := range data[length-unpadding:] {
		if int(b) != unpadding {
			return nil, errors.New("aes: 填充内容不一致")
		}
	}
	// 返回去除填充的数据
	return data[:length-unpadding], nil
}

// EncryptWithIvUrl AES加密，带有偏移量,base64 URL编码
func EncryptWithIvUrl(key, iv, plaintext []byte) (string, error) {
	// 创建AES加密块
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	// 使用PKCS7填充明文
	paddedPlaintext := pkcs7Padding(plaintext, block.BlockSize())
	// 创建密文存储空间
	ciphertext := make([]byte, len(paddedPlaintext))
	// 创建CBC加密模式
	mode := cipher.NewCBCEncrypter(block, iv)
	// 执行加密
	mode.CryptBlocks(ciphertext, paddedPlaintext)
	// 返回base64 URL编码的密文
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// DecryptWithIvUrl AES解密，带有偏移量,base64 URL编码
func DecryptWithIvUrl(key, iv []byte, data string) ([]byte, error) {
	// 创建AES解密块
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	// 解码base64 URL密文
	decodedCiphertext, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	return decryptCBC(block, iv, decodedCiphertext)
}

// pkcs5Padding 使用PKCS5填充方式对数据进行填充
func pkcs5Padding(ciphertext []byte, blockSize int) []byte {
	// 计算需要填充的字节数
	padding := blockSize - len(ciphertext)%blockSize
	// 创建填充字节
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	// 返回填充后的数据
	return append(ciphertext, padText...)
}

// AESEncryptCBC 使用AES CBC模式加密数据
//
// ⚠️ 安全提示：本函数使用密钥前 blockSize 字节作为固定 IV，相同明文会产出相同密文，
// 存在被统计分析的风险。仅用于兼容既有加密数据；新场景请使用 EncryptWithIv 并传入随机 IV。
func AESEncryptCBC(key string, origData []byte) ([]byte, error) {
	// 创建AES加密块
	aesBlock, err := aes.NewCipher([]byte(key))
	if err != nil {
		log.Error("Encrypt错误", zap.Error(err))
		return nil, err
	}
	// 获取块大小
	blockSize := aesBlock.BlockSize()
	// 对原始数据进行PKCS5填充
	origData = pkcs5Padding(origData, blockSize)
	// 创建CBC加密模式，使用密钥的前blockSize字节作为IV
	blockMode := cipher.NewCBCEncrypter(aesBlock, []byte(key)[:blockSize])
	// 创建加密数据存储空间并复制数据
	encrypted := make([]byte, len(origData))
	copy(encrypted, origData)
	// 执行加密
	blockMode.CryptBlocks(encrypted, encrypted)
	return encrypted, nil
}

// AESDecryptCBC 使用AES CBC模式解密数据
func AESDecryptCBC(key string, encrypted []byte) ([]byte, error) {
	// 创建AES解密块
	aesBlock, err := aes.NewCipher([]byte(key))
	if err != nil {
		log.Error("Decrypt错误", zap.Error(err))
		return nil, err
	}
	// 获取块大小
	blockSize := aesBlock.BlockSize()
	// 检查密文长度是否为块大小的整数倍
	if len(encrypted)%blockSize != 0 {
		return nil, errors.New("invalid block size")
	}
	// 创建CBC解密模式，使用密钥的前blockSize字节作为IV
	blockMode := cipher.NewCBCDecrypter(aesBlock, []byte(key)[:blockSize])
	// 创建解密数据存储空间并复制数据
	decrypted := make([]byte, len(encrypted))
	copy(decrypted, encrypted)
	// 执行解密
	blockMode.CryptBlocks(decrypted, decrypted)
	// 去除填充（PKCS5 是 PKCS7 在 8 字节块下的特例，去填充逻辑一致）
	decrypted, err = pkcs7Unpadding(decrypted, blockSize)
	if err != nil {
		log.Error("Decrypt错误", zap.Error(err))
		return nil, err
	}
	return decrypted, nil
}
