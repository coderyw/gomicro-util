// Package jwt
// @Author: yinwei
// @File: jwt_test.go
// @Version: 1.0.0
// @Date: 2025/6/23 16:31

package jwt

import (
	"github.com/golang-jwt/jwt/v5"
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	type args struct {
		claims jwt.Claims
		secret []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "101362013517058054", args: args{
			claims: AuthClaims{
				UserID:         "101362013517058054",
				AppPackageName: "web",
				RegisteredClaims: jwt.RegisteredClaims{
					IssuedAt:  jwt.NewNumericDate(time.Unix(1750648777, 0)),
					NotBefore: jwt.NewNumericDate(time.Unix(1750648777, 0)),
					ExpiresAt: jwt.NewNumericDate(time.Unix(1750735177, 0)),
				},
			},
			secret: []byte("wCk9njsyXJ0t7YkJDFZrsHdK4h9jB6QlIsr3K1Hf4pI="),
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateToken(tt.args.claims, tt.args.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Logf("GenerateToken() got = %v", got)
		})
	}
}

func TestParseToken(t *testing.T) {
	type args struct {
		tokenString string
		secret      []byte
		claims      jwt.Claims
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "过期", args: args{
			tokenString: "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpYXQiOjE3NTA4MTYwMzIsImV4cCI6MTc1MDU2MTQzMiwidXNlcl9pZCI6IjEwNDI0OTg2OTgyMDIzMTY4NiIsImFwcF9wYWNrYWdlX25hbWUiOiIifQ.jGuK8Z8oYNWmvzwZ7pEL3NHs-IPfiyx93B41eMr_OuI",
			secret:      []byte("wCk9njsyXJ0t7YkJDFZrsHdK4h9jB6QlIsr3K1Hf4pI="),
			claims:      &AuthClaims{},
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseToken(tt.args.tokenString, tt.args.secret, tt.args.claims, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Logf("ParseToken() got = %v", got)
		})
	}
}
