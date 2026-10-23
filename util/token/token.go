package token

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	env2 "github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/cst"
	"github.com/coderyw/gomicro-util/util/aes"
)

const (
	// todo 使用各自服务的配置
	TokenSignSalt = "V8nbCQNdDyHclbEa"
	AuthCodeKey   = "ra93tm!#dpfN"
	ExpireTime    = 259200 //259200 = 86400*3
)

// md5Sum calculates the MD5 hash of a given string and returns it as a hex string.
func md5Sum(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

// substr returns a substring of the given string, starting at the specified position with the specified length.
func substr(s string, start, length int) string {
	r := []rune(s)
	if start >= len(r) {
		return ""
	}
	end := start + length
	if end > len(r) {
		end = len(r)
	}
	return string(r[start:end])
}

// base64Decode 模拟 PHP 的 base64_decode 行为
func base64Decode(s string) (string, error) {
	// Remove all characters that are not base64 alphabet
	regex := regexp.MustCompile(`[^A-Za-z0-9+/=]`)
	s = regex.ReplaceAllString(s, "")

	// Add padding if necessary
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// authCode performs encryption or decryption based on the operation flag.
// If operation is false, it encrypts the input string.
// If operation is true, it decrypts the input string.
func authCode(input, key string, operation bool, expiry int64) (string, error) {
	ckeyLength := 4
	key = md5Sum(key)
	keya := md5Sum(substr(key, 0, 16))
	keyb := md5Sum(substr(key, 16, 16))
	var keyc string
	if ckeyLength > 0 {
		if operation {
			keyc = substr(input, 0, ckeyLength)
		} else {
			keyc = substr(md5Sum(fmt.Sprintf("%d", time.Now().UnixNano())), 32-ckeyLength, ckeyLength)
		}
	}

	cryptkey := keya + md5Sum(keya+keyc)
	keyLength := len(cryptkey)

	var stringData string
	if operation {
		// 解密操作
		resData := substr(input, ckeyLength, len(input)-ckeyLength)
		decoded, _ := base64Decode(resData)
		stringData = decoded
	} else {
		// 加密操作
		expiryTime := fmt.Sprintf("%010d", expiry+time.Now().Unix())
		hash := substr(md5Sum(input+keyb), 0, 16)
		stringData = expiryTime + hash + input
	}

	stringLength := len(stringData)
	result := make([]byte, stringLength)
	box := make([]int, 256)
	rndkey := make([]int, 256)

	// Initialize box and rndkey
	for i := 0; i <= 255; i++ {
		box[i] = i
		rndkey[i] = int(cryptkey[i%keyLength])
	}

	// Scramble box using rndkey
	j := 0
	for i := 0; i < 256; i++ {
		j = (j + box[i] + rndkey[i]) % 256
		box[i], box[j] = box[j], box[i]
	}

	// Encrypt or decrypt the stringData
	a, j := 0, 0
	for i := 0; i < stringLength; i++ {
		a = (a + 1) % 256
		j = (j + box[a]) % 256
		box[a], box[j] = box[j], box[a]
		k := box[(box[a]+box[j])%256]
		result[i] = byte(int(stringData[i]) ^ k)
	}

	if operation {
		if len(result) < 10 {
			return "", errors.New("invalid token")
		}

		expiryTime, err := strconv.ParseInt(string(result[:10]), 10, 64)
		if err != nil {
			return "", err
		}
		if (expiryTime == 0 || expiryTime > time.Now().Unix()) && substr(md5Sum(string(result[26:])+keyb), 0, 16) == string(result[10:26]) {
			return string(result[26:]), nil
		}
		return "", errors.New("parse token error")
	} else {
		encoded := base64.StdEncoding.EncodeToString(result)
		return keyc + strings.ReplaceAll(encoded, "=", ""), nil
	}
}

func validateToken(authToken string, checkTime bool) (string, bool) {
	arr := strings.Split(authToken, ",")
	if len(arr) == 2 {
		userID := arr[0]
		timestamp, err := strconv.ParseInt(arr[1], 10, 64)
		if err != nil {
			return "", false
		}
		diff := time.Now().Unix() - timestamp
		if diff > ExpireTime && checkTime { // 259200 = 86400*3
			return "", false
		}
		return userID, true
	} else {
		return "", false
	}
}

// ParseToken
// @Description 解析 auth token
// @Author Jimmy 2024-07-09 16:05:00
// @param authToken
// @param checkTime是否校验时间
// @return userId 用户 id
// @return err错误
func ParseToken(authToken string, checkTime bool) (userId string, err error) {
	defer func() {
		if e := recover(); e != nil {
			log.PrintPanic(e)
			err = errors.New("parse token failed")
			return
		}
	}()
	// 判断环境
	env := env2.GetEnv()
	// 预发环境,测试环境，开发环境,路演环境进行处理
	if env == cst.RUNTIME_ENV_PRE || env == cst.RUNTIME_ENV_TEST || env == cst.RUNTIME_ENV_DEV || env == cst.RUNTIME_ENV_ROADSHOW {
		arrAll := strings.Split(authToken, ":suffix:")
		if len(arrAll) > 0 {
			authToken = arrAll[0]
		}
	}

	decodeString, err := base64Decode(authToken)
	if err != nil {
		return "", err
	}
	// 预发环境,测试环境，开发环境,路演环境进行处理
	if env == cst.RUNTIME_ENV_PRE || env == cst.RUNTIME_ENV_TEST || env == cst.RUNTIME_ENV_DEV || env == cst.RUNTIME_ENV_ROADSHOW {
		arrAll := strings.Split(decodeString, ":suffix:")
		if len(arrAll) > 0 {
			decodeString = arrAll[0]
		}
	}

	decoded, err := url.QueryUnescape(decodeString)
	if err != nil {
		return "", err
	}

	encrypted, err := authCode(decoded, AuthCodeKey, true, 0)
	if err != nil {
		return "", err
	}
	userId, bo := validateToken(encrypted, checkTime)
	if !bo {
		return "", fmt.Errorf("token 过期")
	}
	return userId, nil
}

// ParseTokenNum
//
//	@Description: 解析token userId
//	@Author yw 2024-07-15 15:00:43
//	@param authToken
//	@param checkTime
//	@return userId
//	@return err
func ParseTokenNum(authToken string, checkTime bool) (userId int64, err error) {
	id, err := ParseToken(authToken, checkTime)
	if err != nil {
		return 0, nil
	}
	return strconv.ParseInt(id, 10, 64)
}

// GenerateSign 生成token签名
func GenerateSign(userID string, timestamp string, port string) string {
	// 拼接字符串
	data := userID + timestamp + port + TokenSignSalt
	// 生成 MD5 哈希
	hash := md5.Sum([]byte(data))
	// 返回十六进制表示的签名
	return hex.EncodeToString(hash[:])
}

// CheckToken 验证token有效性,checkTime是否过期验证 true-是 false-否
func CheckToken(authToken string, checkTime bool) (int64, error) {
	// 切割 token 字符串, token 格式是 "userID:timestamp:port:sign"
	parts := strings.Split(authToken, ":")
	if len(parts) < 4 {
		return 0, errors.New("token error") // token 格式不正确
	}

	var (
		userID    = parts[0]
		timestamp = parts[1]
		port      = parts[2]
		sign      = parts[3]
	)

	// 生成签名并进行比较
	expectedSign := GenerateSign(userID, timestamp, port)
	if expectedSign != sign {
		return 0, errors.New("token sign error")
	}

	timestamp2, _ := strconv.Atoi(timestamp)
	diff := time.Now().Unix() - int64(timestamp2)
	if diff > ExpireTime && checkTime { // 259200 = 86400*3
		return 0, errors.New("token expired")
	}

	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		return 0, err
	}

	return int64(userIDInt), nil
}

// MakeEnAesToken 将token加密并返回根据环境包装的token
func MakeEnAesToken(token string, key string, iv string) (string, error) {
	aesToken, err := aes.EncryptWithIvUrl([]byte(key), []byte(iv), []byte(token))
	if err != nil {
		return "", err
	}
	env := env2.GetEnv()
	if env == cst.RUNTIME_ENV_PRE || env == cst.RUNTIME_ENV_TEST || env == cst.RUNTIME_ENV_DEV || env == cst.RUNTIME_ENV_ROADSHOW {
		aesToken = aesToken + getSuffix()
	}
	return aesToken, nil
}

// GetDeAesToken 将token解密并去除环境包装
func GetDeAesToken(token string, key string, iv string) (string, error) {
	env := env2.GetEnv()
	if env == cst.RUNTIME_ENV_PRE || env == cst.RUNTIME_ENV_TEST || env == cst.RUNTIME_ENV_DEV || env == cst.RUNTIME_ENV_ROADSHOW {
		arrAll := strings.Split(token, ":suffix:")
		if len(arrAll) > 0 {
			token = arrAll[0]
		}
	}

	deToken, err := aes.DecryptWithIvUrl([]byte(key), []byte(iv), token)
	if err != nil {
		return "", err
	}
	return string(deToken), nil
}

// getSuffix 获取环境后缀
func getSuffix() string {
	env := env2.GetEnv()
	parts := strings.Split(env, "-")
	if len(parts) > 0 {
		return ":suffix:env-" + strings.ToUpper(parts[len(parts)-1])
	}
	return ""
}
