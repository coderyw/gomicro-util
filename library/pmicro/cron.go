// Package svc
// @Author: yinwei
// @File: cron
// @Version: 1.0.0
// @Date: 2024/8/2 08:59

package pmicro

import (
	cron3 "github.com/coderyw/gomicro-util/library/cron"
)

func (p *Program) initCron() {
	cron3.Init()
	for _, v := range p.Options.CronJob {
		cron3.AddCron(v)
	}
}
