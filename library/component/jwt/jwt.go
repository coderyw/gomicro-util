// Package jwt
// @Author: yinwei
// @File: jwt
// @Version: 1.0.0
// @Date: 2025/6/19 10:03

package jwt

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
)

// ParseToken 解析 jwt token
//
// Parameters:
//   - tokenString: string jwttoken 字符串
//   - secret: []byte 密钥
//   - claims: jwt.Claims 要解析出来的 claims
//   - skipValidate:  bool 是否跳过jwt 验证
//
// Returns:
//   - jwt.Claims 返回解析后的 claims
//   - error 错误信息
func ParseToken(tokenString string, secret []byte, claims jwt.Claims, skipValidate bool) (jwt.Claims, error) {
	var op []jwt.ParserOption
	if skipValidate {
		op = append(op, jwt.WithoutClaimsValidation())
	}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// 确保使用 HS256 算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, op...)

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GenerateToken 生成 jwt
//
// Parameters:
//   - claims: jwt.Claims
//   - secret: []byte
//
// Returns:
//   - string
//   - error
func GenerateToken(claims jwt.Claims, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
