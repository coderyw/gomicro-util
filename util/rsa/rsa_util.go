/**
 * @Author: yinwei
 * @Description:
 * @File: rsa_util
 * @Version: 1.0.0
 * @Date: 2023/10/25 22:09
 */
package rsa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"github.com/coderyw/gomicro-util/util"
)

const (
	// rsa_key_size RSA 密钥位数。512 位早已可被分解，至少应为 2048 位以保证安全。
	// 仅影响 GenRsaKey 新生成的密钥，不影响使用外部既有密钥的加解密。
	rsa_key_size = 2048
)

// EncryptBytesByPub 公钥加密
func EncryptBytesByPub(pubStr string, origData []byte) ([]byte, error) {
	//db, err := base64.StdEncoding.DecodeString(pubStr)
	//if err != nil {
	//	return nil, err
	//}
	block, _ := pem.Decode(util.Str2Bytes(pubStr)) //将密钥解析成公钥实例
	if block == nil {
		return nil, errors.New("public key error")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes) //解析pem.Decode（）返回的Block指针实例
	if err != nil {
		return nil, err
	}
	pub := pubInterface.(*rsa.PublicKey)
	return rsa.EncryptPKCS1v15(rand.Reader, pub, origData) //RSA算法加密
}

func EncryptStrByPub(pubStr string, origData []byte) (string, error) {
	//db, err := base64.StdEncoding.DecodeString(pubStr)
	//if err != nil {
	//	return nil, err
	//}
	block, _ := pem.Decode(util.Str2Bytes(pubStr)) //将密钥解析成公钥实例
	if block == nil {
		return "", errors.New("public key error")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes) //解析pem.Decode（）返回的Block指针实例
	if err != nil {
		return "", err
	}
	pub := pubInterface.(*rsa.PublicKey)
	data, err := rsa.EncryptPKCS1v15(rand.Reader, pub, origData) //RSA算法加密
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DecryptBytesByPri 私钥解密
// ciphertext 必须要由base64解码
func DecryptBytesByPri(private string, ciphertext []byte) ([]byte, error) {
	block, _ := pem.Decode(util.Str2Bytes(private)) //将密钥解析成私钥实例
	if block == nil {
		return nil, errors.New("private key error")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes) //解析pem.Decode（）返回的Block指针实例
	if err != nil {
		return nil, err
	}
	return rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext) //RSA算法解密
}

// DecryptStrByPri 私钥解密
// ciphertext 必须要由base64解码
func DecryptStrByPri(private string, ciphertext string) ([]byte, error) {
	block, _ := pem.Decode(util.Str2Bytes(private)) //将密钥解析成私钥实例
	if block == nil {
		return nil, errors.New("private key error")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes) //解析pem.Decode（）返回的Block指针实例
	if err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}
	return rsa.DecryptPKCS1v15(rand.Reader, priv, data) //RSA算法解密
}

// GenRsaKey 生成rsa公私钥
func GenRsaKey() (privateData, publicData []byte, err error) {
	pk, err := rsa.GenerateKey(rand.Reader, rsa_key_size)
	if err != nil {
		return
	}
	derStream := x509.MarshalPKCS1PrivateKey(pk)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	privateData = pem.EncodeToMemory(block)
	publicKey := &pk.PublicKey
	derPkix, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return
	}
	block = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPkix,
	}
	publicData = pem.EncodeToMemory(block)
	return
}
