// Package micro
// @Author: yinwei
// @File: close
// @Version: 1.0.0
// @Date: 2025/3/5 18:08

package micro

type CloseS struct {
	// 权重，权重值越大，优先级越高
	Weight    int `json:"weight"`
	CloseFunc func()
}
