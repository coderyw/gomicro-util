// Package cron
// @Author: yinwei
// @File: robig_cron_test
// @Version: 1.0.0
// @Date: 2025/6/13 14:17

package cron

import (
	"github.com/coderyw/gomicro-util/library/single"
	cron2 "github.com/coderyw/gomicro-util/model/cron"
	"testing"
)

// 测试定时任务
func TestAddCron(t *testing.T) {
	Init()

	AddCron(cron2.CronJob{
		CronSpc: "0 0/1 * * * ?",
		Name:    "test",
		DoJob: func() {
			println("test")
		},
	})
	AddCron(cron2.CronJob{
		CronSpc: "0/2 * * * * ?",
		Name:    "test1",
		DoJob: func() {
			println("test1")
		},
	})

	<-single.OsKillSingle()
}

func TestStop(t *testing.T) {
	Init()

	AddCron(cron2.CronJob{
		CronSpc: "0 0/1 * * * ?",
		Name:    "test",
		DoJob: func() {
			println("test")
		},
	})
	num := 0
	AddCron(cron2.CronJob{
		CronSpc: "0/2 * * * * ?",
		Name:    "test1",
		DoJob: func() {
			println("test1")
			num++
			if num > 10 {
				Stop("test1")
			}
		},
	})

	<-single.OsKillSingle()
}
