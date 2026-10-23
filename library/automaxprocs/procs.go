// Package automaxprocs
// @Author: yinwei
// @File: procs
// @Version: 1.0.0
// @Date: 2025/6/16 13:08

package automaxprocs

import (
	"fmt"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/automaxprocs/maxprocs"
)

func init() {

	opts := []maxprocs.Option{
		maxprocs.Logger(func(s string, i ...interface{}) {
			k := fmt.Sprintf(s, i...)
			log.Info(k)
		}),
		maxprocs.Min(2),
	}
	maxprocs.Set(opts...)
}
