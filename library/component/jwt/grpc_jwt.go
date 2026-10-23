package jwt

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap/zapcore"
)

type AuthGrpcClaims struct {
	// 使用者
	User string `json:"user"`
	jwt.RegisteredClaims
}

func (a *AuthGrpcClaims) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("user", a.User)
	enc.AddString("iss", a.Issuer)
	if a.Subject != "" {
		enc.AddString("sub", a.Subject)
	}
	enc.AddArray("aud", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range a.Audience {
			encoder.AppendString(v)
		}
		return nil
	}))
	if a.ExpiresAt != nil {
		enc.AddString("exp", a.ExpiresAt.String())
	}
	if a.NotBefore != nil {
		enc.AddString("nbf", a.NotBefore.String())
	}
	if a.IssuedAt != nil {
		enc.AddString("iat", a.IssuedAt.String())
	}
	if a.ID != "" {
		enc.AddString("jti", a.ID)
	}
	return nil
}

// ParseGrpcJwtToken 解析 jwt token
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
func ParseGrpcJwtToken(tokenString string, secret []byte, claims *AuthGrpcClaims, skipValidate bool) error {
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
		return err
	}
	if !token.Valid {
		return errors.New("invalid token")
	}
	return nil
}
