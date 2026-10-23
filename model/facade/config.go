// Package facade
// @Author: yinwei
// @File: config
// @Version: 1.0.0
// @Date: 2024/11/4 14:21

package facade

type ConfigValidator interface {
	CfgCheck() error
}
