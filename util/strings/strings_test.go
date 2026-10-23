// Package strings
// @Author: yinwei
// @File: strings_test.go
// @Version: 1.0.0
// @Date: 2024/5/23 14:54

package strings

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func TestRandStr(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RandStr(tt.args.n); got != tt.want {
				t.Errorf("RandStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkRandStr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = RandStr(12)
	}
}

func TestBase(t *testing.T) {

	encoded := "1-ZGMwZjhiYzIxOWJmMTFmMDg0M2RkZTAwOWJhYWMwZDguMjYzLjM5NTY0NTExNDMxNzUwMDAx-ZGMwZjhiYzIxOWJmMTFmMDg0M2RkZTAwOWJhYWMwZDguMjYzLjM5NTY0NTExNDMxNzUwMDAy-0-Y2EtdXNlcg==-ZGMwZmFiM2UxOWJmMTFmMDg0M2RkZTAwOWJhYWMwZDhAMTAuMTEuMTIuMTI4-Q2FzaW5vQmFsYW5jZXJTZXJ2aWNlLkdldEJhbGFuY2VCeVVzZXJJZA==-MTAuMTEuMTIuMTI4Ojk4MDQ=" // "Hello world!" 的 Base64 编码
	arr := strings.Split(encoded, "-")
	for _, v := range arr {
		if v == "1" || v == "0" {
			fmt.Println(v)
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			fmt.Println("解码失败:", err)
			return
		}

		fmt.Println("解码后的字符串:", string(decoded))
	}
}
