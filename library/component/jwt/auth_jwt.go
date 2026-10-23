// Package jwt
// @Author: yinwei
// @File: auth_jwt
// @Version: 1.0.0
// @Date: 2025/6/19 10:11

package jwt

import (
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"strconv"
	"time"
)

// 自定义声明结构体（payload）
type AuthClaims struct {
	UserID         json.Number `json:"user_id"`
	AppPackageName string      `json:"app_package_name"`
	jwt.RegisteredClaims
}

func NewAuthClaims(userId int64, appPackageName string, expiresIn time.Duration) *AuthClaims {
	return &AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)), // 过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),                // 签发时间
		},
		UserID:         json.Number(strconv.FormatInt(userId, 10)),
		AppPackageName: appPackageName,
	}
}
