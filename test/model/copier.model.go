// Package model
// @Author: yinwei
// @File: copier.model
// @Version: 1.0.0
// @Date: 2024/10/23 16:25

package model

type Source struct {
	A int      `json:"a"`
	B int      `json:"b"`
	C []string `json:"c"`
	D Source1  `json:"d"`
}

type Source1 struct {
	D string `json:"d"`
}

type Target struct {
	A int      `json:"a"`
	B int      `json:"b"`
	C []string `json:"c"`
	D Target1  `json:"d"`
}

type Target1 struct {
	D string `json:"d"`
}
